package config

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/uber-go/zap"
	"github.com/urfave/cli"

	globalconfig "decipher.com/object-drive-server/config"
)

// Globals
var (
	defaultDBDriver = "mysql"
	defaultDBHost   = "127.0.0.1"
	defaultDBPort   = "3306"
	DefaultBucket   = getEnvOrDefault("OD_AWS_S3_BUCKET", "")
)

// AppConfiguration is a structure that defines the known configuration format
// for this application.
type AppConfiguration struct {
	AuditorSettings    AuditSvcConfiguration
	DatabaseConnection DatabaseConfiguration
	ServerSettings     ServerSettingsConfiguration
	AACSettings        AACConfiguration
	CacheSettings      S3DrainProviderOpts
	ZK                 ZKSettings
	EventQueue         EventQueueConfiguration
}

// AACConfiguration holds data required for an AAC client.
type AACConfiguration struct {
	CAPath               string
	ClientCert           string
	ClientKey            string
	HostName             string
	Port                 string
	AACAnnouncementPoint string
}

// AuditSvcConfiguration defines the attributes needed for connecting to the audit service
type AuditSvcConfiguration struct {
	Type string
	Port string
	Host string
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

// EventQueueConfiguration configures publishing to the Kakfa event queue.
type EventQueueConfiguration struct {
	KafkaAddrs []string
}

// S3DrainProviderOpts describes our current disk cache configuration.
type S3DrainProviderOpts struct {
	Root          string
	Partition     string
	LowWatermark  float64
	HighWatermark float64
	EvictAge      int64
	WalkSleep     int64
}

// ServerSettingsConfiguration holds the attributes needed for
// setting up an AppServer listener.
type ServerSettingsConfiguration struct {
	BasePath                  string
	ListenPort                string
	ListenBind                string
	UseTLS                    bool
	CAPath                    string
	ServerCertChain           string
	ServerKey                 string
	MasterKey                 string
	RequireClientCert         bool
	CipherSuites              []string
	MinimumVersion            string
	AclImpersonationWhitelist []string
	PathToStaticFiles         string
	PathToTemplateFiles       string
}

// ZKSettings holds the data required to communicate with Zookeeper.
type ZKSettings struct {
	IP             string
	Port           string
	Address        string
	BasepathOdrive string
}

// NewAppConfiguration loads the configuration from the different sources in the environment.
// If multiple configuration sources can be used, the order of precedence is: env var overrides
// config file.
func NewAppConfiguration(opts CommandLineOpts) AppConfiguration {

	confFile, err := LoadYAMLConfig(opts.Conf)
	if err != nil {
		fmt.Printf("Error loading yaml configuration at path %s: %v\n", confFile, err)
		os.Exit(1)
	}

	dbConf := NewDatabaseConfigFromEnv(confFile, opts)
	serverSettings := NewServerSettingsFromEnv(confFile, opts)
	aacSettings := NewAACSettingsFromEnv(confFile, opts)
	cacheSettings := NewS3DrainProviderOpts(confFile, opts)
	zkSettings := NewZKSettingsFromEnv(confFile, opts)
	eventQueue := NewEventQueueConfiguration(confFile, opts)

	return AppConfiguration{
		AACSettings:        aacSettings,
		CacheSettings:      cacheSettings,
		DatabaseConnection: dbConf,
		EventQueue:         eventQueue,
		ServerSettings:     serverSettings,
		ZK:                 zkSettings,
	}
}

// NewAACSettingsFromEnv inspects the environment and returns a AACConfiguration.
func NewAACSettingsFromEnv(confFile ConfigFile, opts CommandLineOpts) AACConfiguration {

	var conf AACConfiguration

	conf.CAPath = os.Getenv(OD_AAC_CA)
	conf.ClientCert = os.Getenv(OD_AAC_CERT)
	conf.ClientKey = os.Getenv(OD_AAC_KEY)

	// These should get overridden with zookeeper nodes found in OD_ZK_AAC
	conf.HostName = os.Getenv(OD_AAC_HOST)
	conf.Port = os.Getenv(OD_AAC_PORT)
	//Notice that the protocol (thrift) is in this already
	conf.AACAnnouncementPoint = globalconfig.GetEnvOrDefault(OD_ZK_AAC, "/cte/service/aac/1.0/thrift")
	return conf
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
func NewDatabaseConfigFromEnv(confFile ConfigFile, opts CommandLineOpts) DatabaseConfiguration {

	var dbConf DatabaseConfiguration

	// From environment
	dbConf.Username = os.Getenv(OD_DB_USERNAME)
	dbConf.Password = os.Getenv(OD_DB_PASSWORD)
	dbConf.Host = os.Getenv(OD_DB_HOST)
	dbConf.Port = os.Getenv(OD_DB_PORT)
	dbConf.Schema = getEnvOrDefault(OD_DB_SCHEMA, "metadatadb")
	dbConf.CAPath = os.Getenv(OD_DB_CA)
	dbConf.ClientCert = os.Getenv(OD_DB_CERT)
	dbConf.ClientKey = os.Getenv(OD_DB_KEY)
	dbConf.Params = getEnvOrDefault(OD_DB_CONN_PARAMS, "parseTime=true&collation=utf8_unicode_ci")

	// Defaults
	dbConf.Protocol = "tcp"
	dbConf.Driver = defaultDBDriver
	dbConf.UseTLS = true
	dbConf.SkipVerify = true // TODO new variable?

	return dbConf
}

// NewEventQueueConfiguration reades the environment to provide the configuration for the Kafka event queue.
func NewEventQueueConfiguration(confFile ConfigFile, opts CommandLineOpts) EventQueueConfiguration {
	var fqc EventQueueConfiguration
	var empty []string
	fqc.KafkaAddrs = getEnvOrDefaultSplitStringSlice(OD_EVENT_KAFKA_ADDRS, empty)
	return fqc
}

// NewS3DrainProviderOpts reads the environment to provide the configuration options for
// S3DrainProvider.
func NewS3DrainProviderOpts(confFile ConfigFile, opts CommandLineOpts) S3DrainProviderOpts {
	return S3DrainProviderOpts{
		Root:          getEnvOrDefault(OD_CACHE_ROOT, "."),
		Partition:     getEnvOrDefault(OD_CACHE_PARTITION, "cache"),
		LowWatermark:  getEnvOrDefaultFloat(OD_CACHE_LOWWATERMARK, .50),
		HighWatermark: getEnvOrDefaultFloat(OD_CACHE_HIGHWATERMARK, .75),
		EvictAge:      getEnvOrDefaultInt(OD_CACHE_EVICTAGE, 300),
		WalkSleep:     getEnvOrDefaultInt(OD_CACHE_WALKSLEEP, 30),
	}

}

// NewServerSettingsFromEnv inspects the environment and returns a ServerSettingsConfiguration.
func NewServerSettingsFromEnv(confFile ConfigFile, opts CommandLineOpts) ServerSettingsConfiguration {

	var settings ServerSettingsConfiguration

	// From env
	settings.BasePath = getEnvOrDefault(OD_SERVER_BASEPATH, "/services/object-drive/1.0")
	settings.ListenPort = getEnvOrDefault(OD_SERVER_PORT, "4430")
	settings.CAPath = os.Getenv(OD_SERVER_CA)
	settings.ServerCertChain = os.Getenv(OD_SERVER_CERT)
	settings.ServerKey = os.Getenv(OD_SERVER_KEY)
	settings.MasterKey = os.Getenv(OD_ENCRYPT_MASTERKEY)

	if os.Getenv(OD_ENCRYPT_MASTERKEY) == "" {
		log.Fatal("You must set OD_ENCRYPT_MASTERKEY to start odrive")
	}

	// Defaults
	settings.ListenBind = "0.0.0.0"
	settings.UseTLS = opts.UseTLS
	settings.RequireClientCert = true
	settings.MinimumVersion = opts.TLSMinimumVersion
	settings.AclImpersonationWhitelist = confFile.Whitelisted
	settings.CipherSuites = opts.Ciphers
	settings.PathToStaticFiles = opts.StaticRootPath
	settings.PathToTemplateFiles = opts.TemplateDir

	// TODO fill in unset values with config file values.

	return settings
}

// NewZKSettingsFromEnv inspects the environment and returns a AACConfiguration.
func NewZKSettingsFromEnv(confFile ConfigFile, opts CommandLineOpts) ZKSettings {

	var conf ZKSettings
	conf.Address = getEnvOrDefault(OD_ZK_URL, "zk:2181")
	//Notice that https is not in this yet, as odrive might register more than just https
	conf.BasepathOdrive = getEnvOrDefault(OD_ZK_ANNOUNCE, "/cte/service/object-drive/1.0")
	conf.IP = getEnvOrDefault(OD_ZK_MYIP, globalconfig.MyIP)
	serverPort := getEnvOrDefault(OD_SERVER_PORT, "4430")
	conf.Port = getEnvOrDefault(OD_ZK_MYPORT, serverPort)

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
	db.SetMaxIdleConns(int(getEnvOrDefaultInt("OD_DB_MAXIDLECONNS", 10)))
	db.SetMaxOpenConns(int(getEnvOrDefaultInt("OD_DB_MAXOPENCONNS", 10)))
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
	if len(getEnvOrDefault(variable, "")) == 0 {
		log.Printf("WARNING: %s not set.\n", variable)
	}
}

func getEnvOrDefault(name, defaultValue string) string {
	envVal := os.Getenv(name)
	if len(envVal) == 0 {
		return defaultValue
	}
	return envVal
}

func getEnvOrDefaultInt(envVar string, defaultVal int64) int64 {
	if parsed, err := strconv.ParseInt(os.Getenv(envVar), 10, 64); err == nil {
		return parsed
	}
	return defaultVal
}

func getEnvOrDefaultFloat(envVar string, defaultVal float64) float64 {
	if parsed, err := strconv.ParseFloat(os.Getenv(envVar), 64); err == nil {
		return parsed
	}
	return defaultVal
}

func getEnvOrDefaultSplitStringSlice(envVar string, defaultVal []string) []string {
	fromEnv := os.Getenv(envVar)
	if fromEnv == "" {
		return defaultVal
	}
	splitted := strings.Split(os.Getenv(envVar), ",")
	return splitted
}

// CheckAWSEnvironmentVars prevents the server from starting if appropriate vars
// are not set.
func CheckAWSEnvironmentVars(logger zap.Logger) {
	// Variables for the environment can be provided as either the native AWS_REGION, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
	// or be prefixed with the common "OD_" as in OD_AWS_REGION, OD_AWS_ACCESS_KEY_ID, and OD_AWS_SECRET_ACCESS_KEY
	// Environment variables will be normalized to the AWS_ variants to facilitate internal library calls
	region := globalconfig.GetEnvOrDefault(OD_AWS_REGION, globalconfig.GetEnvOrDefault("AWS_REGION", ""))
	if len(region) > 0 {
		os.Setenv("AWS_REGION", region)
	}
	accessKeyID := globalconfig.GetEnvOrDefault(OD_AWS_ACCESS_KEY_ID, globalconfig.GetEnvOrDefault("AWS_ACCESS_KEY_ID", ""))
	if len(accessKeyID) > 0 {
		os.Setenv("AWS_ACCESS_KEY_ID", accessKeyID)
	}
	secretKey := globalconfig.GetEnvOrDefault(OD_AWS_SECRET_ACCESS_KEY, globalconfig.GetEnvOrDefault("AWS_SECRET_ACCESS_KEY", ""))
	if len(secretKey) > 0 {
		os.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
	}
	// If the region is not set, then fail
	if region == "" {
		logger.Fatal("Fatal Error: Environment variable AWS_REGION must be set.")
	}
	return
}

// S3Config stores created config for S3
type S3Config struct {
	S3Session *session.Session
}

// CWConfig config stores config for cloudwatch
type CWConfig struct {
	CWSession          *session.Session
	SleepTimeInSeconds int
	Name               string
}

// NewAWSSessionForS3 is the s3 session
func NewAWSSessionForS3(logger zap.Logger) *S3Config {
	ret := &S3Config{}
	ret.S3Session = newAWSSession(OD_AWS_ENDPOINT, logger)
	return ret
}

// NewAWSSessionForCW is the cw session
func NewAWSSessionForCW(logger zap.Logger) *CWConfig {
	ret := &CWConfig{}
	ret.CWSession = newAWSSession(OD_AWS_CLOUDWATCH_ENDPOINT, logger)
	ret.SleepTimeInSeconds = globalconfig.GetEnvOrDefaultInt(OD_AWS_CLOUDWATCH_INTERVAL, 300)
	ret.Name = globalconfig.GetEnvOrDefault(OD_AWS_CLOUDWATCH_NAME, "host")
	return ret
}

// NewAWSSession instantiates a connection to AWS.
func newAWSSession(service string, logger zap.Logger) *session.Session {

	CheckAWSEnvironmentVars(logger)

	region := os.Getenv("AWS_REGION")
	endpoint := os.Getenv(service)

	// See if AWS creds in environment
	accessKeyID := globalconfig.GetEnvOrDefault(OD_AWS_ACCESS_KEY_ID, globalconfig.GetEnvOrDefault("AWS_ACCESS_KEY_ID", ""))
	secretKey := globalconfig.GetEnvOrDefault(OD_AWS_SECRET_ACCESS_KEY, globalconfig.GetEnvOrDefault("AWS_SECRET_ACCESS_KEY", ""))
	if len(accessKeyID) > 0 && len(secretKey) > 0 {
		logger.Info("aws.credentials", zap.String("provider", "environment variables"))
		var sessionConfig *aws.Config
		if len(endpoint) == 0 {
			sessionConfig = &aws.Config{
				Credentials: credentials.NewEnvCredentials(),
				Region:      aws.String(region),
			}
		} else {
			sessionConfig = &aws.Config{
				Credentials: credentials.NewEnvCredentials(),
				Region:      aws.String(region),
				Endpoint:    aws.String(endpoint),
			}
		}
		//sessionConfig = sessionConfig.WithLogLevel(aws.LogDebugWithHTTPBody).WithDisableComputeChecksums(false)
		return session.New(sessionConfig)
	}
	// Do as IAM
	logger.Info("aws.credentials", zap.String("provider", "iam role"))
	sessionConfig := &aws.Config{
		Region:   aws.String(region),
		Endpoint: aws.String(endpoint),
	}
	//sessionConfig = sessionConfig.WithLogLevel(aws.LogDebugWithHTTPBody).WithDisableComputeChecksums(false)
	return session.New(sessionConfig)
}
