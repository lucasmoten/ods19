package main

import (
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"decipher.com/oduploader/services/zookeeper"
	thrift "github.com/samuel/go-thrift/thrift"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/cmd/metadataconnector/libs/server"

	oduconfig "decipher.com/oduploader/config"
	"decipher.com/oduploader/performance"
	aac "decipher.com/oduploader/services/aac"

	_ "net/http/pprof"
)

// getAACClient gets an instance of AAC on startup.
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

func getEnvVar(name, defaultValue string) string {
	for _, e := range os.Environ() {
		kv := strings.Split(e, "=")
		if kv[0] == name {
			return kv[1]
		}
	}
	return defaultValue
}

func main() {
	// Load Configuration from conf.json
	appConfiguration := config.NewAppConfiguration()
	dbConfig := appConfiguration.DatabaseConnection
	serverConfig := appConfiguration.ServerSettings

	// Setup handle to the database
	db, err := dbConfig.GetDatabaseHandle()
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	// Validate the DSN for the database by pinging it
	pingDBresult := pingDB(db)
	if pingDBresult != 0 {
		// stop if we couldnt ping
		os.Exit(pingDBresult)
	}

	// Setup web server
	s, handler, err := makeServer(serverConfig, db)
	if err != nil {
		log.Fatalf("Fatal error in call to makeServer(): %v", err)
	}
	// with TLS support
	// TODO: Should we encapsulate setting this TLSConfig in makeServer?
	stls := serverConfig.GetTLSConfig()
	s.TLSConfig = &stls
	serverCertFile := serverConfig.ServerCertChain
	serverKeyFile := serverConfig.ServerKey

	handler.MasterKey = getEnvVar("masterkey", "otterpaws")
	if handler.MasterKey == "otterpaws" {
		log.Printf("You should pass in an environment variable 'masterkey' to encrypt database keys")
		log.Printf("Note that if you change masterkey, then the encrypted keys are invalidated")
	}

	// start pprof handler
	//	go func() {
	//		log.Println(http.ListenAndServe("0.0.0.0:4480", nil))
	//	}()

	// start it
	log.Println("Starting server on " + s.Addr)
	log.Fatalln(s.ListenAndServeTLS(serverCertFile, serverKeyFile))
}

// Check the schema and return the cache id that corresponds to it
func schemaCheck(concreteDAO dao.DAO) string {
	//Get information about the database we connected to
	dbState, err := concreteDAO.GetDBState()
	if err != nil {
		log.Printf("!!! Error checking dbState: %v, %v", err, reflect.TypeOf(err))
	} else {
		if dbState.SchemaVersion != dao.SchemaVersion {
			log.Printf(
				"!!! The schema version does not match.  Upgrade the database or risk corruption !!!. '%s' vs '%s'",
				dbState.SchemaVersion,
				dao.SchemaVersion,
			)
			log.Printf("TODO: A data/schema migration should happen right here")
		}
	}
	log.Printf("Database version %s instance is %s", dbState.SchemaVersion, dbState.Identifier)
	return fmt.Sprintf("cache-%s", dbState.Identifier)
}

func makeServer(serverConfig config.ServerSettingsConfiguration, db *sqlx.DB) (*http.Server, *server.AppServer, error) {
	//Try to connect to AAC
	var aac *aac.AacServiceClient
	var err error
	attempts := 120
	for {
		//Give time for AAC connect - EC2 micro needs about 20s
		log.Printf("Waiting to connect to AAC.")
		time.Sleep(1 * time.Second) //there is a fatal in aac connecting, so must sleep
		aac, err = getAACClient()
		if err != nil || aac == nil {
			//TODO: include in DB ping
			log.Printf("Waiting for AAC:%v", err)
		} else {
			log.Printf("We are connected to AAC")
			break
		}
		attempts--
		if attempts <= 0 {
			break
		}
	}

	concreteDAO := dao.DataAccessLayer{MetadataDB: db}
	cacheID := schemaCheck(&concreteDAO)

	templates, err := template.ParseGlob(
		filepath.Join(oduconfig.ProjectRoot,
			"cmd", "metadataconnector", "libs", "server",
			"static", "templates", "*"))
	if err != nil {
		log.Printf("Cloud not discover templates.")
		return nil, nil, err
	}

	staticPath := filepath.Join(oduconfig.ProjectRoot, "cmd", "metadataconnector", "libs", "server", "static")

	//XXXX This default resolves from the docker containers.
	// dockervm doesnt work or resolve from outside
	zkAddress := getEnvVar("ZKURL", "zk:2181")
	zkState, err := zookeeper.RegisterApplication(oduconfig.RootURL, zkAddress)
	if err != nil {
		panic("We cannot run without zookeeper!")
	}
	err = zookeeper.ServiceAnnouncement(zkState, "https", "ALIVE", oduconfig.MyIP, "4430")
	if err != nil {
		panic("We were unable to register with zookeeper!")
	}

	httpHandler := server.AppServer{
		Port:          serverConfig.ListenPort,
		Bind:          serverConfig.ListenBind,
		Addr:          serverConfig.ListenBind + ":" + strconv.Itoa(serverConfig.ListenPort),
		DAO:           &concreteDAO,
		DrainProvider: server.NewS3DrainProvider(cacheID),
		Tracker:       performance.NewJobReporters(1024),
		AAC:           aac,
		ServicePrefix: oduconfig.RootURLRegex,
		TemplateCache: templates,
		StaticDir:     staticPath,
		ZKState:       zkState,
	}

	if httpHandler.AAC == nil {
		panic("We cannot run without the AAC!")
	}

	log.Printf("Using root url:%s", oduconfig.RootURL)
	log.Printf("Using root url regex:%s", oduconfig.RootURLRegex)

	// Compile regexes for Routes
	httpHandler.InitRegex()

	return &http.Server{
		Addr:           string(httpHandler.Addr),
		Handler:        httpHandler,
		ReadTimeout:    100000 * time.Second, //This breaks big downloads
		WriteTimeout:   100000 * time.Second,
		MaxHeaderBytes: 1 << 20, //This prevents clients from DOS'ing us
	}, &httpHandler, nil
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
			fmt.Println("Timeout connecting to database.")
			exitCode = 28
		} else if match, _ := regexp.MatchString(".*lookup.*", err.Error()); match {
			fmt.Println("Unknown host error connecting to database. Review conf.json configuration. Halting")
			exitCode = 6
			return exitCode
		} else if match, _ := regexp.MatchString(".*connection refused.*", err.Error()); match {
			fmt.Println("Connection refused connecting to database. Database may not yet be online.")
			exitCode = 7
		} else {
			fmt.Println("Unhandled error while connecting to database.")
			fmt.Println(err.Error())
			fmt.Println("Halting")
			exitCode = 1
			return exitCode
		}
		if !dbPingSuccess {
			if dbPingAttempt < dbPingAttemptMax {
				fmt.Println("Retrying in 3 seconds")
				time.Sleep(time.Second * 3)
			} else {
				fmt.Println("Maximum retries exhausted. Halting")
				return exitCode
			}
		}
	}
	return exitCode
}
