package config

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

/*
AppConfiguration is a structure that defines the known configuration format
for this application.
*/
type AppConfiguration struct {
	DatabaseConnection DatabaseConnectionConfiguration
}

/*
DatabaseConnectionConfiguration is a structure that defines the attributes
needed for setting up database connection
*/
type DatabaseConnectionConfiguration struct {
	Driver     string
	Username   string
	Password   string
	Protocol   string
	Host       string
	Port       string
	Schema     string
	Params     string
	UseTLS     bool
	SkipVerify bool
	CAPath     string
	ClientCert string
	ClientKey  string
}

/*
NewAppConfiguration loads the configuration file and returns the mapped object
*/
func NewAppConfiguration() AppConfiguration {
	file, _ := os.Open("conf.json")
	decoder := json.NewDecoder(file)
	configuration := AppConfiguration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("error:", err)
	}
	return configuration
}

func (r *DatabaseConnectionConfiguration) GetDatabaseHandle() (*sqlx.DB, error) {
	// Establish configuration settings for Database Connection using
	// the TLS settings in config file
	if r.UseTLS {
		dbTLS := r.buildTLSConfig()
		switch r.Driver {
		case "mysql":
			mysql.RegisterTLSConfig("custom", &dbTLS)
		default:
			panic("Driver not supported")
		}
	}
	// Setup handle to the database
	db, err := sqlx.Open(r.Driver, r.buildDSN())
	return db, err
}

// =============================================================================
// Unexported members
// =============================================================================

/*
buildDSN prepares a Data Source Name (DNS) suitable for mysql using the driver
and documentation found here: https://github.com/go-sql-driver/mysql.
This format is similar to the PEAR DB format but may need alteration
http://pear.php.net/manual/en/package.database.db.intro-dsn.php
*/
func (r *DatabaseConnectionConfiguration) buildDSN() string {
	var dbDSN = ""
	if len(r.Username) > 0 {
		dbDSN += r.Username
		if len(r.Password) > 0 {
			dbDSN += ":" + r.Password
		}
	}
	if len(dbDSN) > 0 {
		dbDSN += "@"
	}
	if len(r.Protocol) > 0 {
		dbDSN += r.Protocol + "("
		if len(r.Host) > 0 {
			dbDSN += r.Host
		} else {
			// default to localhost
			dbDSN += "127.0.0.1"
		}
		dbDSN += ":"
		if len(r.Port) > 0 {
			dbDSN += r.Port
		} else {
			// default port by database type
			switch r.Driver {
			case "mysql":
				dbDSN += "3306"
			default:
				panic("Driver not supported")
			}
		}
		dbDSN += ")"
	}
	dbDSN += "/"
	if len(r.Schema) > 0 {
		dbDSN += r.Schema
	}
	if (len(r.Params) > 0) || (r.UseTLS) {
		dbDSN += "?"
		if r.UseTLS {
			dbDSN += "tls=custom"
			if len(r.Params) > 0 {
				dbDSN += "&"
			}
		}
		if len(r.Params) > 0 {
			dbDSN += r.Params
		}
	}
	return dbDSN
}

func (r *DatabaseConnectionConfiguration) buildTLSConfig() tls.Config {
	// Certificates setup for TLS to MySQL database
	rootCertPool := x509.NewCertPool()
	pem, err := ioutil.ReadFile(r.CAPath)
	if err != nil {
		log.Fatal(err)
	}
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		log.Fatal("Failed to append PEM.")
	}
	clientCert := make([]tls.Certificate, 0, 1)
	certs, err := tls.LoadX509KeyPair(r.ClientCert, r.ClientKey)
	if err != nil {
		log.Fatal(err)
	}
	clientCert = append(clientCert, certs)

	return tls.Config{
		RootCAs:            rootCertPool,
		Certificates:       clientCert,
		ServerName:         r.Host,
		InsecureSkipVerify: r.SkipVerify,
	}
}
