package main

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"decipher.com/object-drive-server/ciphertext"

	"decipher.com/object-drive-server/amazon"
	"decipher.com/object-drive-server/autoscale"
	"decipher.com/object-drive-server/services/finder"
	"decipher.com/object-drive-server/services/zookeeper"
	"decipher.com/object-drive-server/util/testhelpers"

	"github.com/jmoiron/sqlx"
	"github.com/karlseguin/ccache"
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
	//All loggers are derived from the global one
	logger = globalconfig.RootLogger
	//The callback that captures the app pointer for repairing aac
	aacAnnouncer func(at string, announcements map[string]zookeeper.AnnounceData)
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

	// put updates onto updates channel
	updates := StateMonitor(app, time.Duration(60*time.Second))

	err = configureDAO(app, conf.DatabaseConnection)
	if err != nil {
		logger.Error("Error configuring DAO.  Check envrionment variable settings for OD_DB_*", zap.String("err", err.Error()))
		os.Exit(1)
	}

	cacheID, err := getDBIdentifier(app)
	if err != nil {
		logger.Error("Database is not fully initialized with a dbstate record", zap.String("err", err.Error()))
		os.Exit(1)
	}

	//For now, we have one drain provider, just use the default
	ciphertext.SetCiphertextCache(
		ciphertext.S3_DEFAULT_CIPHERTEXT_CACHE,
		ciphertext.NewS3CiphertextCache(conf.CacheSettings, conf.CacheSettings.Partition+"/"+cacheID),
	)

	configureEventQueue(app, conf.EventQueue)

	err = registerWithZookeeper(app, conf.ZK.BasepathOdrive, conf.ZK.Address, conf.ZK.IP, conf.ZK.Port)
	if err != nil {
		logger.Fatal("Could not register with Zookeeper")
	}

	httpServer := &http.Server{
		Addr:           string(app.Addr),
		Handler:        app,
		ReadTimeout:    100000 * time.Second, //This breaks big downloads
		WriteTimeout:   100000 * time.Second,
		MaxHeaderBytes: 1 << 20, //This prevents clients from DOS'ing us
	}
	stls := conf.ServerSettings.GetTLSConfig()
	httpServer.TLSConfig = &stls

	pollAll(app, updates, conf, time.Duration(30*time.Second))

	zkTracking(app, conf)

	//Begin cloudwatch stats
	autoscale.CloudWatchReportingStart(app.Tracker)

	logger.Info("starting server", zap.String("addr", app.Addr))

	//When this gets a shutdown signal, it will terminate when all files are uploaded
	//TODO: we will need to watch all existing drain providers to be sure we can safely shut down
	autoscale.WatchForShutdown(app.ZKState, logger)

	//This blocks until there is an error to stop the server
	err = httpServer.ListenAndServeTLS(
		conf.ServerSettings.ServerCertChain, conf.ServerSettings.ServerKey)
	if err != nil {
		logger.Fatal("stopped server", zap.String("err", err.Error()))
	}
}

func zkTracking(app *server.AppServer, appConf configx.AppConfiguration) {
	srvConf := appConf.ServerSettings
	aacSettings := appConf.AACSettings

	//Tell whoever needs to know about the peers - it uses the server cert
	odriveAnnouncer := func(at string, announcements map[string]zookeeper.AnnounceData) {
		//Create a peer list
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
	zookeeper.TrackAnnouncement(app.ZKState, appConf.ZK.BasepathOdrive+"/https", odriveAnnouncer)
	//I am doing this because I need a reference to app to re-assign the connection in the event of failure.
	aacAnnouncer = func(at string, announcements map[string]zookeeper.AnnounceData) {
		if announcements == nil {
			return
		}
		//Something changed.  Smoke test our connection....
		var err error
		if app.AAC != nil {
			_, err = app.AAC.ValidateAcm(testhelpers.ValidACMUnclassified)
		}
		if app.AAC == nil || err != nil {
			//If it's broke, then fix it by picking an arbitrary AAC
			for _, announcement := range announcements {
				//One that is alive
				if announcement.Status == "ALIVE" {
					//Try a new host,port
					host := announcement.ServiceEndpoint.Host
					port := announcement.ServiceEndpoint.Port
					aacc, err := aac.GetAACClient(host, port, aacSettings.CAPath, aacSettings.ClientCert, aacSettings.ClientKey)
					if err == nil {
						_, err = aacc.ValidateAcm(testhelpers.ValidACMUnclassified)
						if err != nil {
							logger.Error("aac reconnect check error", zap.String("err", err.Error()))
						} else {
							app.AAC = aacc
							logger.Info("aac chosen", zap.Object("announcement", announcement))
							//ok... go with this one!
							break
						}
					} else {
						logger.Error("aac reconnect error", zap.String("err", err.Error()))
					}
				}
			}

		}
	}
	//Watch the AAC thrift announcements
	zookeeper.TrackAnnouncement(app.ZKState, aacSettings.AACAnnouncementPoint, aacAnnouncer)
}

func configureDAO(app *server.AppServer, conf configx.DatabaseConfiguration) error {
	db, err := conf.GetDatabaseHandle()
	if err != nil {
		return err
	}
	pingDBresult := pingDB(conf, db)
	if pingDBresult != 0 {
		return errors.New("Could not ping database. Please check connection settings.")
	}
	concreteDAO := dao.DataAccessLayer{MetadataDB: db}
	app.RootDAO = &concreteDAO

	return nil
}

func configureEventQueue(app *server.AppServer, conf configx.EventQueueConfiguration) {
	logger.Info("Kafka Config", zap.Object("conf", conf))
	if len(conf.KafkaAddrs) == 0 {
		app.EventQueue = finder.NewFakeAsyncKafkaProducer(logger)
		return
	}
	app.EventQueue = finder.NewAsyncKafkaProducer(logger, conf.KafkaAddrs, nil)
}

func registerWithZookeeperTry(app *server.AppServer, zkBasePath, zkAddress, myIP, myPort string) error {
	zkState, err := zookeeper.RegisterApplication(zkBasePath, zkAddress)
	if err != nil {
		return err
	}
	err = zookeeper.ServiceAnnouncement(zkState, "https", "ALIVE", myIP, myPort)
	if err != nil {
		return err
	}
	app.ZKState = zkState

	return nil
}

// recovery when zk is completely lost is automatic once we have successfully connected on startup.
// every connected party will remember which ephemeral nodes it is maintaining, and nodes it created,
// so that the zk could not only disappear, but reappear *empty* and everything recovers.
// however, it insists on being able to connect to zk when we startup to register,
// so, just stall until we can talk to a zk.
func registerWithZookeeper(app *server.AppServer, zkBasePath, zkAddress, myIP, myPort string) error {
	logger.Info("registering odrive AppServer with ZK", zap.String("ip", myIP), zap.String("port", myPort),
		zap.String("zkBasePath", zkBasePath), zap.String("zkAddress", zkAddress))
	err := registerWithZookeeperTry(app, zkBasePath, zkAddress, myIP, myPort)
	for err != nil {
		sleepInSeconds := 10
		logger.Warn("zk cant register", zap.Int("retry time in seconds", sleepInSeconds))
		time.Sleep(time.Duration(sleepInSeconds) * time.Second)
		err = registerWithZookeeperTry(app, zkBasePath, zkAddress, myIP, myPort)
	}
	return err
}

func getDBIdentifier(app *server.AppServer) (string, error) {

	if app.RootDAO == nil {
		return "", errors.New("DAO is nil on AppServer")
	}

	dbState, err := app.RootDAO.GetDBState()
	if err != nil {
		return "UNKNOWN", err
	}
	logger.Info("Database version",
		zap.String("schema", dbState.SchemaVersion),
		zap.String("identifier", dbState.Identifier),
	)
	return dbState.Identifier, nil
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
		MasterKey:                 conf.MasterKey,
	}

	httpHandler.InitRegex()

	return &httpHandler, nil
}

func pingDB(conf configx.DatabaseConfiguration, db *sqlx.DB) int {
	// But ensure database is up, retrying every 3 seconds for up to 1 minute
	dbPingAttempt := 0
	dbPingAttemptMax := 20
	exitCode := 2
	var err error
	var schemaErr error

	for dbPingAttempt < dbPingAttemptMax {

		//Prepare for another round
		dbPingAttempt++
		schemaErr = nil
		err = nil
		exitCode = 0
		sleepInSeconds := 3

		//Dont consider anything successful unless we actually saw the schema row
		err = db.Ping()
		if err == nil {
			tempDAO := dao.DataAccessLayer{MetadataDB: db, Logger: logger}
			_, schemaErr = tempDAO.GetDBState()
			if schemaErr == nil {
				//If we succeed, we are done.  Just return 0
				logger.Info("db connected")
				return 0
			}
		}

		//We could not connect to the database
		if err != nil {
			elogger := logger.
				With(zap.String("err", err.Error())).
				With(zap.String("host", conf.Host)).
				With(zap.String("port", conf.Port)).
				With(zap.String("user", conf.Username)).
				With(zap.String("schema", conf.Schema)).
				With(zap.String("CA", conf.CAPath)).
				With(zap.String("Cert", conf.ClientCert))
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				elogger.Error("Timeout connecting to database.")
				exitCode = 28
			} else if match, _ := regexp.MatchString(".*lookup.*", err.Error()); match {
				elogger.Error("Unknown host error connecting to database. Review OD_DB_HOST environment variable configuration. Halting")
				exitCode = 6
				//hard error.  waiting it won't fix it
				return exitCode
			} else if match, _ := regexp.MatchString(".*connection refused.*", err.Error()); match {
				//It's not an error until we time out
				elogger.Info("Connection refused connecting to database. Database may not yet be online.")
				exitCode = 7
			} else {
				//hard error.  waiting won't fix it
				elogger.Error("Unhandled error while connecting to database. Halting")
				exitCode = 1
				return exitCode
			}
		} else {
			// we could connect, but there was an issue with the schema
			if schemaErr == sql.ErrNoRows || (strings.Contains(schemaErr.Error(), "Table") && strings.Contains(schemaErr.Error(), "doesn't exist")) {
				logger.Warn("Database connection successful but dbstate not yet set.")
				exitCode = 52
			} else {
				//hard error.  waiting won't fix it
				elogger := logger.With(zap.String("err", schemaErr.Error()))
				elogger.Error("Error calling for dbstate. Halting")
				exitCode = 8
				return exitCode
			}
		}

		//Sleep in one place
		logger.Info("db sleep for retry", zap.Int64("time in seconds", int64(sleepInSeconds)))
		time.Sleep(time.Duration(sleepInSeconds) * time.Second)
	}
	return exitCode
}

// StateMonitor spawns a goroutine to keep the ServiceRegistry updated, and periodically log
// the contents of ServiceRegistry.
func StateMonitor(app *server.AppServer, updateInterval time.Duration) chan server.ServiceState {
	if app.ServiceRegistry == nil {
		app.ServiceRegistry = make(map[string]server.ServiceState)
	}

	// TODO: instantiate structured logger here
	updates := make(chan server.ServiceState)
	ticker := time.NewTicker(updateInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				reportStates(app.ServiceRegistry)
			case s := <-updates:
				app.ServiceRegistry[s.Name] = s
			}
		}
	}()
	return updates
}

func reportStates(states server.ServiceStates) {
	logger.Debug(
		"Service states",
		zap.Marshaler("states", states),
	)
}

func pollAll(app *server.AppServer, updates chan server.ServiceState, appCfg configx.AppConfiguration, updateInterval time.Duration) {
	ticker := time.NewTicker(updateInterval)
	go func() {
		for {
			numPollers := 1
			var wg sync.WaitGroup
			wg.Add(numPollers)
			select {
			case <-ticker.C:
				go pollAAC(app, updates, appCfg.AACSettings, &wg)
				// Wait for N pollers to return
				wg.Wait()
			}
		}
	}()
}

// pollAAC encapsulates the AAC health check and attempted reconnect.

func pollAAC(app *server.AppServer, updates chan server.ServiceState, aacCfg configx.AACConfiguration, wg *sync.WaitGroup) {

	defer wg.Done()

	announcements, err := zookeeper.GetAnnouncements(app.ZKState, aacCfg.AACAnnouncementPoint)
	if err != nil {
		logger.Error(
			"aac poll error",
			zap.String("err", err.Error()),
		)
	} else {
		if aacAnnouncer != nil {
			aacAnnouncer(aacCfg.AACAnnouncementPoint, announcements)
		}
	}
}
