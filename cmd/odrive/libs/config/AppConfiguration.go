package config

import (
	"crypto/tls"
	"log"
	"os"
	"path/filepath"

	globalconfig "decipher.com/object-drive-server/config"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
)

// Globals
var (
	defaultDBDriver = "mysql"
	defaultDBHost   = "127.0.0.1"
	defaultDBPort   = "3306"
	DefaultBucket   = globalconfig.GetEnvOrDefault("OD_AWS_S3_BUCKET", "")
)

// AppConfiguration is a structure that defines the known configuration format
// for this application.
type AppConfiguration struct {
	AuditorSettings    AuditSvcConfiguration
	DatabaseConnection DatabaseConfiguration
	ServerSettings     ServerSettingsConfiguration
	AACSettings        AACConfiguration
}

// AACConfiguration ...
type AACConfiguration struct {
	CAPath     string
	ClientCert string
	ClientKey  string
	HostName   string
	Port       string
}

// DatabaseConfiguration is a structure that defines the attributes
// needed for setting up database connection
type DatabaseConfiguration struct {
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
	PathToStaticFiles         string
	PathToTemplateFiles       string
}

// AuditSvcConfiguration defines the attributes needed for connecting to the audit service
type AuditSvcConfiguration struct {
	Type string
	Port string
	Host string
}

// NewAppConfigurationWithDefaults provides some defaults to the constructor
// function for AppConfiguration. Normally these parameters are specified
// on the command line.
func NewAppConfigurationWithDefaults() AppConfiguration {
	ciphers := []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"}
	useTLS := true
	whitelist := []string{"cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"}
	staticRootPath := filepath.Join("libs", "server", "static")
	templatePath := filepath.Join("libs", "server", "static", "templates")
	tlsMinimumVersion := "1.2"
	return NewAppConfiguration(whitelist, ciphers, useTLS, staticRootPath, templatePath, tlsMinimumVersion)
}

// NewAppConfiguration loads the configuration from the environment. Parameters are command
// line flags.
func NewAppConfiguration(whitelist, ciphers []string, useTLS bool, staticRootPath string, templatePath string, tlsMinimumVersion string) AppConfiguration {

	dbConf := NewDatabaseConfigFromEnv()
	serverSettings := NewServerSettingsFromEnv(whitelist, ciphers, useTLS, staticRootPath, templatePath, tlsMinimumVersion)
	aacSettings := NewAACSettingsFromEnv()

	return AppConfiguration{
		DatabaseConnection: dbConf,
		ServerSettings:     serverSettings,
		AACSettings:        aacSettings,
	}
}

// NewDatabaseConfigFromEnv inspects the environment and returns a DatabaseConfiguration.
func NewDatabaseConfigFromEnv() DatabaseConfiguration {

	var dbConf DatabaseConfiguration

	// From environment
	dbConf.Username = os.Getenv(OD_DB_USERNAME)
	dbConf.Password = os.Getenv(OD_DB_PASSWORD)
	dbConf.Host = os.Getenv(OD_DB_HOST)
	dbConf.Port = os.Getenv(OD_DB_PORT)
	dbConf.Schema = os.Getenv(OD_DB_SCHEMA)
	dbConf.CAPath = os.Getenv(OD_DB_CA)
	dbConf.ClientCert = os.Getenv(OD_DB_CERT)
	dbConf.ClientKey = os.Getenv(OD_DB_KEY)

	// Defaults
	dbConf.Protocol = "tcp"
	dbConf.Driver = defaultDBDriver
	dbConf.Params = "parseTime=true&collation=utf8mb4_unicode_ci"
	dbConf.UseTLS = true
	dbConf.SkipVerify = true // TODO new variable?

	return dbConf
}

// NewServerSettingsFromEnv inspects the environment and returns a ServerSettingsConfiguration.
func NewServerSettingsFromEnv(whitelist, ciphers []string, useTLS bool, staticRootPath string, templatePath string, tlsMinimumVersion string) ServerSettingsConfiguration {

	var settings ServerSettingsConfiguration

	// From env
	settings.ListenPort = os.Getenv(OD_SERVER_PORT)
	settings.CAPath = os.Getenv(OD_SERVER_CA)
	settings.ServerCertChain = os.Getenv(OD_SERVER_CERT)
	settings.ServerKey = os.Getenv(OD_SERVER_KEY)

	// Defaults
	settings.ListenBind = "0.0.0.0"
	settings.UseTLS = useTLS
	settings.RequireClientCert = true
	settings.MinimumVersion = tlsMinimumVersion
	settings.AclImpersonationWhitelist = whitelist
	settings.CipherSuites = ciphers
	settings.PathToStaticFiles = staticRootPath
	settings.PathToTemplateFiles = templatePath

	return settings
}

// NewAACSettingsFromEnv inspects the environment and returns a AACConfiguration.
func NewAACSettingsFromEnv() AACConfiguration {

	var conf AACConfiguration

	conf.CAPath = os.Getenv(OD_AAC_CA)
	conf.ClientCert = os.Getenv(OD_AAC_CERT)
	conf.ClientKey = os.Getenv(OD_AAC_KEY)

	// These should get overridden with zookeeper nodes found in OD_ZK_AAC
	conf.HostName = os.Getenv(OD_AAC_HOST)
	conf.Port = os.Getenv(OD_AAC_PORT)

	return conf
}

// GetDatabaseHandle initializes database connection using the configuration
func (r *DatabaseConfiguration) GetDatabaseHandle() (*sqlx.DB, error) {
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
func (r *DatabaseConfiguration) buildDSN() string {
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
	logger.Info("Using this connection string", zap.String("dbdsn", dbDSN))
	return dbDSN
}

// buildTLSConfig prepares a standard go tls.Config with RootCAs and client
// Certificates for communicating with the database securely.
func (r *DatabaseConfiguration) buildTLSConfig() tls.Config {
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
