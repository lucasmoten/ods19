package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"decipher.com/object-drive-server/ciphertext"

	"decipher.com/object-drive-server/amazon"
	"decipher.com/object-drive-server/autoscale"
	"decipher.com/object-drive-server/services/kafka"
	"decipher.com/object-drive-server/services/zookeeper"
	"decipher.com/object-drive-server/util/testhelpers"

	"github.com/karlseguin/ccache"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/uber-go/zap"
	"github.com/urfave/cli"

	globalconfig "decipher.com/object-drive-server/config"
	configx "decipher.com/object-drive-server/configx"
	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/server"

	"decipher.com/object-drive-server/performance"
	"decipher.com/object-drive-server/services/aac"
)

// Globals
var (
	logger = globalconfig.RootLogger
)

// Services that require network
const (
	S3Service        = "s3"
	AACService       = "aac"
	DatabaseService  = "db"
	ZookeeperService = "zk"
)

func main() {

	cliParser := cli.NewApp()
	cliParser.Name = "odrive"
	cliParser.Usage = "object-drive-server binary"
	cliParser.Version = "1.0"

	cliParser.Commands = []cli.Command{
		{
			Name:  "env",
			Usage: "Print all environment variables",
			Action: func(ctx *cli.Context) error {
				configx.PrintODEnvironment()
				return nil
			},
		},
		{
			Name:  "makeScript",
			Usage: "Generate a startup script. Pipe output to a file.",
			Action: func(ctx *cli.Context) error {
				configx.GenerateStartScript()
				return nil
			},
		},
		{
			Name:  "makeEnvScript",
			Usage: "List required env vars in script. Suitable for \"source\". Pipe output to a file.",
			Action: func(ctx *cli.Context) error {
				configx.GenerateSourceEnvScript()
				return nil
			},
		},
		{
			Name:   "testService",
			Usage:  "Run network diagnostic test against a service dependency. Values: s3, aac, db, zk",
			Action: runServiceTest,
		},
	}

	var defaultCiphers cli.StringSlice
	defaultCiphers.Set("TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256")

	cliParser.Flags = []cli.Flag{
		cli.StringSliceFlag{
			Name:  "addCipher",
			Usage: "A Go ciphersuite for TLS configuration. Can be specified multiple times. See: https://golang.org/src/crypto/tls/cipher_suites.go",
			Value: &defaultCiphers,
		},
		cli.BoolTFlag{
			Name:  "useTLS",
			Usage: "Serve content over TLS. Defaults to true.",
		},
		cli.StringSliceFlag{
			Name:  "whitelist",
			Usage: "Whitelisted DNs for impersonation",
		},
		cli.StringFlag{
			Name:  "conf",
			Usage: "Path to yaml configuration file.",
			Value: "odrive.yml",
		},
		cli.StringFlag{
			Name:  "staticRoot",
			Usage: "Path to static files. Defaults to libs/server/static",
			Value: filepath.Join("..", "..", "server", "static"),
		},
		cli.StringFlag{
			Name:  "templateDir",
			Usage: "Path to template files. Defaults to libs/server/static/templates",
			Value: filepath.Join("..", "..", "server", "static", "templates"),
		},
		cli.StringFlag{
			Name:  "tlsMinimumVersion",
			Usage: "Minimum Version of TLS to support (defaults to 1.2, valid values are 1.0, 1.1)",
			Value: "1.2",
		},
	}

	cliParser.Action = func(c *cli.Context) error {

		opts := configx.NewCommandLineOpts(c)
		// TODO move this to main AppConfiguration constructor

		conf := configx.NewAppConfiguration(opts)

		logger.Info("configuration-settings", zap.String("confPath", opts.Conf),
			zap.String("staticRoot", opts.StaticRootPath),
			zap.String("templateDir", opts.TemplateDir),
			zap.String("tlsMinimumVersion", opts.TLSMinimumVersion))

		startApplication(conf)
		return nil
	}

	cliParser.Run(os.Args)
}

func runServiceTest(ctx *cli.Context) error {
	service := ctx.Args().First()
	switch service {
	case S3Service:
		s3Config := configx.NewS3Config()
		if !ciphertext.TestS3Connection(amazon.NewAWSSession(s3Config.AWSConfig, logger)) {
			fmt.Println("Cannot access S3 bucket.")
			os.Exit(1)
		} else {
			fmt.Println("Can read and write bucket referenced by OD_AWS_S3_BUCKET")
			os.Exit(0)
		}
	case AACService:
		fmt.Println("Not implemented for service:", service)
	case DatabaseService:
		fmt.Println("Not implemented for service:", service)
	case ZookeeperService:
		fmt.Println("Not implemented for service:", service)
	default:
		fmt.Println("Unknown service. Please run `odrive help`")
	}
	return nil
}

func startApplication(conf configx.AppConfiguration) {

	app, err := makeServer(conf.ServerSettings)
	if err != nil {
		logger.Error("Error calling makeserver", zap.String("err", err.Error()))
	}

	d, dbID, err := dao.NewDataAccessLayer(conf.DatabaseConnection, dao.WithLogger(logger))
	if err != nil {
		logger.Error("Error configuring DAO.  Check envrionment variable settings for OD_DB_*", zap.String("err", err.Error()))
		os.Exit(1)
	}
	app.RootDAO = d

	//For now, we have one drain provider, just use the default
	zone := ciphertext.S3_DEFAULT_CIPHERTEXT_CACHE
	ciphertext.SetCiphertextCache(zone,
		ciphertext.NewS3CiphertextCache(zone, &conf.CacheSettings, dbID))

	configureEventQueue(app, conf.EventQueue, conf.ZK.Timeout)

	err = connectWithZookeeper(app, conf.ZK.BasepathOdrive, conf.ZK.Address)
	if err != nil {
		logger.Fatal("Could not register with Zookeeper")
	}

	stls := conf.ServerSettings.GetTLSConfig()

	httpServer := &http.Server{
		Addr:           string(app.Addr),
		Handler:        app,
		ReadTimeout:    100000 * time.Second,
		WriteTimeout:   100000 * time.Second,
		MaxHeaderBytes: 1 << 20,
		TLSConfig:      &stls,
	}
	// We go ahead and start the server now, so that when we announce in zk, we are ready.
	exitChan := make(chan error)
	go func() {
		//This blocks until there is an error to stop the server
		exitChan <- httpServer.ListenAndServeTLS(
			conf.ServerSettings.ServerCertChain, conf.ServerSettings.ServerKey)
	}()

	zkTracking(app, conf)

	//Begin cloudwatch stats
	autoscale.CloudWatchReportingStart(app.Tracker)

	logger.Info("starting server", zap.String("addr", app.Addr))

	//When this gets a shutdown signal, it will terminate when all files are uploaded
	//TODO: we will need to watch all existing drain providers to be sure we can safely shut down
	autoscale.WatchForShutdown(app.DefaultZK, logger)

	// Do not announce ephemeral nodes in zk until we have an aac, so that we can service requests immediately
	waitTime := 1
	prevWaitTime := 0
	for app.AAC == nil {
		if waitTime > 10 {
			logger.Error(
				"aac connect is taking too long",
				zap.Int("waitTime in Seconds", waitTime),
			)
		}
		time.Sleep(time.Duration(waitTime) * time.Second)
		waitTime = waitTime + prevWaitTime
		prevWaitTime = waitTime
	}
	// Write our ephemeral node in zk.
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

	// Hang on and don't exit until the listener exits
	err = <-exitChan
	if err != nil {
		logger.Fatal("stopped server", zap.String("err", err.Error()))
	}
}

func zkTracking(app *server.AppServer, conf configx.AppConfiguration) {
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
			return
		}
		// Test our connection after an event hits our queue.
		var err error
		if app.AAC != nil {
			_, err = app.AAC.ValidateAcm(testhelpers.ValidACMUnclassified)
		}
		if app.AAC == nil || err != nil {
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
							app.AAC = aacc
							logger.Info("aac chosen", zap.Object("announcement", announcement))
							// ok... go with this one!
							break
						}
					} else {
						logger.Error("aac reconnect error", zap.String("err", err.Error()))
					}
				}
			}

		}
	}
	// check our AACZK configuration here, and select the correct implementation based on aacConf
	aacZK := app.DefaultZK
	if len(aacConf.ZKAddrs) > 0 {
		logger.Info("connection to custom aac zk", zap.Object("addrs", aacConf.ZKAddrs))
		var err error
		aacZK, err = zookeeper.NewZKState(aacConf.ZKAddrs, int(zkConf.Timeout))
		if err != nil {
			logger.Error("error connecting to custom aac zk", zap.String("err", err.Error()))
		}
	}
	zookeeper.TrackAnnouncement(aacZK, aacConf.AACAnnouncementPoint, aacAnnouncer)

}

// configureEventQueue will set a directly-configured Kafka queue on AppServer, or discover one from ZK.
func configureEventQueue(app *server.AppServer, conf configx.EventQueueConfiguration, zkTimeout int64) {
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
		app.EventQueue, err = kafka.NewAsyncProducer(conf.KafkaAddrs, kafka.WithLogger(logger))
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
		ap, err := kafka.DiscoverKafka(conn, "/brokers/ids", setter, kafka.WithLogger(logger))
		if err != nil {
			logger.Fatal("error discovering kafka from zk", zap.Object("err", err), zap.String("help", help))
		}
		app.EventQueue = ap
		return
	}
	logger.Error("no Kafka queue configured")
}

func connectWithZookeeperTry(app *server.AppServer, zkBasePath, zkAddress string) error {
	// We need the path to our announcements to exist, but not the ephemeral nodes yet
	zkState, err := zookeeper.RegisterApplication(zkBasePath, zkAddress)
	if err != nil {
		return err
	}
	app.DefaultZK = zkState
	// NOTE(cm): We re-assign pointers here to allow all discoverable dependencies to
	// share the same Zookeeper if no custom ZKAddrs are set by other configuration funcs.
	// These pointer assignments will be overwritten if OD_EVENT_ZK_ADDRS or OD_AAC_ZK_ADDRS is set.
	app.EventQueueZK = zkState
	app.AACZK = zkState
	return nil
}

func connectWithZookeeper(app *server.AppServer, zkBasePath, zkAddress string) error {
	err := connectWithZookeeperTry(app, zkBasePath, zkAddress)
	for err != nil {
		sleepInSeconds := 10
		logger.Warn("zk cant register", zap.Int("retry time in seconds", sleepInSeconds))
		time.Sleep(time.Duration(sleepInSeconds) * time.Second)
		err = connectWithZookeeperTry(app, zkBasePath, zkAddress)
	}
	return err
}

func makeServer(conf configx.ServerSettingsConfiguration) (*server.AppServer, error) {

	var templates *template.Template
	var err error

	// If template path specified, ensure templates can be loaded
	if len(conf.PathToTemplateFiles) > 0 {
		templates, err = template.ParseGlob(filepath.Join(conf.PathToTemplateFiles, "*"))
		if err != nil {
			logger.Info("Could not discover templates.")
			return nil, err
		}
	} else {
		templates = nil
	}

	usersLruCache := ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50))
	snippetCache := server.NewSnippetCache()

	httpHandler := server.AppServer{
		Port:                      conf.ListenPort,
		Bind:                      conf.ListenBind,
		Addr:                      conf.ListenBind + ":" + conf.ListenPort,
		Conf:                      conf,
		Tracker:                   performance.NewJobReporters(1024),
		ServicePrefix:             globalconfig.RootURLRegex,
		TemplateCache:             templates,
		StaticDir:                 conf.PathToStaticFiles,
		UsersLruCache:             usersLruCache,
		Snippets:                  snippetCache,
		AclImpersonationWhitelist: conf.AclImpersonationWhitelist,
	}

	httpHandler.InitRegex()

	return &httpHandler, nil
}
