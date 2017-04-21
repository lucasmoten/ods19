package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"decipher.com/object-drive-server/autoscale"
	"decipher.com/object-drive-server/ciphertext"
	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/services/aac"
	"decipher.com/object-drive-server/services/kafka"
	"decipher.com/object-drive-server/services/zookeeper"
	"decipher.com/object-drive-server/util/testhelpers"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/uber-go/zap"
)

// Globals
var (
	logger     = config.RootLogger
	aacCreated = make(chan *aac.AacServiceClient)
)

// Start starts the server and wires together dependencies.
func Start(conf config.AppConfiguration) error {

	// Block forever until Kafka and ZK come online
	blockForRequiredServices(conf)

	app, err := NewAppServer(conf.ServerSettings)
	if err != nil {
		logger.Error("error constructing app server", zap.String("err", err.Error()))
		return err
	}

	d, dbID, err := dao.NewDataAccessLayer(conf.DatabaseConnection, dao.WithLogger(logger))
	if err != nil {
		logger.Error("Error configuring DAO.  Check envrionment variable settings for OD_DB_*", zap.String("err", err.Error()))
		return err
	}
	app.RootDAO = d

	zone := ciphertext.S3_DEFAULT_CIPHERTEXT_CACHE
	cache, loggableErr := ciphertext.NewS3CiphertextCache(zone, conf.CacheSettings, dbID)
	if loggableErr != nil {
		loggableErr.ToFatal(logger)
	}
	ciphertext.SetCiphertextCache(zone, cache)

	configureEventQueue(app, conf.EventQueue, conf.ZK.Timeout)

	err = connectWithZookeeper(app, conf.ZK.BasepathOdrive, conf.ZK.Address, conf.ZK.Timeout)
	if err != nil {
		logger.Fatal("Could not register with Zookeeper")
	}

	tlsConfig := conf.ServerSettings.GetTLSConfig()

	httpServer := &http.Server{
		Addr:           app.Addr,
		Handler:        app,
		ReadTimeout:    100000 * time.Second,
		WriteTimeout:   100000 * time.Second,
		MaxHeaderBytes: 1 << 20,
		TLSConfig:      &tlsConfig,
	}
	exitChan := make(chan error)
	go func() {
		exitChan <- httpServer.ListenAndServeTLS(
			conf.ServerSettings.ServerCertChain, conf.ServerSettings.ServerKey)
	}()

	zkTracking(app, conf)
	logger.Info("starting server", zap.String("addr", app.Addr))

	autoscale.CloudWatchReportingStart(app.Tracker)
	autoscale.WatchForShutdown(app.DefaultZK, logger)

	logger.Info("waiting for aac to be created")
	app.AAC = <-aacCreated
	go func() {
		for {
			select {
			case newAAC := <-aacCreated:
				app.AAC = newAAC
			}
		}
	}()

	// Announce our new service in ZK.
	err = zookeeper.ServiceAnnouncement(app.DefaultZK, "https", "ALIVE", conf.ZK.IP, conf.ZK.Port)
	if err != nil {
		logger.Fatal("Could not announce self in zk")
	} else {
		logger.Info(
			"registering odrive AppServer with ZK",
			zap.String("ip", conf.ZK.IP),
			zap.String("port", conf.ZK.Port),
			zap.String("zkBasePath", conf.ZK.BasepathOdrive),
			zap.String("zkAddress", conf.ZK.Address),
		)
	}

	err = <-exitChan
	return err
}

// configureEventQueue will set a directly-configured Kafka queue on AppServer, or discover one from ZK.
func configureEventQueue(app *AppServer, conf config.EventQueueConfiguration, zkTimeout int64) {
	logger.Info("Kafka Config", zap.Object("conf", conf))

	if len(conf.KafkaAddrs) == 0 && len(conf.ZKAddrs) == 0 {
		// no configuration still provides null implementation
		app.EventQueue = kafka.NewFakeAsyncProducer(logger)
		return
	}

	help := "review OD_EVENT_ZK_ADDRS or OD_EVENT_KAFKA_ADDRS"

	if len(conf.KafkaAddrs) > 0 {
		logger.Info("using direct connect for Kafka queue")
		var err error
		app.EventQueue, err = kafka.NewAsyncProducer(conf.KafkaAddrs, kafka.WithLogger(logger), kafka.WithPublishActions(conf.PublishSuccessActions, conf.PublishFailureActions))
		if err != nil {
			logger.Fatal("cannot direct connect to Kakfa queue", zap.Object("err", err), zap.String("help", help))
		}
		return
	}

	if len(conf.ZKAddrs) > 0 {
		logger.Info("attempting to discover Kafka queue from zookeeper")
		conn, _, err := zk.Connect(conf.ZKAddrs, time.Duration(zkTimeout)*time.Second)
		if err != nil {
			logger.Fatal("err from zk.Connect", zap.Object("err", err), zap.String("help", help))
		}
		setter := func(ap *kafka.AsyncProducer) {
			// Don't just reset the conn because a zk event told you to, do an explicit check.
			if app.EventQueue.Reconnect() {
				app.EventQueue = ap
			}
		}
		// Allow time for kafka to be available in zookeeper
		waitTime := 1
		prevWaitTime := 0
		ap, err := kafka.DiscoverKafka(conn, "/brokers/ids", setter, kafka.WithLogger(logger), kafka.WithPublishActions(conf.PublishSuccessActions, conf.PublishFailureActions))
		for ap == nil || err != nil {
			logger.Warn("kafka was not discovered in zookeeper.", zap.Int("waitTime in seconds", waitTime))
			if waitTime > 600 {
				logger.Error(
					"kafka discovery is taking too long",
					zap.Int("waitTime in Seconds", waitTime),
				)
				break
			}
			time.Sleep(time.Duration(waitTime) * time.Second)
			waitTime = waitTime + prevWaitTime
			prevWaitTime = waitTime
			err = nil
			ap, err = kafka.DiscoverKafka(conn, "/brokers/ids", setter, kafka.WithLogger(logger), kafka.WithPublishActions(conf.PublishSuccessActions, conf.PublishFailureActions))
		}
		if err != nil {
			logger.Fatal("error discovering kafka from zk", zap.Object("err", err), zap.String("help", help))
		}
		logger.Info("kafka discovery successful")
		app.EventQueue = ap
		return
	}
	logger.Error("no Kafka queue configured")
}

func connectWithZookeeperTry(app *AppServer, zkBasePath string, zkAddress string, zkTimeout int64) error {
	// We need the path to our announcements to exist, but not the ephemeral nodes yet
	zkState, err := zookeeper.RegisterApplication(zkBasePath, zkAddress, zkTimeout)
	if err != nil {
		return err
	}
	app.DefaultZK = zkState
	// These pointer assignments will be overwritten if OD_EVENT_ZK_ADDRS or OD_AAC_ZK_ADDRS is set.
	app.EventQueueZK = zkState
	app.AACZK = zkState
	return nil
}

func connectWithZookeeper(app *AppServer, zkBasePath string, zkAddress string, zkTimeout int64) error {
	err := connectWithZookeeperTry(app, zkBasePath, zkAddress, zkTimeout)
	for err != nil {
		sleepInSeconds := 10
		logger.Warn("zk cant register", zap.Int("retry time in seconds", sleepInSeconds))
		time.Sleep(time.Duration(sleepInSeconds) * time.Second)
		err = connectWithZookeeperTry(app, zkBasePath, zkAddress, zkTimeout)
	}
	return err
}

var shutdown = make(chan bool)

func aacKeepalive(app *AppServer, conf config.AppConfiguration) {

	// first run, sleep immediately. Let original ZK code try first.
	time.Sleep(time.Second * 20)

	t := time.NewTicker(time.Duration(30 * time.Second))

	for {
		select {
		case <-t.C:
			if app.AAC != nil {
				logger.Debug("aacKeepalive: checking health")
				_, err := app.AAC.ValidateAcm(testhelpers.ValidACMUnclassified)
				if err != nil {
					logger.Error("aacKeepalive health check failure", zap.Object("err", err))
					aacReconnect(app, conf)
				} else {
					logger.Debug("aacKeepalive health check success")
				}
			} else {
				logger.Error("aacKeepalive saw nil pointer to AAC")
				aacReconnect(app, conf)
			}
		case <-shutdown:
			t.Stop()
			return
		}
	}
}

func aacReconnect(app *AppServer, conf config.AppConfiguration) {

	var addrs []string

	if len(conf.AACSettings.ZKAddrs) > 0 {
		addrs = conf.AACSettings.ZKAddrs
	} else {
		addrs = strings.Split(conf.ZK.Address, ",")
	}
	zkState, err := zookeeper.NewZKState(addrs, 10)
	if err != nil {
		logger.Error("aacReconnect: could not connect to zk addrs", zap.Object("addrs", addrs))
		return
	}

	conn := zkState.Conn
	defer conn.Close()
	path := conf.AACSettings.AACAnnouncementPoint
	members, _, err := conn.Children(path)
	if err != nil {
		logger.Error("aacReconnect: error reading zk path", zap.String("path", path))
		return
	}
	if len(members) < 1 {
		logger.Error("aacReconnect: no members of path", zap.String("path", path))
		return
	}
	for _, item := range members {
		memberPath := path + "/" + item
		ad, _, err := conn.Get(memberPath)
		if err != nil {
			logger.Error("aacReconnect: error getting member", zap.String("path", memberPath))
		}
		var info zookeeper.AnnounceData
		err = json.Unmarshal([]byte(ad), &info)
		if err != nil {
			logger.Error("aacReconnect: could not unmarshal aac announcement", zap.String("data", item))
			continue
		}
		host := info.ServiceEndpoint.Host
		port := info.ServiceEndpoint.Port
		trust := conf.AACSettings.CAPath
		cert := conf.AACSettings.ClientCert
		key := conf.AACSettings.ClientKey
		client, err := aac.GetAACClient(host, port, trust, cert, key)
		if err != nil {
			logger.Error("aacReconnect: error creating aac client with announce data", zap.Object("announcData", info))
			continue
		}
		// we have a client. let's run a test before we set the pointer.
		_, err = client.ValidateAcm(testhelpers.ValidACMUnclassified)
		if err != nil {
			logger.Error("aacReconnect: call to ValidateAcm failed", zap.Object("announcData", info))
			continue
		}
		logger.Info("successfully reconnected to aac")
		aacCreated <- client
		return
	}
	// Something is wrong. We will exit, and the polling routine will call us until shutdown.
	logger.Error("aacReconnect: iterated all members of path but found no aac", zap.String("path", path))
}

func zkTracking(app *AppServer, conf config.AppConfiguration) {

	go aacKeepalive(app, conf)

	srvConf, aacConf, zkConf := conf.ServerSettings, conf.AACSettings, conf.ZK

	odriveAnnouncer := func(at string, announcements map[string]zookeeper.AnnounceData) {
		peerMap := make(map[string]*ciphertext.PeerMapData)
		for announcementKey, announcement := range announcements {
			peerMap[announcementKey] = &ciphertext.PeerMapData{
				Host:    announcement.ServiceEndpoint.Host,
				Port:    announcement.ServiceEndpoint.Port,
				CA:      srvConf.CAPath,
				Cert:    srvConf.ServerCertChain,
				CertKey: srvConf.ServerKey,
			}
		}
		ciphertext.ScheduleSetPeers(peerMap)
	}
	zookeeper.TrackAnnouncement(app.DefaultZK, zkConf.BasepathOdrive+"/https", odriveAnnouncer)

	aacAnnouncer := func(_ string, announcements map[string]zookeeper.AnnounceData) {
		if announcements == nil {
			logger.Info("aac announcements are empty. skipping")
			return
		}
		// Test our connection after an event hits our queue.
		var err error
		if app.AAC != nil {
			_, err = app.AAC.ValidateAcm(testhelpers.ValidACMUnclassified)
		}
		if app.AAC == nil || err != nil {
			if app.AAC == nil {
				logger.Info("aac thrift client is nil and wont be able to service requests. attempting to reconnect")
			}
			if err != nil {
				logger.Info("aac thrift client returned error validating a known good acm. attempting to reconnect", zap.String("err", err.Error()))
			}
			// If it's broke, then fix it by picking an arbitrary AAC
			for _, announcement := range announcements {

				// One that is alive
				if announcement.Status == "ALIVE" {
					// Try a new host,port
					host := announcement.ServiceEndpoint.Host
					port := announcement.ServiceEndpoint.Port
					aacc, err := aac.GetAACClient(host, port, aacConf.CAPath, aacConf.ClientCert, aacConf.ClientKey)
					if err == nil {
						_, err = aacc.ValidateAcm(testhelpers.ValidACMUnclassified)
						if err != nil {
							logger.Error("aac reconnect check error", zap.String("err", err.Error()))
						} else {
							aacCreated <- aacc
							logger.Info("aac chosen", zap.Object("announcement", announcement))
							// ok... go with this one!
							break
						}
					} else {
						logger.Error("aac reconnect error", zap.String("err", err.Error()), zap.Object("announcement", announcement))
					}
				} else {
					logger.Warn("aac announcement skipped as status is not alive", zap.Object("announcement", announcement))
				}
			}

		}
	}
	// check our AACZK configuration here, and select the correct implementation based on aacConf
	aacZK := app.DefaultZK

	if len(aacConf.ZKAddrs) > 0 {
		logger.Info("connection to custom aac zk", zap.Object("addrs", aacConf.ZKAddrs))
		var err error
		aacZK, err = zookeeper.NewZKState(aacConf.ZKAddrs, zkConf.Timeout)
		if err != nil {
			logger.Error("error connecting to custom aac zk", zap.String("err", err.Error()))
		}
	}
	zookeeper.TrackAnnouncement(aacZK, aacConf.AACAnnouncementPoint, aacAnnouncer)
}

func blockForRequiredServices(conf config.AppConfiguration) {
	// TODO: Pick the right ZK for this check
	zkOnline := zookeeper.IsOnline(strings.Split(conf.ZK.Address, ","))
	<-zkOnline
}
