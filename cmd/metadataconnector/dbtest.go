package main

import (
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	thrift "github.com/samuel/go-thrift/thrift"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/server"

	oduconfig "decipher.com/oduploader/config"
	"decipher.com/oduploader/performance"
	aac "decipher.com/oduploader/services/aac"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

//BuildClassificationMap explicitly builds the classification map,
//which might come from a file later
//XXX it's an eyesore to be here, but we need something for now.
func BuildClassificationMap() map[string]string {
	log.Printf("building classification map....")

	ClassificationMap := make(map[string]string)

	ClassificationMap["U"] = `
        {"version":"2.1.0", "classif":"U"}
        `

	ClassificationMap["C"] = `
        {"version":"2.1.0", "classif":"C"}
        `
	ClassificationMap["S"] = `
        {"version":"2.1.0", "classif":"S"}
        `
	ClassificationMap["T"] = `
        {"version":"2.1.0", "classif":"T"}
        `

	ClassificationMap["SCI1"] = `
{
 "version":"2.1.0",
 "classif":"TS",
 "owner_prod":[],
 "atom_energy":[],
 "sar_id":[],
 "sci_ctrls":[ "HCS", "SI-G", "TK" ],
        "disponly_to":[ "" ],
        "dissem_ctrls":[ "OC" ],
        "non_ic":[],
        "rel_to":[],
        "fgi_open":[],
        "fgi_protect":[],
        "portion":"TS//HCS/SI-G/TK//OC",
        "banner":"TOP SECRET//HCS/SI-G/TK//ORCON",
        "dissem_countries":[ "USA" ],
        "accms":[],
        "macs":[],
        "oc_attribs":[ { "orgs":[ "dia" ], "missions":[], "regions":[] } ],
        "f_clearance":[ "ts" ],
        "f_sci_ctrls":[ "hcs", "si_g", "tk" ],
        "f_accms":[],
        "f_oc_org":[ "dia", "dni" ],
        "f_regions":[],
				"f_missions":[],
        "f_share":[],
        "f_atom_energy":[],
        "f_macs":[],
        "disp_only":""
}`
	return ClassificationMap
}

/**
Get an instance of AAC on startup.
Fail to come up if we can't do this.
TODO: restart uploader if we lose AAC connection
*/
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

	// Check environment variables
	checkAWSEnvironmentVars()

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
	stls := serverConfig.GetTLSConfig()
	s.TLSConfig = &stls
	serverCertFile := serverConfig.ServerCertChain
	serverKeyFile := serverConfig.ServerKey

	handler.MasterKey = getEnvVar("masterkey", "otterpaws")
	if handler.MasterKey == "otterpaws" {
		log.Printf("You should pass in an environment variable 'masterkey' to encrypt database keys")
		log.Printf("Note that if you change masterkey, then the encrypted keys are invalidated")
	}

	go handler.CachePurge()

	// start it
	log.Println("Starting server on " + s.Addr)
	log.Fatalln(s.ListenAndServeTLS(serverCertFile, serverKeyFile))
}

//TODO: not sure how much is safe to share concurrently
//This is account as in the ["default"] entry in ~/.aws/credentials
func awsS3() (*s3.S3, *session.Session) {
	sessionConfig := &aws.Config{
		Credentials: credentials.NewEnvCredentials(),
	}
	sess := session.New(sessionConfig)
	svc := s3.New(sess)
	return svc, sess
}

func makeServer(serverConfig config.ServerSettingsConfiguration, db *sqlx.DB) (*http.Server, *server.AppServer, error) {
	//On machines with multiple configs, we at least assume that objectdrive is aliased to
	//the default config
	s3, awsSession := awsS3()

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

	templates, err := template.ParseGlob(
		filepath.Join(oduconfig.ProjectRoot,
			"cmd", "metadataconnector", "libs", "server",
			"static", "templates", "*"))
	if err != nil {
		log.Printf("Cloud not discover templates.")
		return nil, nil, err
	}

	httpHandler := server.AppServer{
		Port:            serverConfig.ListenPort,
		Bind:            serverConfig.ListenBind,
		Addr:            serverConfig.ListenBind + ":" + strconv.Itoa(serverConfig.ListenPort),
		MetadataDB:      db,
		S3:              s3,
		AWSSession:      awsSession,
		CacheLocation:   "cache",
		ServicePrefix:   serverConfig.ServiceName + serverConfig.ServiceVersion,
		Classifications: BuildClassificationMap(),
		Tracker:         performance.NewJobReporters(1024),
		AAC:             aac,
		TemplateCache:   templates,
	}

	if httpHandler.AAC == nil {
		panic("We cannot run without the AAC!")
	}

	return &http.Server{
		Addr:           string(httpHandler.Addr),
		Handler:        httpHandler,
		ReadTimeout:    10000 * time.Second, //This breaks big downloads
		WriteTimeout:   10000 * time.Second,
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

// checkAWSEnvironmentVars prevents the server from starting if appropriate vars
// are not set.
func checkAWSEnvironmentVars() {
	region := os.Getenv("AWS_REGION")
	accessKey := os.Getenv("AWS_ACCESS_KEY")
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	if region == "" || accessKey == "" || accessKeyID == "" {
		log.Fatal("Fatal Error: Environment variables AWS_REGION, AWS_ACCESS_KEY, and AWS_ACCESS_KEY_ID must be set.")
	}
	return
}
