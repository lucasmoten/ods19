package config

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

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
	DefaultBucket   = getEnvOrDefault("OD_AWS_S3_BUCKET", "")
)

// AppConfiguration is a structure that defines the known configuration format
// for this application.
type AppConfiguration struct {
	DatabaseConnection DatabaseConfiguration       `yaml:"database"`
	ServerSettings     ServerSettingsConfiguration `yaml:"server"`
	AACSettings        AACConfiguration            `yaml:"aac"`
	CacheSettings      S3CiphertextCacheOpts       `yaml:"disk_cache"`
	ZK                 ZKSettings                  `yaml:"zk"`
	EventQueue         EventQueueConfiguration     `yaml:"event_queue"`
	Whitelist          []string                    `yaml:"whitelist"`
}

// AACConfiguration holds data required for an AAC client.
type AACConfiguration struct {
	CAPath               string `yaml:"trust"`
	ClientCert           string `yaml:"cert"`
	ClientKey            string `yaml:"key"`
	HostName             string `yaml:"hostname"`
	Port                 string `yaml:"port"`
	AACAnnouncementPoint string `yaml:"zk_path"`
	// ZKAddrs can be set to discover AAC from a non-default Zookeeper cluster.
	ZKAddrs []string `yaml:"zk_addrs"`
}

// CommandLineOpts holds command line options so they can be passed as a param.
type CommandLineOpts struct {
	Ciphers           []string
	UseTLS            bool
	StaticRootPath    string
	TemplateDir       string
	TLSMinimumVersion string
	Conf              string
	Whitelist         []string
}

// DatabaseConfiguration is a structure that defines the attributes
// needed for setting up database connection
type DatabaseConfiguration struct {
	Driver     string `yaml:"driver"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	Protocol   string `yaml:"protocol"`
	Host       string `yaml:"host"`
	Port       string `yaml:"port"`
	Schema     string `yaml:"schema"`
	Params     string `yaml:"conn_params"`
	UseTLS     bool   `yaml:"use_tls"`
	SkipVerify bool   `yaml:"insecure_skip_veriry"`
	CAPath     string `yaml:"trust"`
	ClientCert string `yaml:"cert"`
	ClientKey  string `yaml:"key"`
}

// EventQueueConfiguration configures publishing to the Kakfa event queue.
type EventQueueConfiguration struct {
	KafkaAddrs []string `yaml:"kafka_addrs"`
	ZKAddrs    []string `yaml:"zk_addrs"`
}

// S3CiphertextCacheOpts describes our current disk cache configuration.
type S3CiphertextCacheOpts struct {
	Root          string  `yaml:"root_dir"`
	Partition     string  `yaml:"partition"`
	LowWatermark  float64 `yaml:"low_watermark"`
	HighWatermark float64 `yaml:"high_waterwark"`
	EvictAge      int64   `yaml:"evict_age"`
	WalkSleep     int64   `yaml:"walk_sleep"`
	MasterKey     string  `yaml:"master_key"`
}

// ServerSettingsConfiguration holds the attributes needed for
// setting up an AppServer listener.
type ServerSettingsConfiguration struct {
	BasePath                  string   `yaml:"base_path"`
	ListenPort                string   `yaml:"port"`
	ListenBind                string   `yaml:"bind"`
	UseTLS                    bool     `yaml:"use_tls"`
	CAPath                    string   `yaml:"trust"`
	ServerCertChain           string   `yaml:"cert"`
	ServerKey                 string   `yaml:"key"`
	RequireClientCert         bool     `yaml:"require_client_cert"`
	CipherSuites              []string `yaml:"ciphers"`
	MinimumVersion            string   `yaml:"min_version"`
	AclImpersonationWhitelist []string `yaml:"acl_whitelist"`
	PathToStaticFiles         string   `yaml:"static_root"`
	PathToTemplateFiles       string   `yaml:"template_root"`
}

// ZKSettings holds the data required to communicate with default Zookeeper.
type ZKSettings struct {
	IP             string `yaml:"ip"`
	Port           string `yaml:"port"`
	Address        string `yaml:"address"`
	BasepathOdrive string `yaml:"register_odrive_as"`
	Timeout        int64  `yaml:"timeout"`
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
	cacheSettings := NewS3CiphertextCacheOpts(confFile, opts)
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
func NewAACSettingsFromEnv(confFile AppConfiguration, opts CommandLineOpts) AACConfiguration {

	var conf AACConfiguration

	conf.CAPath = cascade(OD_AAC_CA, confFile.AACSettings.CAPath, "")
	conf.ClientCert = cascade(OD_AAC_CERT, confFile.AACSettings.ClientCert, "")
	conf.ClientKey = cascade(OD_AAC_KEY, confFile.AACSettings.ClientKey, "")

	// HostName and Port should only be set if we want to directly connect to AAC and not use service discovery.
	conf.HostName = cascade(OD_AAC_HOST, confFile.AACSettings.HostName, "")
	conf.Port = cascade(OD_AAC_PORT, confFile.AACSettings.Port, "")

	conf.AACAnnouncementPoint = cascade(OD_ZK_AAC, confFile.AACSettings.AACAnnouncementPoint, "/cte/service/aac/1.0/thrift")

	// If ZKAddrs is set, we attempt to discover AAC from a non-default Zookeeper cluster.
	var empty []string
	conf.ZKAddrs = CascadeStringSlice(OD_AAC_ZK_ADDRS, confFile.AACSettings.ZKAddrs, empty)

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

	// Whitelist (Optional. Usually provided via yaml configuration.)
	whitelist := clictx.StringSlice("whitelist")

	return CommandLineOpts{
		Ciphers:           ciphers,
		UseTLS:            useTLS,
		Conf:              confPath,
		StaticRootPath:    staticRootPath,
		TemplateDir:       templateDir,
		TLSMinimumVersion: tlsMinimumVersion,
		Whitelist:         whitelist,
	}
}

// NewDatabaseConfigFromEnv inspects the environment and returns a DatabaseConfiguration.
func NewDatabaseConfigFromEnv(confFile AppConfiguration, opts CommandLineOpts) DatabaseConfiguration {

	var dbConf DatabaseConfiguration

	// From environment
	dbConf.Username = cascade(OD_DB_USERNAME, confFile.DatabaseConnection.Username, "")
	dbConf.Password = cascade(OD_DB_PASSWORD, confFile.DatabaseConnection.Password, "")
	dbConf.Host = cascade(OD_DB_HOST, confFile.DatabaseConnection.Host, "")
	dbConf.Port = cascade(OD_DB_PORT, confFile.DatabaseConnection.Port, "3306")
	dbConf.Schema = cascade(OD_DB_SCHEMA, confFile.DatabaseConnection.Schema, "metadatadb")
	dbConf.CAPath = cascade(OD_DB_CA, confFile.DatabaseConnection.CAPath, "")
	dbConf.ClientCert = cascade(OD_DB_CERT, confFile.DatabaseConnection.ClientCert, "")
	dbConf.ClientKey = cascade(OD_DB_KEY, confFile.DatabaseConnection.ClientKey, "")
	dbConf.Params = cascade(OD_DB_CONN_PARAMS, confFile.DatabaseConnection.Params, "parseTime=true&collation=utf8_unicode_ci")

	// Defaults
	dbConf.Protocol = "tcp"
	dbConf.Driver = defaultDBDriver
	dbConf.UseTLS = true
	dbConf.SkipVerify = true

	return dbConf
}

// NewEventQueueConfiguration reades the environment to provide the configuration for the Kafka event queue.
func NewEventQueueConfiguration(confFile AppConfiguration, opts CommandLineOpts) EventQueueConfiguration {
	var eqc EventQueueConfiguration
	var empty []string
	eqc.KafkaAddrs = CascadeStringSlice(OD_EVENT_KAFKA_ADDRS, confFile.EventQueue.KafkaAddrs, empty)
	eqc.ZKAddrs = CascadeStringSlice(OD_EVENT_ZK_ADDRS, confFile.EventQueue.ZKAddrs, empty)
	return eqc
}

// NewS3CiphertextCacheOpts reads the environment to provide the configuration options for
// S3CiphertextCache.
func NewS3CiphertextCacheOpts(confFile AppConfiguration, opts CommandLineOpts) S3CiphertextCacheOpts {
	settings := S3CiphertextCacheOpts{
		Root:          cascade(OD_CACHE_ROOT, confFile.CacheSettings.Root, "."),
		Partition:     cascade(OD_CACHE_PARTITION, confFile.CacheSettings.Partition, "cache"),
		LowWatermark:  cascadeFloat(OD_CACHE_LOWWATERMARK, confFile.CacheSettings.LowWatermark, .50),
		HighWatermark: cascadeFloat(OD_CACHE_HIGHWATERMARK, confFile.CacheSettings.HighWatermark, .75),
		EvictAge:      cascadeInt(OD_CACHE_EVICTAGE, confFile.CacheSettings.EvictAge, 300),
		WalkSleep:     cascadeInt(OD_CACHE_WALKSLEEP, confFile.CacheSettings.WalkSleep, 30),
		MasterKey:     cascade(OD_ENCRYPT_MASTERKEY, confFile.CacheSettings.MasterKey, ""),
	}
	//Note: masterKey is a singleton value now.  But there will need to be one per OD_CACHE_PARTITION now
	if settings.MasterKey == "" {
		log.Fatal("You must set master encryption key with OD_ENCRYPT_MASTERKEY to start odrive")
	}
	return settings
}

// NewServerSettingsFromEnv inspects the environment and returns a ServerSettingsConfiguration.
func NewServerSettingsFromEnv(confFile AppConfiguration, opts CommandLineOpts) ServerSettingsConfiguration {

	var settings ServerSettingsConfiguration

	// From env
	settings.BasePath = cascade(OD_SERVER_BASEPATH, confFile.ServerSettings.BasePath, "/services/object-drive/1.0")
	settings.ListenPort = cascade(OD_SERVER_PORT, confFile.ServerSettings.ListenPort, "4430")
	settings.CAPath = cascade(OD_SERVER_CA, confFile.ServerSettings.CAPath, "")
	settings.ServerCertChain = cascade(OD_SERVER_CERT, confFile.ServerSettings.ServerCertChain, "")
	settings.ServerKey = cascade(OD_SERVER_KEY, confFile.ServerSettings.ServerKey, "")

	// We only use conf.yml and cli opts for the ACL whitelist
	settings.AclImpersonationWhitelist = selectNonEmptyStringSlice(opts.Whitelist, confFile.ServerSettings.AclImpersonationWhitelist, confFile.Whitelist)

	// Defaults
	settings.ListenBind = "0.0.0.0"
	settings.UseTLS = opts.UseTLS
	settings.RequireClientCert = true
	settings.MinimumVersion = opts.TLSMinimumVersion
	settings.CipherSuites = opts.Ciphers
	settings.PathToStaticFiles = opts.StaticRootPath
	settings.PathToTemplateFiles = opts.TemplateDir

	return settings
}

// NewZKSettingsFromEnv inspects the environment and returns a AACConfiguration.
func NewZKSettingsFromEnv(confFile AppConfiguration, opts CommandLineOpts) ZKSettings {

	var conf ZKSettings
	conf.Address = cascade(OD_ZK_URL, confFile.ZK.Address, "zk:2181")
	conf.BasepathOdrive = cascade(OD_ZK_ANNOUNCE, confFile.ZK.BasepathOdrive, "/cte/service/object-drive/1.0")
	conf.IP = cascade(OD_ZK_MYIP, confFile.ZK.IP, resolveIP())
	conf.Port = cascade(OD_ZK_MYPORT, confFile.ZK.Port, "4430")
	conf.Timeout = cascadeInt(OD_ZK_TIMEOUT, confFile.ZK.Timeout, 5)

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
	logDSN := dbDSN
	logDSN = strings.Replace(logDSN, r.Password, "{password}", -1)
	logDSN = strings.Replace(logDSN, r.Username, "{username}", -1)
	logger.Info("Using this connection string", zap.String("dbdsn", logDSN))
	return dbDSN
}

// buildTLSConfig prepares a standard go tls.Config with RootCAs and client
// Certificates for communicating with the database securely.
func (conf *DatabaseConfiguration) buildTLSConfig() tls.Config {

	// Root Certificate pool
	// The set of root certificate authorities that this client will use when
	// verifying the server certificate indicated as the identity of the
	// server this config will be used to connect to.
	rootCAsCertPool := buildCertPoolFromPath(conf.CAPath, "for client")

	// Client public and private certificate
	if len(conf.ClientCert) == 0 || len(conf.ClientKey) == 0 {
		return tls.Config{
			RootCAs:            rootCAsCertPool,
			ServerName:         conf.Host,
			InsecureSkipVerify: conf.SkipVerify,
		}
	}
	clientCert := buildx509Identity(conf.ClientCert, conf.ClientKey)

	return tls.Config{
		RootCAs:            rootCAsCertPool,
		Certificates:       clientCert,
		ServerName:         conf.Host,
		InsecureSkipVerify: conf.SkipVerify,
	}

}

// buildTLSConfig prepares a standard go tls.Config with trusted CAs and
// server identity certificates to listen for connecting clients
func (r *ServerSettingsConfiguration) buildTLSConfig() tls.Config {
	return buildServerTLSConfig(r.CAPath, r.ServerCertChain, r.ServerKey, r.RequireClientCert, r.CipherSuites, r.MinimumVersion)
}

func cascade(fromEnv, fromFile, defaultVal string) string {
	if envVal := os.Getenv(fromEnv); envVal != "" {
		return envVal
	}
	if fromFile != "" {
		return fromFile
	}
	return defaultVal
}

func cascadeFloat(fromEnv string, fromFile, defaultVal float64) float64 {
	if parsed, err := strconv.ParseFloat(os.Getenv(fromEnv), 64); err == nil {
		return parsed
	}
	if fromFile != 0.0 {
		return fromFile
	}
	return defaultVal
}

func cascadeInt(fromEnv string, fromFile, defaultVal int64) int64 {
	if parsed, err := strconv.ParseInt(os.Getenv(fromEnv), 10, 64); err == nil {
		return parsed
	}
	if fromFile != 0 {
		return fromFile
	}
	return defaultVal
}

// CascadeStringSlice will select a configuration slice from a splitted env var,
// the config file, or a default slice.
func CascadeStringSlice(fromEnv string, fromFile, defaultVal []string) []string {

	if splitted := strings.Split(os.Getenv(fromEnv), ","); len(splitted) > 0 {
		if splitted[0] != "" {
			return splitted
		}
	}
	if len(fromFile) > 0 {
		if fromFile[0] != "" {
			return fromFile
		}
	}
	return defaultVal
}

func selectNonEmptyStringSlice(slices ...[]string) []string {
	for _, sl := range slices {
		if len(sl) > 0 {
			return sl
		}
	}
	sl := make([]string, 0)
	return sl
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

// AWSConfig for getting a session
type AWSConfig struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
}

// S3Config stores created config for S3
type S3Config struct {
	AWSConfig *AWSConfig
}

// CWConfig config stores config for cloudwatch
type CWConfig struct {
	AWSConfig          *AWSConfig
	SleepTimeInSeconds int
	Name               string
}

// AutoScalingConfig session for the queueing service
type AutoScalingConfig struct {
	AWSConfigSQS         *AWSConfig
	AWSConfigASG         *AWSConfig
	QueueName            string
	AutoScalingGroupName string
	EC2InstanceID        string
	PollingInterval      int64
}

// NewAWSConfig is default values for AWS config
func NewAWSConfig(endpoint string) *AWSConfig {
	ret := &AWSConfig{}
	//Per service
	ret.Endpoint = os.Getenv(endpoint)
	//Same for all
	ret.Region = getEnvOrDefault(OD_AWS_REGION, getEnvOrDefault("AWS_REGION", ""))
	ret.AccessKeyID = getEnvOrDefault(OD_AWS_ACCESS_KEY_ID, getEnvOrDefault("AWS_ACCESS_KEY_ID", ""))
	ret.SecretAccessKey = getEnvOrDefault(OD_AWS_SECRET_ACCESS_KEY, getEnvOrDefault("AWS_SECRET_ACCESS_KEY", ""))
	return ret
}

// NewS3Config is the s3 session
func NewS3Config() *S3Config {
	ret := &S3Config{}
	name := OD_AWS_S3_ENDPOINT
	s3Endpoint := getEnvOrDefault(OD_AWS_S3_ENDPOINT, "")
	if s3Endpoint == "" {
		s3EndpointOld := getEnvOrDefault("OD_AWS_ENDPOINT", "")
		if s3EndpointOld != "" {
			s3Endpoint = s3EndpointOld
			name = "OD_AWS_ENDPOINT"
			logger.Error("OD_AWS_ENDPOINT must be renamed to OD_AWS_S3_ENDPOINT in your env.sh")
		}
	}
	ret.AWSConfig = NewAWSConfig(name)
	return ret
}

// NewCWConfig is the cw session
func NewCWConfig() *CWConfig {
	ret := &CWConfig{}
	ret.AWSConfig = NewAWSConfig(OD_AWS_CLOUDWATCH_ENDPOINT)
	ret.SleepTimeInSeconds = int(getEnvOrDefaultInt(OD_AWS_CLOUDWATCH_INTERVAL, 300))
	ret.Name = getEnvOrDefault(OD_AWS_CLOUDWATCH_NAME, "")
	if ret.Name == "" {
		return nil
	}
	return ret
}

// NewAutoScalingConfig is the sqs session
func NewAutoScalingConfig() *AutoScalingConfig {
	ret := &AutoScalingConfig{}
	ret.AWSConfigSQS = NewAWSConfig(OD_AWS_SQS_ENDPOINT)
	ret.AWSConfigASG = NewAWSConfig(OD_AWS_ASG_ENDPOINT)
	ret.EC2InstanceID = getEnvOrDefault(OD_AWS_ASG_EC2, "")
	ret.AutoScalingGroupName = getEnvOrDefault(OD_AWS_ASG_NAME, "")
	ret.QueueName = getEnvOrDefault(OD_AWS_SQS_NAME, "")
	ret.PollingInterval = getEnvOrDefaultInt(OD_AWS_SQS_INTERVAL, 60)
	return ret
}

func resolveIP() string {
	hostname, err := os.Hostname()
	if err != nil {
		logger.Error("error looking up hostname")
		return ""
	}
	if len(hostname) > 0 {
		myIPs, err := net.LookupIP(hostname)
		if err != nil {
			logger.Error("could not get a set of ips for our hostname")
			return ""
		}
		if len(myIPs) > 0 {
			for a := range myIPs {
				if myIPs[a].To4() != nil {
					return myIPs[a].String()

				}
			}
		} else {
			logger.Error("We did not find our ip")
		}
	} else {
		logger.Error("We could not find our hostname")
	}
	return ""
}
