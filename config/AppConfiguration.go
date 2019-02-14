package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	odrivecrypto "bitbucket.di2e.net/dime/object-drive-server/crypto"
	"bitbucket.di2e.net/dime/object-drive-server/util"
	"bitbucket.di2e.net/greymatter/gov-go/gov/encryptor"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/urfave/cli"
	"go.uber.org/zap"
)

// DBDRIVERMYSQL provides identifer for MySQL database driver.
const DBDRIVERMYSQL = "mysql"

var (
	defaultDBHost = "metadatadb"
	defaultDBPort = "3306"
	// DefaultBucket is the name of the S3 storage bucket to use for encrypted files
	DefaultBucket = getEnvOrDefault(OD_AWS_S3_BUCKET, "")
)

var empty []string

// AppConfiguration is a structure that defines the known configuration format
// for this application.
type AppConfiguration struct {
	DatabaseConnection DatabaseConfiguration       `yaml:"database"`
	ServerSettings     ServerSettingsConfiguration `yaml:"server"`
	AACSettings        AACConfiguration            `yaml:"aac"`
	CacheSettings      DiskCacheOpts               `yaml:"disk_cache"`
	ZK                 ZKSettings                  `yaml:"zk"`
	EventQueue         EventQueueConfiguration     `yaml:"event_queue"`
}

// AACConfiguration holds data required for an AAC client. Host and port are often
// discovered dynamically via Zookeeper.
type AACConfiguration struct {
	// CAPath is the path to a PEM encoded certificate that the AAC trusts.
	CAPath string `yaml:"trust"`
	// ClientCert is the path to a PEM encoded certificate we present to the AAC.
	ClientCert string `yaml:"cert"`
	// ClientKey is the path to a PEM encoded private key.
	ClientKey string `yaml:"key"`
	// CommonName is the name we expect all AAC servers to have when enforcing certificate validation
	CommonName string `yaml:"common_name"`
	// Healthcheck is an ACM expected to pass validation
	HealthCheck string `yaml:"healthcheck"`
	// Hostname is the hostname of the AAC service
	HostName string `yaml:"hostname"`
	// Port is the port AAC is listening on.
	Port string `yaml:"port"`
	// AACAnnouncementPoint is a path to inspect in Zookeeper, if we are using
	// service discovery to connect to AAC.
	AACAnnouncementPoint string `yaml:"zk_path"`
	// ZKAddrs can be set to discover AAC from a non-default Zookeeper cluster.
	ZKAddrs []string `yaml:"zk_addrs"`
	// WarumpTime is the number of seconds to wait for ZK before checking health of AAC
	WarmupTime int64 `yaml:"warmup_time"`
	// RecheckTime is the interval seconds between AAC health status checks
	RecheckTime int64 `yaml:"recheck_time"`
}

// CommandLineOpts holds command line options parsed on application start. This
// object is passed to many higher level constructors, so that command line params
// can override certain configurations.
type CommandLineOpts struct {
	// Ciphers is a list of TLS ciphers we are willing to accept.
	Ciphers []string
	// StaticRootPath is a path to the static web assets directory.
	StaticRootPath string
	// TemplateDir is the path to Go templates directory.
	TemplateDir string
	// TLSMinimumVersion is the minimum TLS version we accept.
	TLSMinimumVersion string
	// Conf is a path to our YAML configuration file.
	Conf string
	// Whitelist holds ACL whitelist entries passed at the command line.
	Whitelist []string
}

// DatabaseConfiguration is a structure that defines the attributes
// needed for setting up database connection
type DatabaseConfiguration struct {
	// Driver specifies the database driver. Only "mysql" is supported.
	Driver string `yaml:"driver"`
	// Username is the database username.
	Username string `yaml:"username"`
	// Password is the database password. If the configuration is intended
	// to execute DDL, a user with write permissions is required.
	Password string `yaml:"password"`
	// Protocol specifies the network protocol. Only "tcp" is supported.
	Protocol string `yaml:"protocol"`
	// Host is the database hostname.
	Host string `yaml:"host"`
	// Port is the database port. Commonly 3306 for MySQL.
	Port string `yaml:"port"`
	// Schema is the database name to connect to. A single server can host
	// many logical schemas. The object drive default is "metadatadb".
	Schema string `yaml:"schema"`
	// Params are custom connection params injected into the DSN. These
	// will vary depending on your server's configuration.
	Params string `yaml:"params"`
	// UseTLS determines whether you should connect to the database with TLS.
	// Defaults to true
	UseTLS bool
	// UseTLSString allows converting string value to bool from configuration
	UseTLSString string `yaml:"use_tls"`
	// SkipVerify controls whether the hostname of an SSL peer is verified.
	// Defaults to false
	SkipVerify bool
	// SkipVerifyString allows converting string value to bool from configuration
	SkipVerifyString string `yaml:"insecure_skip_verify"`
	// CAPath is the path to a PEM encoded certificate. For connecting to
	// some test databases this might be the only SSL asset required, if
	// 2-way SSL is not enforced.
	CAPath string `yaml:"trust"`
	// ClientCert is the path to our PEM encoded client certificate.
	ClientCert string `yaml:"cert"`
	// ClientKey is the path to our PEM encoded client key.
	ClientKey string `yaml:"key"`
	// DeadlockRetryCounter is the number of times to retry statements in a
	// transaction that are failing due to a deadlock
	DeadlockRetryCounter int64 `yaml:"deadlock_retrycounter"`
	// DeadlockRetryDelay is the time to wait in milliseconds before retrying
	// a statement in a transaction that is failing due to a deadlock
	DeadlockRetryDelay int64 `yaml:"deadlock_retrydelay"`
	// MaxIdleConns is the maximum number of idle connections in the connection pool
	MaxIdleConns int64 `yaml:"max_idle_conns"`
	// MaxOpenConns is the maximum number of open connections to the database
	MaxOpenConns int64 `yaml:"max_open_conns"`
	// MaxConnLifetime is the maximum lifetime, in seconds that a connection may be reused
	MaxConnLifetime int64 `yaml:"max_conn_lifetime"`
}

// EventQueueConfiguration configures publishing to the Kafka event queue.
type EventQueueConfiguration struct {
	// KafkaAddrs is a list of host:port pairs of Kafka brokers. If provided,
	// a direct connection to the brokers is established.
	KafkaAddrs []string `yaml:"kafka_addrs"`
	// ZKAddrs is a list of host:port pairs of ZK nodes. A common
	// architecture is to have a ZK cluster entirely dedicated to Kafka. This
	// config option handles that scenario.
	ZKAddrs []string `yaml:"zk_addrs"`
	// PublishSuccessActions, if provided, specifies the types of success actions
	// to publish to Kafka. If empty, all success actions are published.
	PublishSuccessActions []string `yaml:"publish_success_actions"`
	// PublishFailureActions, if provided, specifies the types of success actions
	// to publish to Kafka. If empty, all failure actions are published.
	PublishFailureActions []string `yaml:"publish_failure_actions"`
	// Topic denotes the name of the topic to publish events to in Kafka.
	Topic string `yaml:"topic"`
}

// DiskCacheOpts describes our current disk cache configuration.
type DiskCacheOpts struct {
	// Root specifies an absolute or relative path to set the root directory of
	// the local cache. All uploads are cached on disk. This directory must be
	// writable by the server process.
	Root string `yaml:"root_dir"`
	// Partition is an optional path prefix for objects written to S3.
	Partition string `yaml:"partition"`
	// LowWatermark denotes a percentage of local storage that must be used before
	// the cache eviction routine will operate on items in the cache.
	LowWatermark float64 `yaml:"low_watermark"`
	// HighWatermark denotes a  percentage of local storage. If exceeded, cache
	// items older than EvictAge will be eligible for purge.
	HighWatermark float64 `yaml:"high_waterwark"`
	// EvictAge denotes the minimum age, in seconds, a file in cache before it
	// is eligible for purge from the cache to free up space.
	EvictAge int64 `yaml:"evict_age"`
	// Walk sleep sets frequency, in seconds, for which all files in the cache are
	// examined to determine if they should be purged.
	WalkSleep int64 `yaml:"walk_sleep"`
	// MasterKey is the master encryption key. This must be kept safe. Losing this
	// key will make encrypted data unrecoverable.
	MasterKey string `yaml:"masterkey"`
	// ChunkSize specifies a memory block size to send to S3 when durably persisting
	// cached files.
	ChunkSize int64 `yaml:"chunk_size"`
}

// ServerSettingsConfiguration holds the attributes needed for
// setting up an AppServer listener.
type ServerSettingsConfiguration struct {
	// EncryptEnabled indicates indicates whether or not file streams will be encrypted at rest.
	EncryptEnabled       bool
	EncryptEnabledString string `yaml:"encrypt_enabled"`
	//EncryptableFunctions contains the set of functions to be used for encryption.  When EncryptEnabled is true
	//they encrypt, otherwise they do not. This structure is used throughout
	//the application to support the ability to turn file encryption off and on.
	EncryptableFunctions EncryptableFunctions
	// BasePath is the root URL for static assets. Only used for debug UI.
	BasePath string `yaml:"base_path"`
	// ListenPort is the port the server listens on. Default is 4430.
	ListenPort string `yaml:"port"`
	// ListenBind is the address to bind to. Hardcoded to 0.0.0.0
	ListenBind string `yaml:"bind"`
	// CAPath is the path to a PEM encoded certificate of our CA.
	CAPath string `yaml:"trust"`
	// ServerCertChain is the path to our server's PEM encoded cert.
	ServerCertChain string `yaml:"cert"`
	// ServerKey is the path to our server's PEM encoded key.
	ServerKey string `yaml:"key"`
	// CipherSuites specifies the ciphers we will accept. Common values are
	// TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 and TLS_RSA_WITH_AES_128_CBC_SHA
	CipherSuites []string `yaml:"ciphers"`
	// MinimumVersion is the minimum TLS protocol version we support. Currently TLS 1.2
	MinimumVersion string `yaml:"min_version"`
	// ACLImpersonationWhitelist is a list of Distinguished Names. If a client
	// (usually another machine) is on this list, it may pass us another DN in
	// an HTTP header, and "impersonate" that identity. The common use case
	// is for an edge proxy (such as nginx) to pass through requests from users
	// outside the network. This configuration option must be specified in YAML
	// or on the command line.
	ACLImpersonationWhitelist []string `yaml:"acl_whitelist"`
	// PathToStaticFiles is a location on disk where static assets are stored.
	PathToStaticFiles string `yaml:"static_root"`
	// PathToTemplateFiles is a location on disk where Go templates are stored.
	PathToTemplateFiles string `yaml:"template_root"`
	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled. If IdleTimeout
	// is zero, the value of ReadTimeout is used. If both are
	// zero, ReadHeaderTimeout is used.
	IdleTimeout int64 `yaml:"timeout_idle"`
	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body.
	//
	// Because ReadTimeout does not let Handlers make per-request
	// decisions on each request body's acceptable deadline or
	// upload rate, most users will prefer to use
	// ReadHeaderTimeout. It is valid to use them both.
	ReadTimeout int64 `yaml:"timeout_read"`
	// ReadHeaderTimeout is the amount of time allowed to read
	// request headers. The connection's read deadline is reset
	// after reading the headers and the Handler can decide what
	// is considered too slow for the body.
	ReadHeaderTimeout int64 `yaml:"timeout_read_header"`
	// WriteTimeout is the maximum duration before timing out
	// writes of the response. It is reset whenever a new
	// request's header is read. Like ReadTimeout, it does not
	// let Handlers make decisions on a per-request basis.
	WriteTimeout int64 `yaml:"timeout_write"`
	// Version is set at runtime based on compile time flags
	Version string
}

// ZKSettings holds the data required to communicate with default Zookeeper.
type ZKSettings struct {
	// The IP address of our server, as reported to Zookeeper. If configured,
	// we override the value detected as the server's IP address on startup.
	IP string `yaml:"ip"`
	// The Port of our server, announced to Zookeeper.
	Port string `yaml:"port"`
	// Address is the address of the Zookeeper cluster we attempt to connect to.
	Address string `yaml:"address"`
	// BasepathOdrive is a Zookeeper path. We register ourselves as an ephemeral
	// node under this path.
	BasepathOdrive string `yaml:"register_odrive_as"`
	// Timeout configures a timeout for the Zookeeper driver in seconds.
	Timeout int64 `yaml:"timeout"`
	// RetryDelay configures the number of seconds between retry attempts to connect
	RetryDelay int64 `yaml:"retrydelay"`
	// RecheckTime is the interval seconds between ZK health status checks
	RecheckTime int64 `yaml:"recheck_time"`
}

// NewAppConfiguration loads the configuration from the different sources in the environment.
// If multiple configuration sources can be used, the order of precedence is: env var overrides
// config file.
func NewAppConfiguration(opts CommandLineOpts) AppConfiguration {

	confFile, err := LoadYAMLConfig(opts.Conf)
	if err != nil {
		fmt.Printf("Error loading yaml configuration at path %v: %v\n", confFile, err)
		os.Exit(1)
	}

	dbConf := NewDatabaseConfigFromEnv(confFile, opts)
	confFile.DatabaseConnection = dbConf
	serverSettings := newServerSettingsFromEnv(confFile, opts)
	confFile.ServerSettings = serverSettings
	aacSettings := newAACSettingsFromEnv(confFile, opts)
	confFile.AACSettings = aacSettings
	cacheSettings := newDiskCacheOpts(confFile, opts)
	confFile.CacheSettings = cacheSettings
	zkSettings := newZKSettingsFromEnv(confFile, opts)
	confFile.ZK = zkSettings
	eventQueue := newEventQueueConfiguration(confFile, opts)
	confFile.EventQueue = eventQueue

	appConf := AppConfiguration{
		AACSettings:        aacSettings,
		CacheSettings:      cacheSettings,
		DatabaseConnection: dbConf,
		EventQueue:         eventQueue,
		ServerSettings:     serverSettings,
		ZK:                 zkSettings,
	}

	setEnvironmentFromConfiguration(appConf)

	return appConf
}

// newAACSettingsFromEnv inspects the environment and returns a AACConfiguration.
func newAACSettingsFromEnv(confFile AppConfiguration, opts CommandLineOpts) AACConfiguration {

	var conf AACConfiguration

	conf.CAPath = cascade(OD_AAC_CA, confFile.AACSettings.CAPath, "")
	conf.ClientCert = cascade(OD_AAC_CERT, confFile.AACSettings.ClientCert, "")
	conf.ClientKey = cascade(OD_AAC_KEY, confFile.AACSettings.ClientKey, "")
	conf.CommonName = cascade(OD_AAC_CN, confFile.AACSettings.CommonName, "")

	// Healthcheck
	conf.HealthCheck = cascade(OD_AAC_HEALTHCHECK, confFile.AACSettings.HealthCheck, "{\"version\":\"2.1.0\",\"classif\":\"U\"}")

	// HostName and Port should only be set if we want to directly connect to AAC and not use service discovery.
	conf.HostName = cascade(OD_AAC_HOST, confFile.AACSettings.HostName, "")
	conf.Port = cascade(OD_AAC_PORT, confFile.AACSettings.Port, "")

	conf.AACAnnouncementPoint = cascade(OD_ZK_AAC, confFile.AACSettings.AACAnnouncementPoint, "/cte/service/aac/1.2/thrift")

	// If ZKAddrs is set, we attempt to discover AAC from a non-default Zookeeper cluster.
	conf.ZKAddrs = CascadeStringSlice(OD_AAC_ZK_ADDRS, confFile.AACSettings.ZKAddrs, empty)

	// Time delays for startup and recheck interval
	conf.WarmupTime = cascadeInt(OD_AAC_WARMUP_TIME, confFile.AACSettings.WarmupTime, 20)
	conf.RecheckTime = cascadeInt(OD_AAC_RECHECK_TIME, confFile.AACSettings.RecheckTime, 30)

	return conf
}

// NewCommandLineOpts instantiates CommandLineOpts from a pointer to the parsed command line
// context. The actual parsing is handled by the cli framework.
func NewCommandLineOpts(clictx *cli.Context) CommandLineOpts {
	ciphers := clictx.StringSlice("addCipher")
	// NOTE: cli lib appends to []string that already contains the "default" value. Must trim
	// the default cipher if addCipher is passed at command line.
	if len(ciphers) > 1 {
		ciphers = ciphers[1:]
	}

	// Config file YAML is parsed elsewhere. This is just the path.
	confPath := clictx.String("conf")

	// Static Files Directory (Optional. Can be set to empty for no static files)
	staticRootPath := clictx.String("staticRoot")
	if len(staticRootPath) > 0 {
		if _, err := os.Stat(staticRootPath); os.IsNotExist(err) {
			fmt.Printf("Static Root Path %s does not exist: %v\n", staticRootPath, err)
			os.Exit(1)
		}
	}

	// Template Directory (Optional. Can be set to empty for no templates)
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
	pwd, err := MaybeDecrypt(cascade(OD_DB_PASSWORD, confFile.DatabaseConnection.Password, ""))
	if err != nil {
		log.Printf("Unable to decrypt database password: %v", err)
		os.Exit(1)
	}
	dbConf.Password = pwd
	dbConf.Host = cascade(OD_DB_HOST, confFile.DatabaseConnection.Host, "")
	dbConf.Port = cascade(OD_DB_PORT, confFile.DatabaseConnection.Port, "3306")
	dbConf.Schema = cascade(OD_DB_SCHEMA, confFile.DatabaseConnection.Schema, "metadatadb")
	dbConf.CAPath = cascade(OD_DB_CA, confFile.DatabaseConnection.CAPath, "")
	dbConf.ClientCert = cascade(OD_DB_CERT, confFile.DatabaseConnection.ClientCert, "")
	dbConf.ClientKey = cascade(OD_DB_KEY, confFile.DatabaseConnection.ClientKey, "")
	dbConf.Params = cascade(OD_DB_CONN_PARAMS, confFile.DatabaseConnection.Params, "parseTime=true&collation=utf8_unicode_ci&readTimeout=30s")
	dbConf.Protocol = cascade(OD_DB_PROTOCOL, confFile.DatabaseConnection.Protocol, "tcp")
	dbConf.Driver = cascade(OD_DB_DRIVER, confFile.DatabaseConnection.Driver, DBDRIVERMYSQL)
	dbConf.UseTLS = CascadeBoolFromString(OD_DB_USE_TLS, confFile.DatabaseConnection.UseTLSString, true)
	dbConf.SkipVerify = CascadeBoolFromString(OD_DB_SKIP_VERIFY, confFile.DatabaseConnection.SkipVerifyString, false)
	dbConf.MaxIdleConns = cascadeInt(OD_DB_MAXIDLECONNS, confFile.DatabaseConnection.MaxIdleConns, 10)
	dbConf.MaxOpenConns = cascadeInt(OD_DB_MAXOPENCONNS, confFile.DatabaseConnection.MaxOpenConns, 10)
	dbConf.MaxConnLifetime = cascadeInt(OD_DB_CONNMAXLIFETIME, confFile.DatabaseConnection.MaxConnLifetime, 30)
	dbConf.DeadlockRetryCounter = cascadeInt(OD_DB_DEADLOCK_RETRYCOUNTER, confFile.DatabaseConnection.DeadlockRetryCounter, 30)
	dbConf.DeadlockRetryDelay = cascadeInt(OD_DB_DEADLOCK_RETRYDELAYMS, confFile.DatabaseConnection.DeadlockRetryDelay, 55)

	// Sanity readTimeout
	if !strings.Contains(dbConf.Params, "readTimeout=") {
		log.Printf("WARNING: No readTimeout parameter specified in OD_DB_CONN_PARAMS or in conn_params. Setting 30s default")
		dbConf.Params = dbConf.Params + "&readTimeout=30s"
	}

	return dbConf
}

// newEventQueueConfiguration reads the environment to provide the configuration for the Kafka event queue.
func newEventQueueConfiguration(confFile AppConfiguration, opts CommandLineOpts) EventQueueConfiguration {
	var eqc EventQueueConfiguration
	eqc.KafkaAddrs = CascadeStringSlice(OD_EVENT_KAFKA_ADDRS, confFile.EventQueue.KafkaAddrs, empty)
	eqc.ZKAddrs = CascadeStringSlice(OD_EVENT_ZK_ADDRS, confFile.EventQueue.ZKAddrs, empty)
	eqc.PublishSuccessActions = CascadeStringSlice(OD_EVENT_PUBLISH_SUCCESS_ACTIONS, confFile.EventQueue.PublishSuccessActions, []string{"*"})
	eqc.PublishFailureActions = CascadeStringSlice(OD_EVENT_PUBLISH_FAILURE_ACTIONS, confFile.EventQueue.PublishFailureActions, []string{"*"})
	eqc.Topic = cascade(OD_EVENT_TOPIC, confFile.EventQueue.Topic, "odrive-event")
	return eqc
}

// GetTokenJarKey is ONLY used for startup config, it's ok to be Fatal
// when this fails.  Either supply a plaintext value, or supply a correctly encrypted value
func GetTokenJarKey() ([]byte, error) {
	rootPassword := os.Getenv(OD_TOKENJAR_PASSWORD)
	if rootPassword == "" {
		rootPassword = "BeDr0cK-Ro0t-K3y"
	}
	tokenJar := os.Getenv(OD_TOKENJAR_LOCATION)
	if tokenJar == "" {
		tokenJar = "/opt/services/object-drive-1.0/token.jar"
	}
	key, err := encryptor.KeyFromTokenJar(tokenJar, rootPassword)
	if err != nil {
		return nil, fmt.Errorf("GetTokenJarKey failed (missing or malformed token.jar): %v", err)
	}
	return key, nil
}

// MaybeDecrypt is ONLY used for startup config.  It is fatal when this fails.
func MaybeDecrypt(val string) (string, error) {
	if strings.HasPrefix(val, "ENC{") && strings.HasSuffix(val, "}") {
		key, err := GetTokenJarKey()
		if err != nil {
			return "", err
		}
		val, err = encryptor.ReplaceAll(val, key)
		if err != nil {
			return "", fmt.Errorf("MaybeDecrypt failed. Malformed encrypt?: %v", err)
		}
	}
	return val, nil
}

// newDiskCacheOpts reads the environment to provide the configuration options for DiskCache.
func newDiskCacheOpts(confFile AppConfiguration, opts CommandLineOpts) DiskCacheOpts {
	masterKey, err := MaybeDecrypt(cascade(OD_ENCRYPT_MASTERKEY, confFile.CacheSettings.MasterKey, ""))
	if err != nil {
		// If we get an error parsing the masterKey here we CANNOT continue, because we may begin writing
		// data committed to a gibberish key of "".  This must be an exit.  Nothing should work until the key
		// is supplied corrected, or just as plaintext.
		log.Printf(`
		The master encryption key was encoded with ENC{...}, but it will not decode properly.  
		Make sure that it was generated with the current token.jar: %v", err)`,
			err,
		)
		os.Exit(1)
	}
	if !confFile.ServerSettings.EncryptEnabled {
		masterKey = ""
	} else if masterKey == "" {
		log.Fatal("You must set master encryption key with OD_ENCRYPT_MASTERKEY to start the service when encryption is enabled")
	}
	settings := DiskCacheOpts{
		Root:          cascade(OD_CACHE_ROOT, confFile.CacheSettings.Root, "."),
		Partition:     cascade(OD_CACHE_PARTITION, confFile.CacheSettings.Partition, "cache"),
		LowWatermark:  cascadeFloat(OD_CACHE_LOWWATERMARK, confFile.CacheSettings.LowWatermark, .50),
		HighWatermark: cascadeFloat(OD_CACHE_HIGHWATERMARK, confFile.CacheSettings.HighWatermark, .75),
		EvictAge:      cascadeInt(OD_CACHE_EVICTAGE, confFile.CacheSettings.EvictAge, 300),
		WalkSleep:     cascadeInt(OD_CACHE_WALKSLEEP, confFile.CacheSettings.WalkSleep, 30),
		MasterKey:     masterKey,
		ChunkSize:     cascadeInt(OD_AWS_S3_FETCH_MB, confFile.CacheSettings.ChunkSize, 16),
	}
	return settings
}

// newServerSettingsFromEnv inspects the environment and returns a ServerSettingsConfiguration.
func newServerSettingsFromEnv(confFile AppConfiguration, opts CommandLineOpts) ServerSettingsConfiguration {

	var settings ServerSettingsConfiguration

	// From env
	settings.BasePath = cascade(OD_SERVER_BASEPATH, confFile.ServerSettings.BasePath, "/services/object-drive/1.0")
	settings.CAPath = cascade(OD_SERVER_CA, confFile.ServerSettings.CAPath, "")
	settings.ServerCertChain = cascade(OD_SERVER_CERT, confFile.ServerSettings.ServerCertChain, "")
	settings.ServerKey = cascade(OD_SERVER_KEY, confFile.ServerSettings.ServerKey, "")
	settings.ListenBind = cascade(OD_SERVER_BINDADDRESS, confFile.ServerSettings.ListenBind, "0.0.0.0")
	settings.ListenPort = cascade(OD_SERVER_PORT, confFile.ServerSettings.ListenPort, "4430")
	settings.IdleTimeout = cascadeInt(OD_SERVER_TIMEOUT_IDLE, confFile.ServerSettings.IdleTimeout, 60)
	settings.ReadTimeout = cascadeInt(OD_SERVER_TIMEOUT_READ, confFile.ServerSettings.ReadTimeout, 0)
	settings.ReadHeaderTimeout = cascadeInt(OD_SERVER_TIMEOUT_READHEADER, confFile.ServerSettings.ReadHeaderTimeout, 5)
	settings.WriteTimeout = cascadeInt(OD_SERVER_TIMEOUT_WRITE, confFile.ServerSettings.WriteTimeout, 3600)
	settings.EncryptEnabled = CascadeBoolFromString(OD_ENCRYPT_ENABLED, confFile.ServerSettings.EncryptEnabledString, true)
	settings.EncryptableFunctions = NewEncryptableFunctions(settings.EncryptEnabled)

	// Defaults
	settings.MinimumVersion = opts.TLSMinimumVersion
	// Use environment, configuration file, or cli options (includes a default) for the Cipher Suites (whichever has values first is used)
	settings.CipherSuites = selectNonEmptyStringSlice(CascadeStringSlice(OD_SERVER_CIPHERS, confFile.ServerSettings.CipherSuites, opts.Ciphers))

	// Use cli options, environment, or configuration file for the ACL whitelist (whichever has values first is used)
	settings.ACLImpersonationWhitelist = selectNonEmptyStringSlice(opts.Whitelist, getEnvSliceFromPrefix(OD_SERVER_ACL_WHITELIST), confFile.ServerSettings.ACLImpersonationWhitelist)
	// Command line argument, if given, supersedes environment, configuration file, and default for static root and templates
	if len(opts.StaticRootPath) > 0 {
		settings.PathToStaticFiles = opts.StaticRootPath
	} else {
		settings.PathToStaticFiles = cascade(OD_SERVER_STATIC_ROOT, confFile.ServerSettings.PathToStaticFiles, "")
	}
	if len(opts.TemplateDir) > 0 {
		settings.PathToTemplateFiles = opts.TemplateDir
	} else {
		settings.PathToTemplateFiles = cascade(OD_SERVER_TEMPLATE_ROOT, confFile.ServerSettings.PathToTemplateFiles, "")
	}

	return settings
}

//NewEncryptableFunctions creates the set of function that can have optional encryption
func NewEncryptableFunctions(encryptEnabled bool) EncryptableFunctions {
	if encryptEnabled {
		return EncryptableFunctions{
			EncryptionBanner:       NoopEncryptionBannerF,
			EncryptionWarning:      NoopEncryptionWarningF,
			DoCipherByReaderWriter: odrivecrypto.DoCipherByReaderWriter,
		}
	} else {
		return EncryptableFunctions{
			EncryptionBanner:       EncryptionBannerF,
			EncryptionWarning:      EncryptionWarningF,
			DoCipherByReaderWriter: odrivecrypto.DoNocipherByReaderWriter,
		}
	}
}

// newZKSettingsFromEnv inspects the environment and returns a AACConfiguration.
func newZKSettingsFromEnv(confFile AppConfiguration, opts CommandLineOpts) ZKSettings {

	var conf ZKSettings
	conf.Address = cascade(OD_ZK_URL, confFile.ZK.Address, "zk:2181")
	conf.BasepathOdrive = cascade(OD_ZK_ANNOUNCE, confFile.ZK.BasepathOdrive, "/services/object-drive/1.0")
	conf.IP = cascade(OD_ZK_MYIP, confFile.ZK.IP, util.GetIP(logger))
	conf.Port = cascade(OD_ZK_MYPORT, confFile.ZK.Port, confFile.ServerSettings.ListenPort)
	conf.Timeout = cascadeInt(OD_ZK_TIMEOUT, confFile.ZK.Timeout, 5)
	conf.RetryDelay = cascadeInt(OD_ZK_RETRYDELAY, confFile.ZK.RetryDelay, 3)
	conf.RecheckTime = cascadeInt(OD_ZK_RECHECK_TIME, confFile.ZK.RecheckTime, 30)

	return conf
}

// GetDatabaseHandle initializes database connection using the configuration
func (r *DatabaseConfiguration) GetDatabaseHandle() (*sqlx.DB, error) {
	// Establish configuration settings for Database Connection using
	// the TLS settings in config file
	if r.UseTLS {
		dbTLS := r.buildTLSConfig()
		switch r.Driver {
		case DBDRIVERMYSQL:
			mysql.RegisterTLSConfig("custom", &dbTLS)
		default:
			panic("Driver not supported")
		}
	} else {
		logger.Warn("database client connection is not using tls", zap.String(OD_DB_USE_TLS, os.Getenv(OD_DB_USE_TLS)))
	}
	// Setup handle to the database
	db, err := sqlx.Open(r.Driver, r.buildDSN())
	db.SetConnMaxLifetime(time.Second * time.Duration(r.MaxConnLifetime))
	db.SetMaxIdleConns(int(r.MaxIdleConns))
	db.SetMaxOpenConns(int(r.MaxConnLifetime))
	return db, err
}

// NewTLSClientConfig gets a config to make TLS connections
func NewTLSClientConfig(trustPath, certPath, keyPath, serverName string, insecure bool) (*tls.Config, error) {
	trustBytes, err := ioutil.ReadFile(trustPath)
	if err != nil {
		return nil, fmt.Errorf("Error parsing CA trust %s: %v", trustPath, err)
	}
	trustCertPool := x509.NewCertPool()
	if !trustCertPool.AppendCertsFromPEM(trustBytes) {
		return nil, fmt.Errorf("Error adding CA trust to pool: %v", err)
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("Error parsing cert: %v", err)
	}
	cfg := tls.Config{
		Certificates:             []tls.Certificate{cert},
		ClientCAs:                trustCertPool,
		InsecureSkipVerify:       insecure,
		ServerName:               serverName,
		PreferServerCipherSuites: true,
	}
	cfg.BuildNameToCertificate()

	return &cfg, nil
}

// NewTLSClientConn gets a TLS connection for a client, not sharing the config
func NewTLSClientConn(trustPath, certPath, keyPath, serverName, host, port string, insecure bool) (io.ReadWriteCloser, error) {
	conf, err := NewTLSClientConfig(trustPath, certPath, keyPath, serverName, insecure)
	if err != nil {
		return nil, err
	}
	return tls.Dial("tcp", fmt.Sprintf("%s:%s", host, port), conf)
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
			case DBDRIVERMYSQL:
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
	if len(r.Password) > 0 {
		logDSN = strings.Replace(logDSN, r.Password, "{password}", -1)
	}
	if len(r.Username) > 0 {
		logDSN = strings.Replace(logDSN, r.Username, "{username}", -1)
	}
	logger.Info("using this connection string", zap.String("dbdsn", logDSN))
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

	if conf.SkipVerify {
		logger.Warn("database tls client connection is not verifying server")
	}

	// Return a 1-way (server trust) if no client certificate configured
	if len(conf.ClientCert) == 0 || len(conf.ClientKey) == 0 {
		return tls.Config{
			RootCAs:            rootCAsCertPool,
			ServerName:         conf.Host,
			InsecureSkipVerify: conf.SkipVerify,
		}
	}

	// Return a 2-way (client/server trust) if client certificates are configured
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
	return buildServerTLSConfig(r.CAPath, r.ServerCertChain, r.ServerKey, r.CipherSuites, r.MinimumVersion)
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

func CascadeBoolFromString(fromEnv string, fromFile string, defaultVal bool) bool {
	if envVal := os.Getenv(fromEnv); envVal != "" {
		return (strings.ToLower(envVal) == "true")
	}
	if fromFile != "" {
		return (strings.ToLower(fromFile) == "true")
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

func getEnvSliceFromPrefix(envVar string) []string {
	sl := make([]string, 0)
	for _, e := range os.Environ() {
		i := strings.Index(e, "=")
		k := e[:i]
		v := e[i+1 : len(e)]
		if strings.HasPrefix(strings.ToUpper(k), strings.ToUpper(envVar)) && len(v) > 0 {
			sl = append(sl, v)
		}
	}
	return sl
}

// AWSConfig holds data suitable for creating AWS service session objects.
// Different regions and datacenters can be accessed by specifying non-default
// values for Endpoint and Region. Values for this struct may be provided
// by environment variables, IAM roles, AWS credentials files, or Object Drive
// application configuration.
type AWSConfig struct {
	// Endpoint represents the AWS datacenter that provides a service.
	Endpoint string
	// Region specifies the AWS region, e.g. "us-east-1"
	Region string
	// AccessKeyID is the AWS identity.
	AccessKeyID string
	// SecretAccessKey is the AWS secret key.
	SecretAccessKey string
}

// S3Config stores created config for S3.
type S3Config struct {
	AWSConfig *AWSConfig
}

// CWConfig config stores config for cloudwatch.
type CWConfig struct {
	AWSConfig *AWSConfig
	// SleepTimeInSeconds is the Cloudwatch polling interval, in seconds.
	SleepTimeInSeconds int
	// Name denotes the Cloudwatch namespace. This names where operators
	// can view Cloudwatch metrics for our service. A recommended value
	// would be the ZK path we announce to.
	Name string
}

// AutoScalingConfig session for the queueing service
type AutoScalingConfig struct {
	AWSConfigSQS *AWSConfig
	AWSConfigASG *AWSConfig
	// QueueName is the SQS queue name. If blank, SQS is disabled.
	QueueName string
	// GroupName is the name of our service's ASG.
	AutoScalingGroupName string
	// EC2InstanceID uniquely identifies the EC2 instance we are running on.
	// This is required to terminate our own instance.
	EC2InstanceID string
	// PollingInterval is the interval, in seconds, for polling our Cloudwatch metrics.
	PollingInterval int64
	// QueueBatchSize denotes the number of messages to retrieve from SQS per
	// each fetch to determine if message is intended to be processed by this
	// instance
	QueueBatchSize int64
}

// NewAWSConfig is default values for AWS config
func NewAWSConfig(endpoint string) *AWSConfig {
	ret := &AWSConfig{}
	//Per service
	ret.Endpoint = os.Getenv(endpoint)
	//Same for all
	ret.Region = getEnvOrDefault(OD_AWS_REGION, getEnvOrDefault("AWS_REGION", ""))
	ret.AccessKeyID = getEnvOrDefault(OD_AWS_ACCESS_KEY_ID, getEnvOrDefault("AWS_ACCESS_KEY_ID", ""))
	var err error
	ret.SecretAccessKey, err = MaybeDecrypt(getEnvOrDefault(OD_AWS_SECRET_ACCESS_KEY, getEnvOrDefault("AWS_SECRET_ACCESS_KEY", "")))
	if err != nil {
		log.Printf("The AWS_SECRET_ACCESS_KEY was supplied encrypted with the ENC{...} scheme, but it's not valid.  Supply this key un-encrypted, or re-encrypt it so that it's valid and matches token.jar: %v", err)
		os.Exit(1)
	}
	return ret
}

// NewS3Config is the s3 session
func NewS3Config() *S3Config {
	ret := &S3Config{}
	ret.AWSConfig = NewAWSConfig(OD_AWS_S3_ENDPOINT)
	return ret
}

// NewCWConfig is the cw session
func NewCWConfig() *CWConfig {
	ret := &CWConfig{}
	ret.AWSConfig = NewAWSConfig(OD_AWS_CLOUDWATCH_ENDPOINT)
	ret.SleepTimeInSeconds = int(getEnvOrDefaultInt(OD_AWS_CLOUDWATCH_INTERVAL, 300))
	ret.Name = getEnvOrDefault(OD_AWS_CLOUDWATCH_NAME, "")
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
	if ret.PollingInterval < 5 {
		ret.PollingInterval = 5
	}
	ret.QueueBatchSize = getEnvOrDefaultInt(OD_AWS_SQS_BATCHSIZE, 10)
	if ret.QueueBatchSize > 10 {
		ret.QueueBatchSize = 10
	}
	if ret.QueueBatchSize < 1 {
		ret.QueueBatchSize = 1
	}
	return ret
}

// setEnvironmentFromConfiguration will assign back to environment variables what the current
// configuration is. This is currently a helper function because there are places in the code
// that use the environment directly without regard to what the configuration of the app is.
// Not all environment variables are currently supported by the yaml configuration and structs
// defined above, so those lines are commented out below
func setEnvironmentFromConfiguration(conf AppConfiguration) {

	os.Setenv(OD_AAC_CA, conf.AACSettings.CAPath)
	os.Setenv(OD_AAC_CERT, conf.AACSettings.ClientCert)
	os.Setenv(OD_AAC_CN, conf.AACSettings.CommonName)
	os.Setenv(OD_AAC_HEALTHCHECK, conf.AACSettings.HealthCheck)
	os.Setenv(OD_AAC_HOST, conf.AACSettings.HostName)
	//os.Setenv(OD_AAC_INSECURE_SKIP_VERIFY,
	os.Setenv(OD_AAC_KEY, conf.AACSettings.ClientKey)
	os.Setenv(OD_AAC_PORT, conf.AACSettings.Port)
	os.Setenv(OD_AAC_RECHECK_TIME, string(conf.AACSettings.RecheckTime))
	os.Setenv(OD_AAC_WARMUP_TIME, string(conf.AACSettings.WarmupTime))
	os.Setenv(OD_AAC_ZK_ADDRS, strings.Join(conf.AACSettings.ZKAddrs, ","))
	// os.Setenv(OD_AWS_ACCESS_KEY_ID,
	// os.Setenv(OD_AWS_ASG_EC2,
	// os.Setenv(OD_AWS_ASG_ENDPOINT,
	// os.Setenv(OD_AWS_ASG_NAME,
	// os.Setenv(OD_AWS_CLOUDWATCH_ENDPOINT,
	// os.Setenv(OD_AWS_CLOUDWATCH_INTERVAL,
	// os.Setenv(OD_AWS_CLOUDWATCH_NAME,
	// os.Setenv(OD_AWS_REGION,
	// os.Setenv(OD_AWS_S3_BUCKET,
	// os.Setenv(OD_AWS_S3_ENDPOINT,
	// os.Setenv(OD_AWS_S3_FETCH_MB,
	// os.Setenv(OD_AWS_SECRET_ACCESS_KEY,
	// os.Setenv(OD_AWS_SQS_BATCHSIZE,
	// os.Setenv(OD_AWS_SQS_ENDPOINT,
	// os.Setenv(OD_AWS_SQS_INTERVAL,
	// os.Setenv(OD_AWS_SQS_NAME,
	// os.Setenv(OD_CACHE_EVICTAGE,
	// os.Setenv(OD_CACHE_HIGHWATERMARK,
	// os.Setenv(OD_CACHE_LOWWATERMARK,
	// os.Setenv(OD_CACHE_PARTITION,
	// os.Setenv(OD_CACHE_ROOT,
	// os.Setenv(OD_CACHE_WALKSLEEP,
	// os.Setenv(OD_DB_CA,
	// os.Setenv(OD_DB_CERT,
	// os.Setenv(OD_DB_CONN_PARAMS,
	// os.Setenv(OD_DB_CONNMAXLIFETIME,
	// os.Setenv(OD_DB_DEADLOCK_RETRYCOUNTER,
	// os.Setenv(OD_DB_DEADLOCK_RETRYDELAYMS,
	// os.Setenv(OD_DB_DRIVER,
	// os.Setenv(OD_DB_HOST,
	// os.Setenv(OD_DB_KEY,
	// os.Setenv(OD_DB_MAXIDLECONNS,
	// os.Setenv(OD_DB_MAXOPENCONNS,
	// os.Setenv(OD_DB_PASSWORD,
	// os.Setenv(OD_DB_PORT,
	// os.Setenv(OD_DB_PROTOCOL,
	// os.Setenv(OD_DB_SCHEMA,
	os.Setenv(OD_DB_SKIP_VERIFY, strconv.FormatBool(conf.DatabaseConnection.SkipVerify))
	os.Setenv(OD_DB_USE_TLS, strconv.FormatBool(conf.DatabaseConnection.UseTLS))
	// os.Setenv(OD_DB_USERNAME,
	// os.Setenv(OD_ENCRYPT_MASTERKEY,
	// os.Setenv(OD_EVENT_KAFKA_ADDRS,
	// os.Setenv(OD_EVENT_PUBLISH_FAILURE_ACTIONS,
	// os.Setenv(OD_EVENT_PUBLISH_SUCCESS_ACTIONS,
	// os.Setenv(OD_EVENT_TOPIC,
	// os.Setenv(OD_EVENT_ZK_ADDRS,
	// os.Setenv(OD_EXTERNAL_HOST,
	// os.Setenv(OD_EXTERNAL_PORT,
	// os.Setenv(OD_LOG_LEVEL,
	// os.Setenv(OD_LOG_LOCATION,
	// os.Setenv(OD_LOG_MODE,
	// os.Setenv(OD_PEER_CN,
	// os.Setenv(OD_PEER_SIGNIFIER,
	// os.Setenv(OD_PEER_INSECURE_SKIP_VERIFY,
	// os.Setenv(OD_SERVER_ACL_WHITELIST,
	// os.Setenv(OD_SERVER_BASEPATH,
	// os.Setenv(OD_SERVER_BINDADDRESS,
	// os.Setenv(OD_SERVER_CA,
	// os.Setenv(OD_SERVER_CERT,
	// os.Setenv(OD_SERVER_CIPHERS,
	// os.Setenv(OD_SERVER_KEY,
	// os.Setenv(OD_SERVER_PORT,
	// os.Setenv(OD_SERVER_STATIC_FILES,
	// os.Setenv(OD_SERVER_TEMPLATE_FILES,
	// os.Setenv(OD_SERVER_TIMEOUT_IDLE,
	// os.Setenv(OD_SERVER_TIMEOUT_READ,
	// os.Setenv(OD_SERVER_TIMEOUT_READHEADER,
	// os.Setenv(OD_SERVER_TIMEOUT_WRITE,
	// os.Setenv(OD_TOKENJAR_LOCATION,
	// os.Setenv(OD_TOKENJAR_PASSWORD,
	// os.Setenv(OD_ZK_AAC,
	os.Setenv(OD_ZK_ANNOUNCE, conf.AACSettings.AACAnnouncementPoint)
	// os.Setenv(OD_ZK_MYIP,
	// os.Setenv(OD_ZK_MYPORT,
	// os.Setenv(OD_ZK_RECHECK_TIME,
	// os.Setenv(OD_ZK_RETRYDELAY,
	// os.Setenv(OD_ZK_TIMEOUT,
	// os.Setenv(OD_ZK_URL,

}
