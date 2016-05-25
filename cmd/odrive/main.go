package main

import (
	"errors"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"decipher.com/object-drive-server/services/audit"
	"decipher.com/object-drive-server/services/zookeeper"
	"decipher.com/object-drive-server/util/testhelpers"

	"github.com/jmoiron/sqlx"

	"decipher.com/object-drive-server/cmd/odrive/libs/config"
	"decipher.com/object-drive-server/cmd/odrive/libs/dao"
	"decipher.com/object-drive-server/cmd/odrive/libs/server"

	globalconfig "decipher.com/object-drive-server/config"

	"decipher.com/object-drive-server/performance"
	"decipher.com/object-drive-server/services/aac"
)

func main() {
	// Load Configuration from conf.json
	conf := config.NewAppConfiguration()

	app, err := makeServer(conf.ServerSettings)
	if err != nil {
		log.Fatalf("Error calling makeServer: %v\n", err)
	}

	// put updates onto updates channel
	updates := StateMonitor(app, time.Duration(10*time.Second))

	if false {
		configureAuditor(app, conf.AuditorSettings)
	}

	err = configureDAO(app, conf.DatabaseConnection)
	if err != nil {
		log.Printf("Error configuring DAO. %v\nPlease review connection settings in conf.json\n", err)
		os.Exit(1)
	}

	cacheRoot := globalconfig.GetEnvOrDefault("OD_CACHE_ROOT", ".")
	cacheID, err := getDBIdentifier(app)
	if err != nil {
		log.Printf("ERROR getting DB IDentifier: %v\n", err)
	}
	cachePartition := globalconfig.GetEnvOrDefault("OD_CACHE_PARTITION", "cache") + "/" + cacheID
	configureDrainProvider(app, globalconfig.StandaloneMode, cacheRoot, cachePartition)

	zkAddress := globalconfig.GetEnvOrDefault("OD_ZK_URL", "zk:2181")
	zkBasePath := globalconfig.GetEnvOrDefault("OD_ZK_BASEPATH", "/service/object-drive/1.0")

	//These are the IP:port as seen by the outside.  They are not necessarily the same as the internal port that the server knows,
	//because this is created by the -p $OD_ZK_MYPORT:$OD_SERVER_PORT mapping on docker machine $OD_ZK_MYIP.
	zkMyIP := globalconfig.GetEnvOrDefault("OD_ZK_MYIP", globalconfig.MyIP)
	serverPort := globalconfig.GetEnvOrDefault("OD_SERVER_PORT", "4430")
	zkMyPort := globalconfig.GetEnvOrDefault("OD_ZK_MYPORT", serverPort)

	err = registerWithZookeeper(app, zkBasePath, zkAddress, zkMyIP, zkMyPort)
	if err != nil {
		log.Fatal("Could not register with Zookeeper")
	}

	app.MasterKey = globalconfig.GetEnvOrDefault("OD_ENCRYPT_MASTERKEY", "otterpaws")
	if app.MasterKey == "otterpaws" {
		log.Printf("You should pass in an environment variable 'OD_ENCRYPT_MASTERKEY' to encrypt database keys")
		log.Printf("Note that if you change masterkey, then the encrypted keys are invalidated")
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

	pollAll(app, updates, time.Duration(3*time.Second))

	log.Println("Starting server on " + app.Addr)
	log.Fatalln(
		httpServer.ListenAndServeTLS(
			conf.ServerSettings.ServerCertChain, conf.ServerSettings.ServerKey))
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

func configureDAO(app *server.AppServer, conf config.DatabaseConnectionConfiguration) error {
	db, err := conf.GetDatabaseHandle()
	if err != nil {
		return err
	}
	pingDBresult := pingDB(db)
	if pingDBresult != 0 {
		return errors.New("Could not ping database. Please check connection settings.")
	}
	concreteDAO := dao.DataAccessLayer{MetadataDB: db}
	app.DAO = &concreteDAO

	return nil
}

func configureDrainProvider(app *server.AppServer, standalone bool, root, cacheID string) {
	var dp server.DrainProvider
	if globalconfig.StandaloneMode {
		log.Printf("Draining cache locally")
		dp = server.NewNullDrainProvider(root, cacheID)
	} else {
		log.Printf("Draining cache to S3")
		dp = server.NewS3DrainProvider(root, cacheID)
	}

	app.DrainProvider = dp
}

func registerWithZookeeper(app *server.AppServer, zkBasePath, zkAddress, myIP, myPort string) error {

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

func getDBIdentifier(app *server.AppServer) (string, error) {

	if app.DAO == nil {
		return "", errors.New("DAO is nil on AppServer")
	}

	dbState, err := app.DAO.GetDBState()
	if err != nil {
		log.Printf("Error calling GetDBState(): %v", err)
		return "UNKNOWN", err
	}
	log.Printf("Database version %s instance is %s", dbState.SchemaVersion, dbState.Identifier)
	return dbState.Identifier, nil
}

func makeServer(conf config.ServerSettingsConfiguration) (*server.AppServer, error) {

	templates, err := template.ParseGlob(
		filepath.Join(globalconfig.ProjectRoot,
			"cmd", "odrive", "libs", "server",
			"static", "templates", "*"))
	if err != nil {
		log.Printf("Cloud not discover templates.")
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

func pingDB(db *sqlx.DB) int {
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
		} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Println("Timeout connecting to database.")
			exitCode = 28
		} else if match, _ := regexp.MatchString(".*lookup.*", err.Error()); match {
			log.Println("Unknown host error connecting to database. Review conf.json configuration. Halting")
			exitCode = 6
			return exitCode
		} else if match, _ := regexp.MatchString(".*connection refused.*", err.Error()); match {
			log.Println("Connection refused connecting to database. Database may not yet be online.")
			exitCode = 7
		} else {
			log.Println("Unhandled error while connecting to database.")
			log.Println(err.Error())
			log.Println("Halting")
			exitCode = 1
			return exitCode
		}
		if !dbPingSuccess {
			if dbPingAttempt < dbPingAttemptMax {
				log.Println("Retrying in 3 seconds")
				time.Sleep(time.Second * 3)
			} else {
				log.Println("Maximum retries exhausted. Halting")
				return exitCode
			}
		} else {
			log.Println("Database connection succesful!")
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

func reportStates(states map[string]server.ServiceState) {
	// TODO nicer formatting
	log.Println("Service states:", states)
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

	if app.ServiceRegistry == nil {
		return
	}

	currentState, ok := app.ServiceRegistry["AAC"]
	if !ok {
		updates <- server.ServiceState{Name: "AAC", Status: "FAILURE", Updated: time.Now()}
		return
	}

	time.Sleep(currentState.Delay(currentState.Retries))

	if currentState.Status == "PERMANENT_FAILURE" {
		// NO-OP for now to try to reconnect forever?
	}

	tryReconnect := false
	if currentState.Status == "FAILURE" {
		tryReconnect = true
	}
	if app.AAC == nil {
		tryReconnect = true
	}
	if app.AAC != nil {
		// We ALWAYS poll if we have a ref to the AAC.
		resp, err := app.AAC.ValidateAcm(testhelpers.ValidACMUnclassified)
		if err != nil {
			tryReconnect = true
		}
		if resp != nil {
			if !resp.Success {
				tryReconnect = true
			}
		}
	}

	if tryReconnect {
		retries := currentState.Retries + 1
		client, err := aac.GetAACClient()
		if err != nil {
			updates <- server.ServiceState{Name: "AAC", Retries: retries, Status: "FAILURE", Updated: time.Now()}
			return
		}

		if client == nil {
			updates <- server.ServiceState{Name: "AAC", Retries: retries, Status: "FAILURE", Updated: time.Now()}
			return
		}

		if client != nil {
			if client.Client != nil {
				resp, err := client.ValidateAcm(testhelpers.ValidACMUnclassified)
				if err != nil {
					updates <- server.ServiceState{Name: "AAC", Retries: retries, Status: "FAILURE", Updated: time.Now()}
					return
				}
				if resp != nil {
					if !resp.Success {
						updates <- server.ServiceState{Name: "AAC", Retries: retries, Status: "FAILURE", Updated: time.Now()}
						return
					}
				}
			}
		} else {
			updates <- server.ServiceState{Name: "AAC", Retries: retries, Status: "FAILURE", Updated: time.Now()}
			return
		}
		// if success, set on app
		app.AAC = client
		updates <- server.ServiceState{Name: "AAC", Status: "CONNECTED", Updated: time.Now()}
		return
	}
}
