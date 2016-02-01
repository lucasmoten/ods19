package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"

	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/server"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

//TODO: not sure how much is safe to share concurrently
//This is account as in the ["default"] entry in ~/.aws/credentials
func awsS3(account string) (*s3.S3, *session.Session) {
	sessionConfig := &aws.Config{
		Credentials: credentials.NewSharedCredentials("", account),
	}
	sess := session.New(sessionConfig)
	svc := s3.New(sess)
	return svc, sess
}

func main() {

	// uri := "/object//list"
	// var rxListObjects = regexp.MustCompile("" + "/object/.*/list")
	// var rxObject = regexp.MustCompile("" + "/object")
	// switch {
	// case rxListObjects.MatchString(uri):
	// 	fmt.Println("matched rxListObjects")
	// case rxObject.MatchString(uri):
	// 	fmt.Println("matched rxObject")
	// case true:
	// 	fmt.Println("true #1")
	// 	fallthrough
	// case true:
	// 	fmt.Println("true #2")
	// }
	// os.Exit(-1)

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
	s, err := makeServer(serverConfig, db)
	// with TLS support
	stls := serverConfig.GetTLSConfig()
	s.TLSConfig = &stls
	serverCertFile := serverConfig.ServerCertChain
	serverKeyFile := serverConfig.ServerKey
	// start it
	log.Println("Starting server on " + s.Addr)
	log.Fatalln(s.ListenAndServeTLS(serverCertFile, serverKeyFile))
}

func makeServer(serverConfig config.ServerSettingsConfiguration, db *sqlx.DB) (*http.Server, error) {
	//On machines with multiple configs, we at least assume that objectdrive is aliased to
	//the default config
	s3, awsSession := awsS3("default")

	httpHandler := server.AppServer{
		Port:          serverConfig.ListenPort,
		Bind:          serverConfig.ListenBind,
		Addr:          serverConfig.ListenBind + ":" + strconv.Itoa(serverConfig.ListenPort),
		MetadataDB:    db,
		S3:            s3,
		AWSSession:    awsSession,
		CacheLocation: "cache",
		ServicePrefix: serverConfig.ServiceName + serverConfig.ServiceVersion,
	}

	return &http.Server{
		Addr:           string(httpHandler.Addr),
		Handler:        httpHandler,
		ReadTimeout:    10000 * time.Second, //This breaks big downloads
		WriteTimeout:   10000 * time.Second,
		MaxHeaderBytes: 1 << 20, //This prevents clients from DOS'ing us
	}, nil
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
