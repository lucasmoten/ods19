package config

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var (
	defaultDBDriver = "mysql"
	defaultDBHost   = "127.0.0.1"
	defaultDBPort   = "3306"
)

// AppConfiguration is a structure that defines the known configuration format
// for this application.
type AppConfiguration struct {
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
	ListenPort        int
	ListenBind        string
	UseTLS            bool
	CAPath            string
	ServerCertChain   string
	ServerKey         string
	RequireClientCert bool
	CipherSuites      []string
	MinimumVersion    string
}

// NewAppConfiguration loads the configuration file and returns the mapped
// object
func NewAppConfiguration() AppConfiguration {
	file, _ := os.Open("conf.json")
	decoder := json.NewDecoder(file)
	configuration := AppConfiguration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("error:", err)
	}

	// Use reflection to iterate over the struct
	//ExpandEnvironmentVariables(configuration)
	// reflection issues with structs and pointers for now, so just hardcoding this. As the struct changes, this logic will also need updated
	configuration.DatabaseConnection.CAPath = os.ExpandEnv(configuration.DatabaseConnection.CAPath)
	configuration.DatabaseConnection.ClientCert = os.ExpandEnv(configuration.DatabaseConnection.ClientCert)
	configuration.DatabaseConnection.ClientKey = os.ExpandEnv(configuration.DatabaseConnection.ClientKey)
	configuration.ServerSettings.CAPath = os.ExpandEnv(configuration.ServerSettings.CAPath)
	configuration.ServerSettings.ServerCertChain = os.ExpandEnv(configuration.ServerSettings.ServerCertChain)
	configuration.ServerSettings.ServerKey = os.ExpandEnv(configuration.ServerSettings.ServerKey)

	// Done
	return configuration
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
	return db, err
}

// GetTLSConfig returns the build TLS Configuration object based upon Server
// Settings Configuration
func (r *ServerSettingsConfiguration) GetTLSConfig() tls.Config {
	return r.buildTLSConfig()
}

// =============================================================================
// Unexported members
// =============================================================================

// buildDSN prepares a Data Source Name (DNS) suitable for mysql using the
// driver and documentation found here: https://github.com/go-sql-driver/mysql.
// This format is similar to the PEAR DB format but may need alteration
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
