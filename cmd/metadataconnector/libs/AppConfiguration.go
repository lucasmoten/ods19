package config

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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
	Username   string
	Password   string
	Host       string
	Port       string
	Schema     string
	Params     string
	UseTLS     bool
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

func (r *DatabaseConnectionConfiguration) GetDSN() string {
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
	if len(r.Host) > 0 {
		dbDSN += "tcp(" + r.Host + ":" + r.Port + ")"
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

func (r *DatabaseConnectionConfiguration) GetTLSConfig() tls.Config {
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
		RootCAs:      rootCertPool,
		Certificates: clientCert,
		ServerName:   r.Host,
		//InsecureSkipVerify: true,
	}
}
