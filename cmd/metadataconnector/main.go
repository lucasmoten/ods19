package main

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"decipher.com/object-drive-server/services/zookeeper"
	thrift "github.com/samuel/go-thrift/thrift"

	"github.com/jmoiron/sqlx"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/config"
	"decipher.com/object-drive-server/cmd/metadataconnector/libs/dao"
	"decipher.com/object-drive-server/cmd/metadataconnector/libs/server"

	oduconfig "decipher.com/object-drive-server/config"
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

	err = configureDAO(app, conf.DatabaseConnection)
	if err != nil {
		log.Printf("Error configuring DAO. %v\nPlease review connection settings in conf.json\n", err)
		os.Exit(1)
	}

	err = configureAACClient(app)
	if err != nil {
		log.Printf("ERROR: could not connect to AAC: %v\n", err)
	}

	cacheID := schemaCheck(app)
	configureDrainProvider(app, oduconfig.StandaloneMode, cacheID)

	zkAddress := getEnvOrDefault("ZKURL", "zk:2181")
	err = registerWithZookeeper(app, oduconfig.RootURL, zkAddress, oduconfig.MyIP)
	if err != nil {
		log.Fatal("Could not register with Zookeeper")
	}

	app.MasterKey = getEnvOrDefault("masterkey", "otterpaws")
	if app.MasterKey == "otterpaws" {
		log.Printf("You should pass in an environment variable 'masterkey' to encrypt database keys")
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

	log.Println("Starting server on " + app.Addr)
	log.Fatalln(
		httpServer.ListenAndServeTLS(
			conf.ServerSettings.ServerCertChain, conf.ServerSettings.ServerKey))
}

func configureAACClient(app *server.AppServer) error {

	var client *aac.AacServiceClient
	var err error
	attempts := 120
	for {
		log.Printf("Waiting to connect to AAC.")
		client, err = getAACClient()
		if err != nil || client == nil {
			log.Printf("Waiting for AAC:%v", err)
		} else {
			log.Printf("We are connected to AAC")
			break
		}
		attempts--
		if attempts <= 0 {
			break
		}
		time.Sleep(1 * time.Second) //there is a fatal in aac connecting, so must sleep
	}

	app.AAC = client
	return nil

}

// TODO: restart uploader if we lose AAC connection.
func getAACClient() (*aac.AacServiceClient, error) {
	trustPath := filepath.Join(oduconfig.CertsDir, "clients", "client.trust.pem")
	certPath := filepath.Join(oduconfig.CertsDir, "clients", "test_1.cert.pem")
	keyPath := filepath.Join(oduconfig.CertsDir, "clients", "test_1.key.pem")
	conn, err := oduconfig.NewOpenSSLTransport(
		trustPath, certPath, keyPath, "twl-server-generic2", "9093", nil)

	if err != nil {
		log.Printf("cannot create aac client: %v", err)
		return nil, err
	}
	trns := thrift.NewTransport(thrift.NewFramedReadWriteCloser(conn, 0), thrift.BinaryProtocol)
	client := thrift.NewClient(trns, true)
	return &aac.AacServiceClient{Client: client}, nil
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

func configureDrainProvider(app *server.AppServer, standalone bool, cacheID string) {
	var dp server.DrainProvider
	if oduconfig.StandaloneMode {
		log.Printf("Draining cache locally")
		dp = server.NewNullDrainProvider(cacheID)
	} else {
		log.Printf("Draining cache to S3")
		dp = server.NewS3DrainProvider(cacheID)
	}

	app.DrainProvider = dp
}

func registerWithZookeeper(app *server.AppServer, zkBasePath, zkAddress, myIP string) error {

	zkState, err := zookeeper.RegisterApplication(zkBasePath, zkAddress)
	if err != nil {
		return err
	}
	err = zookeeper.ServiceAnnouncement(zkState, "https", "ALIVE", myIP, 4430)
	if err != nil {
		return err
	}

	app.ZKState = zkState

	return nil
}

func schemaCheck(app *server.AppServer) string {

	dbState, err := app.DAO.GetDBState()
	if err != nil {
		log.Printf("Error calling GetDBState(): %v", err)
	} else {
		if dbState.SchemaVersion != dao.SchemaVersion {
			msg := "ERROR: Schema mismatch. '%s' vs '%s'"
			log.Printf(msg, dbState.SchemaVersion, dao.SchemaVersion)
			log.Printf("TODO: A data/schema migration should happen right here")
		}
	}
	log.Printf("Database version %s instance is %s", dbState.SchemaVersion, dbState.Identifier)
	return fmt.Sprintf("cache-%s", dbState.Identifier)
}

func makeServer(conf config.ServerSettingsConfiguration) (*server.AppServer, error) {

	templates, err := template.ParseGlob(
		filepath.Join(oduconfig.ProjectRoot,
			"cmd", "metadataconnector", "libs", "server",
			"static", "templates", "*"))
	if err != nil {
		log.Printf("Cloud not discover templates.")
		return nil, err
	}

	staticPath := filepath.Join(oduconfig.ProjectRoot, "cmd", "metadataconnector", "libs", "server", "static")

	userCache := server.NewUserCache()
	snippetCache := server.NewSnippetCache()

	httpHandler := server.AppServer{
		Port:          conf.ListenPort,
		Bind:          conf.ListenBind,
		Addr:          conf.ListenBind + ":" + strconv.Itoa(conf.ListenPort),
		Tracker:       performance.NewJobReporters(1024),
		ServicePrefix: oduconfig.RootURLRegex,
		TemplateCache: templates,
		StaticDir:     staticPath,
		Users:         userCache,
		Snippets:      snippetCache,
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

func getEnvOrDefault(name, defaultValue string) string {
	envVal := os.Getenv(name)
	if len(envVal) == 0 {
		return defaultValue
	}
	return envVal
}