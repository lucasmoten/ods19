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

	"decipher.com/object-drive-server/services/audit"
	"decipher.com/object-drive-server/services/zookeeper"
	"decipher.com/object-drive-server/util/testhelpers"

	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
	"github.com/urfave/cli"

	"decipher.com/object-drive-server/cmd/odrive/libs/config"
	"decipher.com/object-drive-server/cmd/odrive/libs/dao"
	"decipher.com/object-drive-server/cmd/odrive/libs/server"

	globalconfig "decipher.com/object-drive-server/config"

	"decipher.com/object-drive-server/performance"
	"decipher.com/object-drive-server/services/aac"
)

// Globals
var (
	//All loggers are derived from the global one
	logger = globalconfig.RootLogger
	//The location for finding odrive zk nodes
	zkOdrive = globalconfig.GetEnvOrDefault("OD_ZK_ROOT", "/cte") +
		globalconfig.GetEnvOrDefault("OD_ZK_BASEPATH", "/service/object-drive/1.0") + "/https"
	//The location for finding aac zk nodes
	zkAAC = globalconfig.GetEnvOrDefault(
		"OD_ZK_AAC",
		globalconfig.GetEnvOrDefault("OD_ZK_ROOT", "/cte")+"/service/aac/2.2/thrift",
	)
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
				config.PrintODEnvironment()
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
	}

	cliParser.Action = func(c *cli.Context) error {

		ciphers := c.StringSlice("addCipher")
		useTLS := c.BoolT("useTLS")
		// cli lib appends to []string that already contains the "default" value. Must trim
		// the default cipher if addCipher is passed at command line.
		if len(ciphers) > 1 {
			ciphers = ciphers[1:]
		}

		// Load YAML, with optional filename passed
		confPath := c.String("conf")
		confFile, err := config.LoadYAMLConfig(confPath)
		if err != nil {
			fmt.Printf("Error loading yaml configuration at path %s: %v\n", confFile, err)
			os.Exit(1)
		}

		startApplication(confFile.Whitelisted, ciphers, useTLS)
		return nil
	}

	cliParser.Run(os.Args)
}

func runServiceTest(ctx *cli.Context) error {
	service := ctx.Args().First()
	switch service {
	case S3Service:
		sess := server.NewAWSSession()
		if !server.TestS3Connection(sess) {
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
	}
	return nil
}

func startApplication(whitelist, ciphers []string, useTLS bool) {

	globalconfig.SetupGlobalDefaults()

	// Load Configuration from conf.json
	conf := config.NewAppConfiguration(whitelist, ciphers, useTLS)

	app, err := makeServer(conf.ServerSettings)
	if err != nil {
		//Yet we continue when there is an error?
		logger.Error("Error calling makeserver", zap.String("err", err.Error()))
	}

	// put updates onto updates channel
	updates := StateMonitor(app, time.Duration(60*time.Second))

	if false {
		configureAuditor(app, conf.AuditorSettings)
	}

	err = configureDAO(app, conf.DatabaseConnection)
	if err != nil {
		logger.Error("Error configuring DAO.  Check envrionment variable settings for OD_DB_*", zap.String("err", err.Error()))
		os.Exit(1)
	}

	cacheRoot := globalconfig.GetEnvOrDefault("OD_CACHE_ROOT", ".")
	cacheID, err := getDBIdentifier(app)
	if err != nil {
		logger.Error("Database is not fully initialized with a dbstate record", zap.String("err", err.Error()))
		os.Exit(1)
	}

	cachePartition := globalconfig.GetEnvOrDefault("OD_CACHE_PARTITION", "cache") + "/" + cacheID
	configureDrainProvider(app, globalconfig.StandaloneMode, cacheRoot, cachePartition)

	zkAddress := globalconfig.GetEnvOrDefault("OD_ZK_URL", "zk:2181")
	zkBasePath := globalconfig.GetEnvOrDefault("OD_ZK_BASEPATH", "/service/object-drive/1.0")

	//Once we know which cluster we are attached to (ie: the database and bucket partition), note it in the logs
	logger.Info(
		"join cluster",
		zap.String("database", cacheID),
		zap.String("bucket", config.DefaultBucket),
		zap.String("partition", cachePartition),
	)

	//These are the IP:port as seen by the outside.  They are not necessarily the same as the internal port that the server knows,
	//because this is created by the -p $OD_ZK_MYPORT:$OD_SERVER_PORT mapping on docker machine $OD_ZK_MYIP.
	zkMyIP := globalconfig.GetEnvOrDefault("OD_ZK_MYIP", globalconfig.MyIP)
	serverPort := globalconfig.GetEnvOrDefault("OD_SERVER_PORT", "4430")
	zkMyPort := globalconfig.GetEnvOrDefault("OD_ZK_MYPORT", serverPort)

	err = registerWithZookeeper(app, zkBasePath, zkAddress, zkMyIP, zkMyPort)
	if err != nil {
		logger.Fatal("Could not register with Zookeeper")
	}

	app.MasterKey = globalconfig.GetEnvOrDefault("OD_ENCRYPT_MASTERKEY", "otterpaws")
	if app.MasterKey == "otterpaws" {
		logger.Fatal(
			"You should pass in an environment variable 'OD_ENCRYPT_MASTERKEY' to encrypt database keys",
			zap.String("note",
				"Note that if you change masterkey, then the encrypted keys are invalidated",
			),
		)

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

	pollAll(app, updates, time.Duration(30*time.Second))

	zkTracking(app)

	logger.Info("starting server", zap.String("addr", app.Addr))
	//This blocks until there is an error to stop the server
	err =
		httpServer.ListenAndServeTLS(
			conf.ServerSettings.ServerCertChain, conf.ServerSettings.ServerKey)
	if err != nil {
		logger.Fatal("stopped server", zap.String("err", err.Error()))
	}
}

func zkTracking(app *server.AppServer) {
	zookeeper.TrackAnnouncement(app.ZKState, zkOdrive, nil)

	//I am doing this because I need a reference to app to re-assign the connection.
	//The polling scheme isn't doing it. (Out of date, reporting CONNECTED or FAILED when it hasn't actually tried since last report, etc)
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
					aacc, err := aac.GetAACClient(host, port)
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
	zookeeper.TrackAnnouncement(app.ZKState, zkAAC, aacAnnouncer)
}

func configureAuditor(app *server.AppServer, settings config.AuditSvcConfiguration) {

	switch settings.Type {
	case "blackhole":
		app.Auditor = audit.NewBlackHoleAuditor()
	default:
		// TODO return error instead?
		app.Auditor = audit.NewBlackHoleAuditor()
	}

	app.Auditor.Start()
}

func configureDAO(app *server.AppServer, conf config.DatabaseConfiguration) error {
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

func configureDrainProvider(app *server.AppServer, standalone bool, root, cacheID string) {
	var dp server.DrainProvider
	if globalconfig.StandaloneMode {
		logger.Info("Draining cache locally")
		dp = server.NewNullDrainProvider(root, cacheID)
	} else {
		logger.Info("Draining cache to S3")
		dp = server.NewS3DrainProvider(root, cacheID)
	}

	app.DrainProvider = dp
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

func makeServer(conf config.ServerSettingsConfiguration) (*server.AppServer, error) {

	templates, err := template.ParseGlob(
		filepath.Join(globalconfig.ProjectRoot,
			"cmd", "odrive", "libs", "server",
			"static", "templates", "*"))
	if err != nil {
		logger.Info("Cloud not discover templates.")
		return nil, err
	}

	staticPath := filepath.Join(globalconfig.ProjectRoot, "cmd", "odrive", "libs", "server", "static")

	userCache := server.NewUserCache()
	snippetCache := server.NewSnippetCache()

	httpHandler := server.AppServer{
		Port:                      conf.ListenPort,
		Bind:                      conf.ListenBind,
		Addr:                      conf.ListenBind + ":" + conf.ListenPort,
		Tracker:                   performance.NewJobReporters(1024),
		ServicePrefix:             globalconfig.RootURLRegex,
		TemplateCache:             templates,
		StaticDir:                 staticPath,
		Users:                     userCache,
		Snippets:                  snippetCache,
		AclImpersonationWhitelist: conf.AclImpersonationWhitelist,
	}

	httpHandler.InitRegex()

	return &httpHandler, nil
}

func pingDB(conf config.DatabaseConfiguration, db *sqlx.DB) int {
	// But ensure database is up, retrying every 3 seconds for up to 1 minute
	dbPingAttempt := 0
	dbPingSuccess := false
	dbPingAttemptMax := 20
	exitCode := 2
	for dbPingAttempt < dbPingAttemptMax && !dbPingSuccess {
		dbPingAttempt++
		err := db.Ping()

		if err == nil {
			dbPingSuccess = true
			exitCode = 0
		} else {
			elogger := logger.
				With(zap.String("err", err.Error())).
				With(zap.String("host", conf.Host)).
				With(zap.String("port", conf.Host)).
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
				return exitCode
			} else if match, _ := regexp.MatchString(".*connection refused.*", err.Error()); match {
				elogger.Error("Connection refused connecting to database. Database may not yet be online.")
				exitCode = 7
			} else {
				elogger.Error("Unhandled error while connecting to database. Halting")
				exitCode = 1
				return exitCode
			}
		}

		if !dbPingSuccess {
			if dbPingAttempt < dbPingAttemptMax {
				logger.Warn("Retrying in 3 seconds")
				time.Sleep(time.Second * 3)
			} else {
				logger.Error("Maximum retries exhausted. Halting")
				return exitCode
			}
		} else {
			tempDAO := dao.DataAccessLayer{MetadataDB: db}
			_, err := tempDAO.GetDBState()
			if err != nil {
				dbPingSuccess = false
				if err == sql.ErrNoRows || (strings.Contains(err.Error(), "Table") && strings.Contains(err.Error(), "doesn't exist")) {
					logger.Warn("Database connection successful but dbstate not yet set. Retrying in 1 second")
					exitCode = 52
					time.Sleep(time.Second * 1)
				} else {
					elogger := logger.With(zap.String("err", err.Error()))
					elogger.Error("Error calling for dbstate. Halting")
					exitCode = 8
					return exitCode
				}
			} else {
				logger.Info("Database connection successful")
			}
		}
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

func pollAll(app *server.AppServer, updates chan server.ServiceState, updateInterval time.Duration) {
	ticker := time.NewTicker(updateInterval)
	go func() {
		for {
			numPollers := 1
			var wg sync.WaitGroup
			wg.Add(numPollers)
			select {
			case <-ticker.C:
				go pollAAC(app, updates, &wg)
				// Wait for N pollers to return
				wg.Wait()
			}
		}
	}()
}

// pollAAC encapsulates the AAC health check and attempted reconnect.

func pollAAC(app *server.AppServer, updates chan server.ServiceState, wg *sync.WaitGroup) {

	defer wg.Done()

	announcements, err := zookeeper.GetAnnouncements(app.ZKState, zkAAC)
	if err != nil {
		logger.Error(
			"aac poll error",
			zap.String("err", err.Error()),
		)
	} else {
		if aacAnnouncer != nil {
			aacAnnouncer(zkAAC, announcements)
		}
	}
}
