package config

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"path/filepath"

	globalconfig "decipher.com/object-drive-server/config"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
	"github.com/urfave/cli"
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

// AACConfiguration holds data required for an AAC client.
type AACConfiguration struct {
	CAPath     string
	ClientCert string
	ClientKey  string
	HostName   string
	Port       string
}

// CommandLineOpts holds command line options so they can be passed as a param.
type CommandLineOpts struct {
	Ciphers           []string
	UseTLS            bool
	StaticRootPath    string
	TemplateDir       string
	TLSMinimumVersion string
	Conf              string
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
	BasePath                  string
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
	whitelist := []string{"cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"}
	opts := CommandLineOpts{
		Ciphers:           []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
		UseTLS:            true,
		StaticRootPath:    filepath.Join("libs", "server", "static"),
		TemplateDir:       filepath.Join("libs", "server", "static", "templates"),
		TLSMinimumVersion: "1.2",
	}
	return NewAppConfiguration(whitelist, opts)
}

// NewAppConfiguration loads the configuration from the environment. Parameters are command
// line flags.
func NewAppConfiguration(whitelist []string, opts CommandLineOpts) AppConfiguration {

	dbConf := NewDatabaseConfigFromEnv()
	serverSettings := NewServerSettingsFromEnv(whitelist, opts)
	aacSettings := NewAACSettingsFromEnv()

	return AppConfiguration{
		DatabaseConnection: dbConf,
		ServerSettings:     serverSettings,
		AACSettings:        aacSettings,
	}
}

// NewCommandLineOpts instantiates CommandLineOpts from a pointer to the parsed command line
// context. The actual parsing is handled by the cli framework.
func NewCommandLineOpts(clictx *cli.Context) CommandLineOpts {
	ciphers := clictx.StringSlice("addCipher")
	useTLS := clictx.BoolT("useTLS")
	// NOTE: cli lib appends to []string that already contains the "default" value. Must trim
	// the default cipher if addCipher is passed at command line.
	if len(ciphers) > 1 {
		ciphers = ciphers[1:]
	}

	// Config file YAML is parsed elsewhere. This is just the path.
	confPath := clictx.String("conf")

	// Static Files Directory (Optional. Has a default, but can be set to empty for no static files)
	staticRootPath := clictx.String("staticRoot")
	if len(staticRootPath) > 0 {
		if _, err := os.Stat(staticRootPath); os.IsNotExist(err) {
			fmt.Printf("Static Root Path %s does not exist: %v\n", staticRootPath, err)
			os.Exit(1)
		}
	}

	// Template Directory (Optional. Has a default, but can be set to empty for no templates)
	templateDir := clictx.String("templateDir")
	if len(templateDir) > 0 {
		if _, err := os.Stat(templateDir); os.IsNotExist(err) {
			fmt.Printf("Template folder %s does not exist: %v\n", templateDir, err)
			os.Exit(1)
		}
	}

	// TLS Minimum Version (Optional. Has a default, but can be made a lower version)
	tlsMinimumVersion := clictx.String("tlsMinimumVersion")

	return CommandLineOpts{
		Ciphers:           ciphers,
		UseTLS:            useTLS,
		Conf:              confPath,
		StaticRootPath:    staticRootPath,
		TemplateDir:       templateDir,
		TLSMinimumVersion: tlsMinimumVersion,
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
	dbConf.Params = os.Getenv(OD_DB_CONN_PARAMS)

	if dbConf.Params == "" {
		msg := "OD_DB_CONN_PARAMS is blank. Recommended value: parseTime=true&collation=utf8_unicode_ci"
		logger.Warn("db warning", zap.String("config_warning", msg))
	}

	// Defaults
	dbConf.Protocol = "tcp"
	dbConf.Driver = defaultDBDriver
	dbConf.UseTLS = true
	dbConf.SkipVerify = true // TODO new variable?

	return dbConf
}

// NewServerSettingsFromEnv inspects the environment and returns a ServerSettingsConfiguration.
func NewServerSettingsFromEnv(whitelist []string, opts CommandLineOpts) ServerSettingsConfiguration {

	var settings ServerSettingsConfiguration

	// From env
	settings.BasePath = globalconfig.GetEnvOrDefault(OD_SERVER_BASEPATH, "/services/object-drive/1.0")
	settings.ListenPort = os.Getenv(OD_SERVER_PORT)
	settings.CAPath = os.Getenv(OD_SERVER_CA)
	settings.ServerCertChain = os.Getenv(OD_SERVER_CERT)
	settings.ServerKey = os.Getenv(OD_SERVER_KEY)

	// Defaults
	settings.ListenBind = "0.0.0.0"
	settings.UseTLS = opts.UseTLS
	settings.RequireClientCert = true
	settings.MinimumVersion = opts.TLSMinimumVersion
	settings.AclImpersonationWhitelist = whitelist
	settings.CipherSuites = opts.Ciphers
	settings.PathToStaticFiles = opts.StaticRootPath
	settings.PathToTemplateFiles = opts.TemplateDir

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
