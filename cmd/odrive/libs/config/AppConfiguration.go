package config

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"os"

	globalconfig "decipher.com/object-drive-server/config"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var (
	defaultDBDriver = "mysql"
	defaultDBHost   = "127.0.0.1"
	defaultDBPort   = "3306"
	// DefaultBucket is the AWS S3 bucket name
	DefaultBucket = globalconfig.GetEnvOrDefault("OD_AWS_S3_BUCKET", "decipherers")
)

// AppConfiguration is a structure that defines the known configuration format
// for this application.
type AppConfiguration struct {
	AuditorSettings    AuditSvcConfiguration
	DatabaseConnection DatabaseConnectionConfiguration
	ServerSettings     ServerSettingsConfiguration
}

// DatabaseConnectionConfiguration is a structure that defines the attributes
// needed for setting up database connection
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

// ServerSettingsConfiguration is a structure defining the attributes needed for
// setting up the server listener
type ServerSettingsConfiguration struct {
	ListenPort                string
	ListenBind                string
	UseTLS                    bool
	CAPath                    string
	ServerCertChain           string
	ServerKey                 string
	RequireClientCert         bool
	CipherSuites              []string
	MinimumVersion            string
	AclImpersonationWhitelist []string
}

// AuditSvcConfiguration defines the attributes needed for connecting to the audit service
type AuditSvcConfiguration struct {
	Type string
	Port string
	Host string
}

// NewAppConfiguration loads the configuration file and returns an AppConfiguration.
func NewAppConfiguration(path string) AppConfiguration {

	file, err := os.Open(path)
	if err != nil {
		fmt.Println("conf.json not found")
	}
	decoder := json.NewDecoder(file)
	configuration := AppConfiguration{}
	err = decoder.Decode(&configuration)
	if err != nil {
		log.Fatal("Could not decode configuration file")
	}

	warnIfNotSet("GOPATH")

	configuration.DatabaseConnection.Driver = os.ExpandEnv(configuration.DatabaseConnection.Driver)
	configuration.DatabaseConnection.Username = os.ExpandEnv(configuration.DatabaseConnection.Username)
	configuration.DatabaseConnection.Password = os.ExpandEnv(configuration.DatabaseConnection.Password)
	configuration.DatabaseConnection.Protocol = os.ExpandEnv(configuration.DatabaseConnection.Protocol)
	configuration.DatabaseConnection.Host = os.ExpandEnv(configuration.DatabaseConnection.Host)
	configuration.DatabaseConnection.Port = os.ExpandEnv(configuration.DatabaseConnection.Port)
	configuration.DatabaseConnection.Schema = os.ExpandEnv(configuration.DatabaseConnection.Schema)
	configuration.DatabaseConnection.Params = os.ExpandEnv(configuration.DatabaseConnection.Params)
	configuration.DatabaseConnection.CAPath = os.ExpandEnv(configuration.DatabaseConnection.CAPath)
	configuration.DatabaseConnection.ClientCert = os.ExpandEnv(configuration.DatabaseConnection.ClientCert)
	configuration.DatabaseConnection.ClientKey = os.ExpandEnv(configuration.DatabaseConnection.ClientKey)
	configuration.ServerSettings.ListenPort = os.ExpandEnv(configuration.ServerSettings.ListenPort)
	configuration.ServerSettings.ListenBind = os.ExpandEnv(configuration.ServerSettings.ListenBind)

	configuration.ServerSettings.CAPath = os.ExpandEnv(configuration.ServerSettings.CAPath)
	configuration.ServerSettings.ServerCertChain = os.ExpandEnv(configuration.ServerSettings.ServerCertChain)
	configuration.ServerSettings.ServerKey = os.ExpandEnv(configuration.ServerSettings.ServerKey)

	// Done
	return configuration
}

func displayFormatForConfigFile() {
	samplefile := `
	{
      "AuditSvc": {
        "type": "blackhole",
        "host": "",
        "port": ""
      },        
	  "DatabaseConnection": {
	    "Driver": "mysql"
	    ,"Username": "username"
	    ,"Password": "password"
	    ,"Protocol": "tcp"
	    ,"Host": "fully.qualified.domain.name.for.database.host"
	    ,"Port": "port"
	    ,"Schema": "databasename"
	    ,"Params": "additional parameters, if any"
	    ,"UseTLS": true
	    ,"SkipVerify": false
	    ,"CAPath": "/path/to/trust/folder/of/ca/pems"
	    ,"ClientCert": "/path/to/database/client/cert/pem"
	    ,"ClientKey": "/path/to/database/client/key/pem"
	  },
	  "ServerSettings": {
	    "ListenPort": port
	    ,"ListenBind": "0.0.0.0"
	    ,"UseTLS": true
	    ,"CAPath": "/path/to/trust/folder/of/ca/pems"
	    ,"ServerCertChain": "/path/to/web/server/cert/pem"
	    ,"ServerKey": "/path/to/web/server/key/pem"
	    ,"RequireClientCert": true
	    ,"CipherSuites" : [
	       "TLS_RSA_WITH_RC4_128_SHA"
	  		,"TLS_RSA_WITH_3DES_EDE_CBC_SHA"
	  		,"TLS_RSA_WITH_AES_128_CBC_SHA"
	  		,"TLS_RSA_WITH_AES_256_CBC_SHA"
	  		,"TLS_ECDHE_ECDSA_WITH_RC4_128_SHA"
	  		,"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA"
	  		,"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA"
	  		,"TLS_ECDHE_RSA_WITH_RC4_128_SHA"
	  		,"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA"
	  		,"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA"
	  		,"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA"
	  		,"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
	  		,"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"
	  		,"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
	  		,"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384"
	    ]
	    ,"MinimumVersion": "1.2"
        ,"AclImpersonationWhitelist": [
            "cn=server allowed to impersonate,ou=org1,ou=org2,o=organization,c=us"
        ]
	  }
	}
	`
	fmt.Println(samplefile)
}

// GetDatabaseHandle initializes database connection using the configuration
func (r *DatabaseConnectionConfiguration) GetDatabaseHandle() (*sqlx.DB, error) {
	// Establish configuration settings for Database Connection using
	// the TLS settings in config file
	if r.UseTLS {
		dbTLS := r.buildTLSConfig()
		switch r.Driver {
		case defaultDBDriver:
			mysql.RegisterTLSConfig("custom", &dbTLS)
		default:
			panic("Driver not supported")
		}
	}
	// Setup handle to the database
	db, err := sqlx.Open(r.Driver, r.buildDSN())
	db.SetMaxIdleConns(globalconfig.GetEnvOrDefaultInt("OD_DB_MAXIDLECONNS", 10))
	db.SetMaxOpenConns(globalconfig.GetEnvOrDefaultInt("OD_DB_MAXOPENCONNS", 10))
	return db, err
}

// GetTLSConfig returns the build TLS Configuration object based upon Server
// Settings Configuration
func (r *ServerSettingsConfiguration) GetTLSConfig() tls.Config {
	return r.buildTLSConfig()
}

// buildDSN prepares a Data Source Name (DNS) suitable for mysql using the
// driver and documentation found here: https://github.com/go-sql-driver/mysql.
// This format is similar to the PEAR DB format, but may need alteration.
// http://pear.php.net/manual/en/package.database.db.intro-dsn.php
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
			dbDSN += defaultDBHost
		}
		dbDSN += ":"
		if len(r.Port) > 0 {
			dbDSN += r.Port
		} else {
			// default port by database type
			switch r.Driver {
			case defaultDBDriver:
				dbDSN += defaultDBPort
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
	log.Printf("Using this connection string: %s\n", dbDSN)
	return dbDSN
}

// buildTLSConfig prepares a standard go tls.Config with RootCAs and client
// Certificates for communicating with the database securely.
func (r *DatabaseConnectionConfiguration) buildTLSConfig() tls.Config {
	return buildClientTLSConfig(r.CAPath, r.ClientCert, r.ClientKey, r.Host, r.SkipVerify)
}

// buildTLSConfig prepares a standard go tls.Config with trusted CAs and
// server identity certificates to listen for connecting clients
func (r *ServerSettingsConfiguration) buildTLSConfig() tls.Config {
	return buildServerTLSConfig(r.CAPath, r.ServerCertChain, r.ServerKey, r.RequireClientCert, r.CipherSuites, r.MinimumVersion)
}

func warnIfNotSet(variable string) {
	if len(globalconfig.GetEnvOrDefault(variable, "")) == 0 {
		log.Printf("WARNING: %s not set.\n", variable)
	}
}
