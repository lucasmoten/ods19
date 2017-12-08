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

	"github.com/deciphernow/object-drive-server/util"

	"github.com/deciphernow/commons/gov/encryptor"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"github.com/urfave/cli"
)

var (
	defaultDBDriver = "mysql"
	defaultDBHost   = "metadatadb"
	defaultDBPort   = "3306"
	// DefaultBucket is the name of the S3 storage bucket to use for encrypted files
	DefaultBucket = getEnvOrDefault("OD_AWS_S3_BUCKET", "")
)

var empty []string

// AppConfiguration is a structure that defines the known configuration format
// for this application.
type AppConfiguration struct {
	DatabaseConnection DatabaseConfiguration       `yaml:"database"`
	ServerSettings     ServerSettingsConfiguration `yaml:"server"`
	AACSettings        AACConfiguration            `yaml:"aac"`
	CacheSettings      S3CiphertextCacheOpts       `yaml:"disk_cache"`
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
	// Hostname is the hostname of the AAC service
	HostName string `yaml:"hostname"`
	// Port is the port AAC is listening on.
	Port string `yaml:"port"`
	// AACAnnouncementPoint is a path to inspect in Zookeeper, if we are using
	// service discovery to connect to AAC.
	AACAnnouncementPoint string `yaml:"zk_path"`
	// ZKAddrs can be set to discover AAC from a non-default Zookeeper cluster.
	ZKAddrs []string `yaml:"zk_addrs"`
}

// CommandLineOpts holds command line options parsed on application start. This
// object is passed to many higher level constructors, so that command line params
// can override certain configurations.
type CommandLineOpts struct {
	// Ciphers is a list of TLS ciphers we are willing to accept.
	Ciphers []string
	// UseTLS specifies whether we will only accept TLS connections.
	UseTLS bool
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
	// Port is the database port. Commonly 3363 for MySQL.
	Port string `yaml:"port"`
	// Schema is the database name to connect to. A single server can host
	// many logical schemas. The object drive default is "metadatadb".
	Schema string `yaml:"schema"`
	// Params are custom connection params injected into the DSN. These
	// will vary depending on your server's configuration.
	Params string `yaml:"conn_params"`
	// UseTLS determines whether you should connect to the database with TLS.
	// This is currently hardcoded to true.
	UseTLS bool `yaml:"use_tls"`
	// SkipVerify controls whether the hostname of an SSL peer is verified.
	// This is hardcoded to false for legacy compatibility reasons.
	SkipVerify bool `yaml:"insecure_skip_veriry"`
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
}

// EventQueueConfiguration configures publishing to the Kakfa event queue.
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

// S3CiphertextCacheOpts describes our current disk cache configuration.
type S3CiphertextCacheOpts struct {
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
	// items older than EvictAge will be eligible for puge.
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
	// BasePath is the root URL for static assets. Only used for debug UI.
	BasePath string `yaml:"base_path"`
	// ListenPort is the port the server listens on. Default is 4430.
	ListenPort string `yaml:"port"`
	// ListenBind is the address to bind to. Hardcoded to 0.0.0.0
	ListenBind string `yaml:"bind"`
	// UseTLS controls whether the server requires TLS. Default is true.
	UseTLS bool `yaml:"use_tls"`
	// CAPath is the path to a PEM encoded certificate of our CA.
	CAPath string `yaml:"trust"`
	// ServerCertChain is the path to our server's PEM encoded cert.
	ServerCertChain string `yaml:"cert"`
	// ServerKey is the path to our server's PEM encoded key.
	ServerKey string `yaml:"key"`
	// RequireClientCert specifies whether clients must present a certificate
	// signed by our CA. Default is true.
	RequireClientCert bool `yaml:"require_client_cert"`
	// CipherSuites specifies the ciphers we will accept. Common values are
	// TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 and TLS_RSA_WITH_AES_128_CBC_SHA
	CipherSuites []string `yaml:"ciphers"`
	// MinimumVersion is the minimum TLS protocol version we support. Currently TLS 1.2
	MinimumVersion string `yaml:"min_version"`
	// ACLImpersonationWhitelist is a list of Distinguished Names. If a client
	// (usually another machine) is on this list, it may pass us another DN in
	// an HTTP header, and "impersonate" that identitiy. The common use case
	// is for an edge proxy (such as nginx) to pass through requests from users
	// outside the network. This configuration option must be specified in YAML
	// or on the command line.
	ACLImpersonationWhitelist []string `yaml:"acl_whitelist"`
	// PathToStaticFiles is a location on disk where static assets are stored.
	PathToStaticFiles string `yaml:"static_root"`
	// PathToTemplateFiles is a location on disk where Go templates are stored.
	PathToTemplateFiles string `yaml:"template_root"`
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
	serverSettings := NewServerSettingsFromEnv(confFile, opts)
	aacSettings := NewAACSettingsFromEnv(confFile, opts)
	cacheSettings := NewS3CiphertextCacheOpts(confFile, opts)
	zkSettings := NewZKSettingsFromEnv(confFile, opts)
	if zkSettings.Port == "" {
		zkSettings.Port = serverSettings.ListenPort
	}
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

	// Defaults
	dbConf.Protocol = "tcp"
	dbConf.Driver = defaultDBDriver
	dbConf.UseTLS = true
	dbConf.SkipVerify = true

	// Parameters necessary to handle deadlock situations
	dbConf.DeadlockRetryCounter = cascadeInt(OD_DEADLOCK_RETRYCOUNTER, confFile.DatabaseConnection.DeadlockRetryCounter, 30)
	dbConf.DeadlockRetryDelay = cascadeInt(OD_DEADLOCK_RETRYDELAYMS, confFile.DatabaseConnection.DeadlockRetryDelay, 55)

	return dbConf
}

// NewEventQueueConfiguration reades the environment to provide the configuration for the Kafka event queue.
func NewEventQueueConfiguration(confFile AppConfiguration, opts CommandLineOpts) EventQueueConfiguration {
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

// NewS3CiphertextCacheOpts reads the environment to provide the configuration options for
// S3CiphertextCache.
func NewS3CiphertextCacheOpts(confFile AppConfiguration, opts CommandLineOpts) S3CiphertextCacheOpts {
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
	settings := S3CiphertextCacheOpts{
		Root:          cascade(OD_CACHE_ROOT, confFile.CacheSettings.Root, "."),
		Partition:     cascade(OD_CACHE_PARTITION, confFile.CacheSettings.Partition, "cache"),
		LowWatermark:  cascadeFloat(OD_CACHE_LOWWATERMARK, confFile.CacheSettings.LowWatermark, .50),
		HighWatermark: cascadeFloat(OD_CACHE_HIGHWATERMARK, confFile.CacheSettings.HighWatermark, .75),
		EvictAge:      cascadeInt(OD_CACHE_EVICTAGE, confFile.CacheSettings.EvictAge, 300),
		WalkSleep:     cascadeInt(OD_CACHE_WALKSLEEP, confFile.CacheSettings.WalkSleep, 30),
		MasterKey:     masterKey,
		ChunkSize:     cascadeInt(OD_AWS_S3_FETCH_MB, confFile.CacheSettings.ChunkSize, 16),
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

	// Use environment, configuration file, or cli options (includes a default) for the Cipher Suites (whichver has values first is used)
	settings.CipherSuites = selectNonEmptyStringSlice(CascadeStringSlice(OD_SERVER_CIPHERS, confFile.ServerSettings.CipherSuites, opts.Ciphers))

	// Use cli options, environment, or configuration file for the ACL whitelist (whichever has values first is used)
	settings.ACLImpersonationWhitelist = selectNonEmptyStringSlice(opts.Whitelist, getEnvSliceFromPrefix(OD_SERVER_ACL_WHITELIST), confFile.ServerSettings.ACLImpersonationWhitelist)

	// Defaults
	settings.ListenBind = "0.0.0.0"
	settings.UseTLS = opts.UseTLS
	settings.RequireClientCert = true
	settings.MinimumVersion = opts.TLSMinimumVersion
	settings.PathToStaticFiles = opts.StaticRootPath
	settings.PathToTemplateFiles = opts.TemplateDir

	return settings
}

// NewZKSettingsFromEnv inspects the environment and returns a AACConfiguration.
func NewZKSettingsFromEnv(confFile AppConfiguration, opts CommandLineOpts) ZKSettings {

	var conf ZKSettings
	conf.Address = cascade(OD_ZK_URL, confFile.ZK.Address, "zk:2181")
	conf.BasepathOdrive = cascade(OD_ZK_ANNOUNCE, confFile.ZK.BasepathOdrive, "/services/object-drive/1.0")
	conf.IP = cascade(OD_ZK_MYIP, confFile.ZK.IP, util.GetIP(logger))
	conf.Port = cascade(OD_ZK_MYPORT, confFile.ZK.Port, "")
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
		log.Printf("The AWS_SECRET_ACCESS_KEY was supplied encrypted with the ENC{...} scheme, but it's not valid.  Supply this key unencrypted, or re-encrypt it so that it's valid and matches token.jar: %v", err)
		os.Exit(1)
	}
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
