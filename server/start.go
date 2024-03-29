package server

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/auth"
	"bitbucket.di2e.net/dime/object-drive-server/autoscale"
	"bitbucket.di2e.net/dime/object-drive-server/ciphertext"
	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/dao"
	"bitbucket.di2e.net/dime/object-drive-server/services/aac"
	"bitbucket.di2e.net/dime/object-drive-server/services/kafka"
	"bitbucket.di2e.net/dime/object-drive-server/services/zookeeper"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"github.com/samuel/go-zookeeper/zk"
	"go.uber.org/zap"
)

// Globals
var (
	logger     = config.RootLogger
	aacCreated = make(chan *aac.AacServiceClient)
)

// Start starts the server and wires together dependencies.
func Start(conf config.AppConfiguration) error {

	// Block forever until Kafka and ZK come online
	blockForRequiredServices(logger, conf)

	app, err := NewAppServer(conf.ServerSettings)
	if err != nil {
		logger.Error("error constructing app server", zap.Error(err))
		return err
	}

	logger.Info(conf.ServerSettings.EncryptableFunctions.EncryptionStateBanner())

	d, dbID, err := dao.NewDataAccessLayer(conf.DatabaseConnection, dao.WithLogger(logger))
	if err != nil {
		logger.Info("error configuring dao.  check environment variable settings for OD_DB_*", zap.Error(err))
		return err
	}
	if d.ReadOnly {
		if util.ContainsAny(d.SchemaVersion, dao.SchemaVersionsSupported) {
			//if d.SchemaVersion < dao.SchemaVersion {
			logger.Info(fmt.Sprintf("database schema is at version '%s' and dao expects one of '%s'. operating in read only mode until the database is upgraded.", d.SchemaVersion, strings.Join(dao.SchemaVersionsSupported, ",")))
		}
	}
	app.RootDAO = d
	go daoReadOnlyCheck(app, conf.DatabaseConnection)

	zone := ciphertext.S3_DEFAULT_CIPHERTEXT_CACHE
	cache, loggableErr := ciphertext.NewDiskCache(zone, conf.CacheSettings, dbID)
	if loggableErr != nil {
		loggableErr.ToFatal(logger)
	}
	ciphertext.SetCiphertextCache(zone, cache)

	configureEventQueue(app, conf.EventQueue, conf.ZK.Timeout)

	err = connectWithZookeeper(app, conf.ZK.AnnouncementPoint, conf.ZK.Address, conf.ZK.Timeout, conf.ZK.RetryDelay)
	if err != nil {
		logger.Warn("could not register with zookeeper")
	}

	tlsConfig := conf.ServerSettings.GetTLSConfig()

	httpServer := &http.Server{
		Addr:              app.Addr,
		Handler:           app,
		IdleTimeout:       time.Duration(conf.ServerSettings.IdleTimeout) * time.Second,
		ReadTimeout:       time.Duration(conf.ServerSettings.ReadTimeout) * time.Second,
		ReadHeaderTimeout: time.Duration(conf.ServerSettings.ReadHeaderTimeout) * time.Second,
		WriteTimeout:      time.Duration(conf.ServerSettings.WriteTimeout) * time.Second,
		MaxHeaderBytes:    1 << 20,
		TLSConfig:         &tlsConfig,
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
	aacState := <-aacCreated
	app.AAC = aacState
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
		logger.Fatal("could not announce self in zk")
	} else {
		logger.Info(
			"registering odrive AppServer with ZK",
			zap.String("ip", conf.ZK.IP),
			zap.String("port", conf.ZK.Port),
			zap.String("announcementPoint", conf.ZK.AnnouncementPoint),
			zap.String("address", conf.ZK.Address),
		)
	}

	err = <-exitChan
	return err
}

// configureEventQueue will set a directly-configured Kafka queue on AppServer, or discover one from ZK.
func configureEventQueue(app *AppServer, conf config.EventQueueConfiguration, zkTimeout int64) {
	logger.Info("kafka config", zap.Any("conf", conf))

	if len(conf.KafkaAddrs) == 0 && len(conf.ZKAddrs) == 0 {
		// no configuration still provides null implementation
		app.EventQueue = kafka.NewFakeAsyncProducer(logger)
		return
	}

	help := "review OD_EVENT_ZK_ADDRS or OD_EVENT_KAFKA_ADDRS"

	if len(conf.KafkaAddrs) > 0 {
		logger.Info("using direct connect for kafka queue")
		// DIMEODS-1156 - direct connect has no watcher to re-establish if it fails.
		logger.Warn("direct connect is not as durable as discovery via zookeeper. consider using OD_EVENT_ZK_ADDRS instead for durability")
		var err error
		app.EventQueue, err = kafka.NewAsyncProducer(conf.KafkaAddrs, kafka.WithLogger(logger), kafka.WithPublishActions(conf.PublishSuccessActions, conf.PublishFailureActions), kafka.WithTopic(conf.Topic))
		if err != nil {
			logger.Fatal("cannot direct connect to Kafka queue", zap.Error(err), zap.String("help", help))
		}
		return
	}

	if len(conf.ZKAddrs) > 0 {
		logger.Info("attempting to discover kafka queue from zookeeper")
		conn, _, err := zk.Connect(conf.ZKAddrs, time.Duration(zkTimeout)*time.Second)
		if err != nil {
			logger.Fatal("err from zk.Connect", zap.Error(err), zap.String("help", help))
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
		ap, err := kafka.DiscoverKafka(conn, "/brokers/ids", setter, kafka.WithLogger(logger), kafka.WithPublishActions(conf.PublishSuccessActions, conf.PublishFailureActions), kafka.WithTopic(conf.Topic))
		for ap == nil || err != nil {
			logger.Info("kafka was not discovered in zookeeper.", zap.Int("waitTime in seconds", waitTime))
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
			ap, err = kafka.DiscoverKafka(conn, "/brokers/ids", setter, kafka.WithLogger(logger), kafka.WithPublishActions(conf.PublishSuccessActions, conf.PublishFailureActions), kafka.WithTopic(conf.Topic))
		}
		if err != nil {
			logger.Fatal("error discovering kafka from zk", zap.Error(err), zap.String("help", help))
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
	if app.DefaultZK != nil && app.DefaultZK.Conn != nil && zkState != nil && app.DefaultZK.Conn != zkState.Conn {
		app.DefaultZK.Conn.Close()
	}
	app.DefaultZK = zkState
	// These pointer assignments will be overwritten if OD_EVENT_ZK_ADDRS or OD_AAC_ZK_ADDRS is set.
	app.EventQueueZK = zkState
	app.AACZK = zkState
	return nil
}

func connectWithZookeeper(app *AppServer, zkBasePath string, zkAddress string, zkTimeout int64, zkRetryDelay int64) error {
	err := connectWithZookeeperTry(app, zkBasePath, zkAddress, zkTimeout)
	for err != nil {
		sleepInSeconds := int(math.Max(1, math.Min(60, float64(zkRetryDelay))))
		logger.Warn("zk cant register", zap.Int("retry time in seconds", sleepInSeconds))
		time.Sleep(time.Duration(sleepInSeconds) * time.Second)
		err = connectWithZookeeperTry(app, zkBasePath, zkAddress, zkTimeout)
	}
	return err
}

var shutdown = make(chan bool)

func zkKeepalive(app *AppServer, conf config.AppConfiguration) {

	// first run, sleep immediately. Let original ZK code try first.
	warmupTime := int(math.Max(1, math.Min(60, float64(conf.ZK.RetryDelay))))
	time.Sleep(time.Second * time.Duration(warmupTime))

	recheckTime := int(math.Max(1, math.Min(600, float64(conf.ZK.RecheckTime))))
	t := time.NewTicker(time.Duration(time.Second * time.Duration(recheckTime)))

	for {
		select {
		case <-t.C:
			if app.DefaultZK != nil {
				if !app.DefaultZK.IsTerminated {
					logger.Debug("zkKeepalive checking health")
					children, _, err := app.DefaultZK.Conn.Children(conf.ZK.AnnouncementPoint + "/https")
					if err != nil {
						logger.Debug("zkKeepalive health check failure looking for children at our endpoint")
					} else {
						if len(children) > 0 {
							// make sure our ephemeral node exists!
							foundOurself := false
							for _, v := range children {
								if v == config.NodeID {
									foundOurself = true
									break
								}
							}
							if foundOurself {
								logger.Debug("zkKeepalive health check success")
							} else {
								if !app.DefaultZK.IsTerminated {
									logger.Debug("zkKeepalive health check failed to find our node, reannouncing")
									zookeeper.DoReAnnouncements(app.DefaultZK, logger)
								}
							}
						} else {
							if !app.DefaultZK.IsTerminated {
								logger.Debug("zkKeepalive health check failure, no children, including us, at announcement path, reannouncing")
								zookeeper.DoReAnnouncements(app.DefaultZK, logger)
							}
						}
					}
				}
			} else {
				logger.Error("zkKeepalive saw nil pointer to ZK, attempting reconnect")
				connectWithZookeeper(app, conf.ZK.AnnouncementPoint, conf.ZK.Address, conf.ZK.Timeout, conf.ZK.RetryDelay)
			}
		case <-shutdown:
			t.Stop()
			return
		}
	}
}

func aacKeepalive(app *AppServer, conf config.AppConfiguration) {

	// first run, sleep immediately. Let original ZK code try first.
	warmupTime := int(math.Max(1, math.Min(60, float64(conf.AACSettings.WarmupTime))))
	time.Sleep(time.Second * time.Duration(warmupTime))

	recheckTime := int(math.Max(1, math.Min(600, float64(conf.AACSettings.RecheckTime))))
	t := time.NewTicker(time.Duration(time.Second * time.Duration(recheckTime)))

	for {
		select {
		case <-t.C:
			if app.AAC != nil {
				logger.Debug("aacKeepalive checking health")
				aacAuth := auth.NewAACAuth(logger, app.AAC)
				_, _, err := aacAuth.GetFlattenedACM(conf.AACSettings.HealthCheck)
				if err != nil {
					logger.Error("aacKeepalive health check failure", zap.Error(err))
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
		logger.Error("aacReconnect: could not connect to zk addrs", zap.Any("addrs", addrs))
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
	// randomize member order
	shuffleStringSlice(members)
	// look at members and try to connect
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
		commonname := conf.AACSettings.CommonName
		client, err := aac.GetAACClient(host, port, trust, cert, key, commonname)
		if err != nil {
			logger.Error("aacReconnect: error creating aac client with announce data", zap.Any("announceData", info))
			continue
		}
		// we have a client. let's run a test before we set the pointer.
		_, err = client.ValidateAcm(conf.AACSettings.HealthCheck)
		if err != nil {
			logger.Error("aacReconnect: call to ValidateAcm failed", zap.Any("announceData", info))
			continue
		}
		logger.Info("successfully reconnected to aac")
		aacCreated <- client
		return
	}
	// Something is wrong. We will exit, and the polling routine will call us until shutdown.
	logger.Error("aacReconnect: iterated all members of path but found no aac", zap.String("path", path))
}

func shuffleStringSlice(vals []string) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for len(vals) > 0 {
		n := len(vals)
		randIndex := r.Intn(n)
		vals[n-1], vals[randIndex] = vals[randIndex], vals[n-1]
		vals = vals[:n-1]
	}
}

func zkTracking(app *AppServer, conf config.AppConfiguration) {

	go aacKeepalive(app, conf)
	go zkKeepalive(app, conf)

	srvConf, aacConf, zkConf := conf.ServerSettings, conf.AACSettings, conf.ZK

	if strings.ToLower(os.Getenv(config.OD_PEER_ENABLED)) == "true" {
		logger.Info("setting up announcer to check for peers")
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
		zookeeper.TrackAnnouncement(app.DefaultZK, zkConf.AnnouncementPoint+"/https", odriveAnnouncer)
	} else {
		// DIMEODS-1262 - Add additional logging to positively note that we're not peer enabled
		logger.Info("ignoring existence of peers for cipher cache. OD_PEER_ENABLED is set to a value other than true", zap.String("od_peer_enabled", os.Getenv(config.OD_PEER_ENABLED)))
	}

	aacAnnouncer := func(_ string, announcements map[string]zookeeper.AnnounceData) {
		if announcements == nil || len(announcements) == 0 {
			// Responding to this can only remove the last working aac that missed its zk lease,
			// and won't fix anything because it's empty.  so, only respond if there are
			// more than zero announcements.
			logger.Info("aac announcements are empty. skipping")
			return
		}
		// Test our connection after an event hits our queue.
		var err error
		if app.AAC != nil {
			_, err = app.AAC.ValidateAcm(conf.AACSettings.HealthCheck)
		}
		if app.AAC == nil || err != nil {
			if app.AAC == nil {
				logger.Info("aac thrift client is nil and wont be able to service requests. attempting to reconnect")
			}
			if err != nil {
				logger.Info("aac thrift client returned error validating a known good acm. attempting to reconnect", zap.Error(err))
			}
			// If it's broke, then fix it by picking an arbitrary AAC
			for _, announcement := range announcements {

				// One that is alive
				if announcement.Status == "ALIVE" {
					// Try a new host,port
					host := announcement.ServiceEndpoint.Host
					port := announcement.ServiceEndpoint.Port
					aacc, err := aac.GetAACClient(host, port, aacConf.CAPath, aacConf.ClientCert, aacConf.ClientKey, aacConf.CommonName)
					if err == nil {
						_, err = aacc.ValidateAcm(conf.AACSettings.HealthCheck)
						if err != nil {
							logger.Error("aac reconnect check error", zap.Error(err))
						} else {
							aacCreated <- aacc
							logger.Info("aac chosen", zap.Any("announcement", announcement))
							// ok... go with this one!
							break
						}
					} else {
						logger.Error("aac reconnect error", zap.Error(err), zap.Any("announcement", announcement))
					}
				} else {
					logger.Warn("aac announcement skipped as status is not alive", zap.Any("announcement", announcement))
				}
			}

		}
	}
	// check our AACZK configuration here, and select the correct implementation based on aacConf
	aacZK := app.DefaultZK

	if len(aacConf.ZKAddrs) > 0 {
		logger.Info("connection to custom aac zk", zap.Any("addrs", aacConf.ZKAddrs))
		var err error
		aacZK, err = zookeeper.NewZKState(aacConf.ZKAddrs, zkConf.Timeout)
		if err != nil {
			logger.Error("error connecting to custom aac zk", zap.Error(err))
		}
	}
	zookeeper.TrackAnnouncement(aacZK, aacConf.AACAnnouncementPoint, aacAnnouncer)
}

func blockForRequiredServices(l *zap.Logger, conf config.AppConfiguration) {
	// TODO: Pick the right ZK for this check
	l.Info("waiting for zookeeper to come online")
	addrs := strings.Split(conf.ZK.Address, ",")
	zkOnline := zookeeper.IsOnline(addrs)
	<-zkOnline
	l.Info("zookeeper cluster found", zap.String("addrs", conf.ZK.Address))
}

func daoReadOnlyCheck(app *AppServer, dbconf config.DatabaseConfiguration) {
	healthCheckInterval := int(config.GetEnvOrDefaultInt(config.OD_DB_RECHECK_TIME, 30))
	if healthCheckInterval <= 0 {
		logger.Info("db healthcheck disabled as OD_DB_RECHECK_TIME set to <= 0")
		return
	}
	t := time.NewTicker(time.Duration(time.Duration(healthCheckInterval) * time.Second))

	for {
		select {
		case <-t.C:
			curOpenConns := app.RootDAO.GetOpenConnectionCount()
			logger.Debug("db checking health", zap.Int("open-conns", curOpenConns))
			maxOpenConns := int(config.GetEnvOrDefaultInt(config.OD_DB_MAXOPENCONNS, 10))

			if curOpenConns >= maxOpenConns {
				logger.Warn("db connections at peak. consider increasing OD_DB_MAXOPENCONNS", zap.Int("maxOpenConns", maxOpenConns), zap.Int("cur-open-conns", curOpenConns))
				logger.Info("db closing and reopening database")
				err := app.RootDAO.GetDatabase().Close()
				if err != nil {
					logger.Error("db encountered error while closing database", zap.Error(err))
				}
				d, _, err := dao.NewDataAccessLayer(dbconf, dao.WithLogger(logger))
				if err != nil {
					logger.Error("db encountered error while reopening database", zap.Error(err))
				}
				app.RootDAO = d
				logger.Info("db reopened")
			}

			beforeReadOnly := app.RootDAO.IsReadOnly(false)
			// refreshes
			afterReadOnly := app.RootDAO.IsReadOnly(true)
			// Did state change?
			if beforeReadOnly != afterReadOnly {
				if beforeReadOnly {
					logger.Info("dao has entered the writeable state")
				} else {
					logger.Info("dao is read only")
				}
			}
			logger.Debug("db health check success")
		case <-shutdown:
			t.Stop()
			return
		}
	}
}
