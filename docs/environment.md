FORMAT: 1A

# Object Drive

<table style="width:100%;border:0px;padding:0px;border-spacing:0;border-collapse:collapse;font-family:Helvetica;font-size:10pt;vertical-align:center;"><tbody><tr><td style="padding:0px;font-size:10pt;">Version</td><td style="padding:0px;font-size:10pt;">--Version--</td><td style="width:20%;font-size:8pt;"> </td><td style="padding:0px;font-size:10pt;">Build</td><td style="padding:0px;font-size:10pt;">--BuildNumber--</td><td style="width:20%;font-size:8pt;"></td><td style="padding:0px;font-size:10pt;">Date</td><td style="padding:0px;font-size:10pt;">--BuildDate--</td></tr></tbody></table>

# Group Navigation

## Table of Contents

+ [Service Overview](../../)
+ [RESTful API documentation](rest.html)
+ [Emitted Events documentation](events.html)
+ [Environment](environment.html)
+ [Changelog](changelog.html)
+ [BoringCrypto](boringcrypto.html)

# Group Environment Setup

The following environment variables can be set in the environment for usage by the object drive services. All environment variables are prefixed with `OD_` denoting 'object-drive'. This is akin to a namespace, and allows for reviewing settings on a system that may have environment variables defined for other uses.  You can quickly see all your environment variables for Object Drive via the following shell command:
```
env | grep OD_ | sort
```

For most environments, the following environment variables are the minimum that need to be set
<ul>
<li>OD_AAC_CA</li>
<li>OD_AAC_CERT</li>
<li>OD_AAC_CN</li>
<li>OD_AAC_KEY</li>
<li>OD_DB_CA</li>
<li>OD_DB_HOST</li>
<li>OD_DB_PASSWORD</li>
<li>OD_DB_SCHEMA</li>
<li>OD_DB_USERNAME</li>
<li>OD_ENCRYPT_MASTERKEY</li>
<li>OD_SERVER_CA</li>
<li>OD_SERVER_CERT</li>
<li>OD_SERVER_KEY</li>
</ul>

### AAC Integration
AAC Integration is used for authorization requests. At the time of this writing it is tightly coupled for CRUD type operations and uses snippets for listing/querying sets of objects.

| Name | Description | 
| --- | --- |
| OD_AAC_CA <br />_(since v1.0)_<br />__`Required`__ | The path to the certificate authority folder or file containing public certificate(s) in unencrypted PEM format to trust as the server when connecting to AAC.  | 
| OD_AAC_CERT <br />_(since v1.0)_<br />__`Required`__ | The path to the public certificate in unencrypted PEM format for the user credentials connecting to AAC. |
| OD_AAC_CN <br />_(since v1.0.12)_<br />__`Required`__ | The CN that we expect all AAC servers to have.  We use this when we enforce certificate verification.   |
| OD_AAC_HEALTHCHECK <br />_(since v1.0.12) | An acm expected to validate against the AAC service. <br />__`Default: {"version":"2.1.0","classif":"U"}`__ |
| OD_AAC_HOST <br />_(since v1.0)_ | The host of the AAC server to perform a direct connect instead of discovery.<br /><br />This should not be set for production environments. |
| OD_AAC_INSECURE_SKIP_VERIFY <br />_(since v1.0.1.22)_ | This turns off certificate verification.  <br />__`Default: false`__ |
| OD_AAC_KEY <br />_(since v1.0)_<br />__`Required`__ | The path to the private key in unencrypted PEM format for the user credentials connecting to AAC.  |
| OD_AAC_PORT <br />_(since v1.0)_ | The port of the AAC server to perform a direct connect instead of discovery.<br /><br />This should not be set for production environments. |
| OD_AAC_RECHECK_TIME <br />_(since v1.0.14)_ | The interval seconds between AAC health status checks (1-600) <br />__`Default: 30`__ |
| OD_AAC_WARMUP_TIME <br />_(since v1.0.14)_ | The number of seconds to wait for ZK before checking health of AAC (1-60) <br />__`Default: 10`__ |
| OD_AAC_ZK_ADDRS <br />_(since v1.0.1.7)_ | Comma-separated list of host:port pairs to connect to a Zookeeper cluster specific to AAC discovery. If this value is not set, AAC will be discovered using list of host:port pairs in OD_ZK_URL. |

### AWS S3
Amazon Web Services environment variables contain credentials for AWS used for S3 when configuring permanent storage.

| Name | Description | 
| --- | --- | 
| OD_AWS_ACCESS_KEY_ID <br />_(since v1.0)_ | The AWS Access Key. If leveraging [IAM Roles](https://console.aws.amazon.com/iam/home), do not set this variable. |
| OD_AWS_REGION <br />_(since v1.0)_ | The AWS region to use. (i.e. us-east-1, us-west-2).  |
| OD_AWS_S3_BUCKET <br />_(since v1.0)_ | The S3 Bucket name to use.  The credentials used defined in OD_AWS_SECRET_ACCESS_KEY and OD_AWS_ACCESS_KEY_ID must have READ and WRITE privileges to the bucket. If this value is not set then the instance will only store files in local cache.  |
| OD_AWS_S3_ENDPOINT <br />_(since v1.0)_ | The AWS S3 URL endpoint to use. [Documented here](http://docs.aws.amazon.com/general/latest/gr/rande.html). <br />__`Default: s3.amazonaws.com`__ |
| OD_AWS_S3_FETCH_MB <br />_(since v1.0.1.3)_ | The size (in MB) of chunks to pull from S3 in cases where Object Drive is re-caching from S3.  This is a compromise between response time vs billing caused by S3 billing per request. <br />__`Default: 16`__ |
| OD_AWS_SECRET_ACCESS_KEY <br />_(since v1.0)_ | AWS secret key. Access and secret key variables override credentials stored in credential and config files. If leveraging [IAM Roles](https://console.aws.amazon.com/iam/home), do not set this variable. <br />Values wrappped in `ENC{...}` are decrypted using token.jar.  |

### AWS Autoscaling
CloudWatch, SQS, and AutoScale with alarms (installed in AWS) interact to produce autoscaling behavior.

| Name | Description | 
| --- | --- | 
| OD_AWS_ASG_EC2 | This is the name assigned to the AMI instance that got launched, like a host name in the autoscaling group. This should be set to the AWS EC2 InstanceId if SQS and ASG are enabled which will be targetted for autoscaling actions |
| OD_AWS_ASG_ENDPOINT | The endpoint of the autoscaling group service. For some environments, it may be necessary to override this from the default used by the SDK. <br />__`Default: autoscaling.amazonaws.com`__ |
| OD_AWS_ASG_NAME | The is the name of the autoscaling group.  When unset, autoscaling is disabled. |
| OD_AWS_CLOUDWATCH_ENDPOINT| The location of cloudwatch monitoring. For some environments, it may be necessary to override this from the default. <br />__`Default: monitoring.us-east-1.amazonaws.com`__ |
| OD_AWS_CLOUDWATCH_INTERVAL | The frequency in seconds for how often stats are computed and sent to cloudwatch. <br />__`Default: 300`__ |
| OD_AWS_CLOUDWATCH_NAME | When reporting to cloud watch, a namespace is used. Leave blank to disable cloudwatch reports.  Usually set same as OD_ZK_ANNOUNCE is used as the value because it is unique per cluster.  When unset, metrics are not sent to Amazon Cloudwatch and are only logged. These metrics are  available on the /stats endpoint |
| OD_AWS_SQS_BATCHSIZE | The number of messages (1-10) to request from lifecycle queue per polling interval to examine for shutdown <br />__`Default: 10`__ |
| OD_AWS_SQS_ENDPOINT | The endpoint name of the SQS service.  For some environments, it may be necessary to override this from the default used by the SDK. <br />__`Default: sqs.us-east-1.amazonaws.com`__ |
| OD_AWS_SQS_INTERVAL | Poll interval for the lifecycle queue in seconds. Valid values between 5 and 60. <br />__`Default: 60`__ |
| OD_AWS_SQS_NAME | This is the name of the lifecycle queue.  When unset, autoscale is not used in termination | 

### Cache for Files
Storage cache on disk as an intermediary for upload/download to and from S3 for permanent storage. Otherwise, this is used solely for local storage with cache eviction disabled.

| Name | Description | 
| --- | --- | 
| OD_CACHE_EVICTAGE <br />_(since v1.0)_ | Denotes the minimum age, in seconds, a file in cache before it is eligible for eviction (purge) from the cache to free up space.  <br />__`Default: 300`__ |
| OD_CACHE_FILELIMIT <br />_(since v1.0.20)_ | Denotes the maximum number of cached files to keep. A value of 0 allows for unlimited files. This settings is useful if an excessive amount of time and processing is spent on checking large quantities of small files. <br />__`Default: 0`__ |
| OD_CACHE_FILESLEEP <br />_(since v1.0.20)_ | Denotes the duration, in milliseconds, for which the cache purge operation should sleep prior to processing each file. <br />__`Default: 0`__ |
| OD_CACHE_HIGHTHRESHOLDPERCENT <br />_(since v1.0.20)_ | Denotes a percentage of the file storage on the local mount point as the high size such that when the total space used exceeds the allocated percentage, a file in the cache will be purged if its age last used exceeds the eviction age time. <br />__`Default: 0.75`__ |
| OD_CACHE_LOWTHRESHOLDPERCENT <br />_(since v1.0.20)_ | Denotes a percentage of the file storage on the local mount point as the low size where total consumption must be at least that specified for files to be considered for purging. <br />__`Default: 0.50`__ |
| OD_CACHE_PARTITION <br />_(since v1.0)_ | An optional path for prefixing folders as part of the key in S3 prior to the cache folder. Intended for delineating different environments. For example, the Jenkins Continuous Integration Build Environment uses "jenkins/build" to easily identify files that were put in by the jenkins build instances that may safely be purged from the system. |
| OD_CACHE_ROOT <br />_(since v1.0)_ | An optional absolute or relative path to set the root of the local cache settings to override the default which beings in the same folder as working directory from which the Object Drive instance was started.  <br />__`Default: .`__ |
| OD_CACHE_WALKSLEEP <br />_(since v1.0)_ | Denotes the duration, in seconds, for which the cache purge operation should sleep prior to starting the next iteration. <br />__`Default: 30`__ |

### Cache Peer to Peer
When multiple instances of object-drive need to contact each other to collaborate on ciphertext. This option permits these instances to retrieve ciphertext from peers as a precursor to retrieving from permanent storage. This is useful to reduce costs, and as fault tolerance.

| Name | Description |
| --- | --- | 
| OD_PEER_CN <br />_(since v1.0.1.7)_ | The name associated with the certificate.  This may need to change when certificates are changed, but if it works at default, leave it.  This `MUST` be set in order to connect when this feature is enabled. <br />__`Default: twl-server-generic2`__ |
| OD_PEER_ENABLED <br />_(since v1.0.20)_ | Indicates whether an instance is permitted to retrieve ciphertext from its peers. <br />__`Default: true`__ |
| OD_PEER_INSECURE_SKIP_VERIFY <br />_(since v1.0.1.7)_ | This can turn off certificate verification for connecting to peer instances in the cluster. The trust, certificate, and key used by peer connections are those that are defined in the OD_SERVER_CA, OD_SERVER_CERT, and OD_SERVER_KEY.  <br />__`Default: false`__ |
| OD_PEER_SIGNIFIER <br />_(since v1.0.1.7)_ | This is a pseudonym used to signify a P2P client, which is set because it prevents users from accessing via gateway when the USER_DN header is assigned.  This generally does not need to be changed. <br />__`Default: P2P`__ |

### Cache for User AO
Configuration settings for managing the internal user authorization object cache that contributes to improved performance in ACM associations

| Name | Description |
| --- | --- | 
| OD_USERAOCACHE_LRU_TIME <br />_(since v1.0.20)_ | The time in seconds that a user AO will be cached in memory unless necessary to evict per least-recently-used caching constraints. <br />__`Default: 600`__ |
| OD_USERAOCACHE_TIMEOUT <br />_(since v1.0.20)_ | The permitted time in seconds to allow a User AO Cache rebuild to happen asynchronously before it will be assumed to have failed and permit another thread to attempt rebuild. <br />__`Default: 40`__ |


### Database for Metadata
The database is used to store metadata about objects and supports querying for matching objects to drive list operations and filter for user access.

| Name | Description | 
| --- | --- | 
| OD_DB_ACMGRANTEECACHE_LRU_TIME <br />_(since v1.0.23)_ | The time in seconds that an acmgrantee will be cached in memory unless necessary to evict per least-recently-used caching constraints. <br />__`Default: 600`__ |
| OD_DB_CA <br />_(since v1.0)_ | The path to the certificate authority folder or file containing public certificate(s) in unencrypted PEM format to trust as the server when connecting to the database over TLS.  When connecting to Amazon RDS, use the rds-combined-ca-bundle.pem. Additional documentation may be [found here](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.SSL.html) |
| OD_DB_CERT <br />_(since v1.0)_ | The path to the public certificate in unencrypted PEM format for the user credentials connecting to the database when using 2 way SSL.<br />This option is not available for Amazon RDS.  |
| OD_DB_CN <br />_(since v1.0.1.12)_ | The common name of the x509 certificate for the database.<br />This option is not available for Amazon RDS. |
| OD_DB_CONN_PARAMS <br />_(since v1.0)_ | Custom parameters to include for the database connection. <br /><br />For MySQL/MariaDB, the following value (not a default) is recommended: <span style="font-family:Arial;font-size:10pt;"> parseTime=true&collation=utf8_unicode_ci&readTimeout=30s</span><br /><br />If readTimeout is not specified, it will be defaulted to 30s  |
| OD_DB_CONNMAXLIFETIME <br />_(since v1.0.17)_ | The maximum amount of time, in seconds, that a database connection may be reused. 0 indicates indefinitely. <br />__`Default: 30`__ | 
| OD_DB_DEADLOCK_RETRYCOUNTER <br />_(since v1.0.19)_ | Indicates the number of times a create or update operation should be retried if the transaction fails due to a database deadlock. <br />__`Default: 30`__ |
| OD_DB_DEADLOCK_RETRYDELAYMS <br />_(since v1.0.19)_ | The duration in milliseconds between retry attempts for a create or update operation when a transaction fails due to a deadlock in the database. <br />__`Default: 55`__ |
| OD_DB_DRIVER <br />_(since v1.0.19)_ | The database driver to use. Supported values are <ul><li>mysql</li></ul>__`Default: mysql`__ |
| OD_DB_HOST <br />_(since v1.0)_ | The name or IP address of the MySQL / MariaDB / Aurora conforming database. <br />__`Default: metadatadb`__  | 
| OD_DB_KEY <br />_(since v1.0)_ | The path to the private key in unencrypted PEM format for the user credentials connecting to the database using 2 way SSL.<br />This option is not available for Amazon RDS.  | 
| OD_DB_MAXIDLECONNS <br />_(since v1.0)_| The maximum number of database connections to keep idle. Overrides language default of 2. <br />__`Default: 10`__ |
| OD_DB_MAXOPENCONNS <br />_(since v1.0)_ | The maximum number of database connections to keep open. Overrides language default of unlimited. <br />__`Default: 10`__ |
| OD_DB_PASSWORD <br />_(since v1.0)_<br />__`Required`__ | The password portion of credentials when connecting to the database. <br />Values wrappped in `ENC{...}` are decrypted using token.jar. |
| OD_DB_PORT <br />_(since v1.0)_ | The port that the MySQL / MariaDB / Aurora instance is listening on.  <br />__`Default: 3306`__ |  |
| OD_DB_PROTOCOL <br />_(since v1.0.19)_ | The protocol to use when communicating with the database. Supported values are <ul><li>tcp</li></ul>__`Default: tcp`__ |
| OD_DB_RECHECK_TIME <br />_(since v1.0.20)_| The interval seconds between database health status checks. Values less than 1 will disable the health check. <br />__`Default: 30`__ |
| OD_DB_SCHEMA <br />_(since v1.0)_<br />__`Required`__ | The schema to connect to after logging into the database.  |  |
| OD_DB_SKIP_VERIFY <br />_(since v1.0.19)_ | Indicates whether the hostname of an x509 certfiicate for SSL/TLS is verified. <br />__`Default: false`__ |
| OD_DB_USE_TLS <br />_(since v1.0.19)_ | Indicates whether the database connection should use encrypted using SSL/TLS. <br />__`Default: true`__ |
| OD_DB_USERNAME <br />_(since v1.0)_<br />__`Required`__ | The username portion of credentials when connecting to database.  |

### Event Publishing

Object Drive publishes a single event stream for client applications.

| Name | Description | 
| --- | --- | 
| OD_EVENT_KAFKA_ADDRS <br />_(since v1.0)_ | A comma-separated list of **host:port** pairs.  These are Kafka brokers. If both OD_EVENT_KAFKA_ADDRS and OD_EVENT_ZK_ADDRS are provided, then OD_EVENT_KAFKA_ADDRS will take precedence. <br />This should not be set for production environments. |
| OD_EVENT_PUBLISH_FAILURE_ACTIONS <br />_(since v1.0.1.14)_ | A comma delimited list of event action types that should be published to kafka if request failed. <br />A value of `*` indicates all failure events are published. <br />Supported values: <ul><li>access</li><li>authenticate</li><li>create</li><li>delete</li><li>list</li><li>undelete</li><li>unknown</li><li>update</li><li>zip</li></ul>__`Default: *`__ |
| OD_EVENT_PUBLISH_SUCCESS_ACTIONS <br />_(since v1.0.1.14)_ | A comma delimited list of event action types that should be published to kafka if request succeeded. <br />A value of `*` indicates all failure events are published. <br />Supported values: <ul><li>access</li><li>authenticate</li><li>create</li><li>delete</li><li>list</li><li>undelete</li><li>unknown</li><li>update</li><li>zip</li></ul>Recommended: create,delete,undelete,update<br />__`Default: *`__ |
| OD_EVENT_TOPIC <br />_(since v1.0.10)_ | The name of the topic for which events will be published to. <br />__`Default: odrive-event`__ |
| OD_EVENT_ZK_ADDRS <br />_(since v1.0.1.8)_ | Discovery of kafka nodes may be supported through the use of a Zookeeper cluster.  A comma-separated list of **host:port** pairs. These are Zookeeper nodes. If both OD_EVENT_KAFKA_ADDRS and OD_EVENT_ZK_ADDRS are provided, then OD_EVENT_KAFKA_ADDRS will take precedence.  Configurations using OD_EVENT_ZK_ADDRS will reconnect on failure. |

**NOTE:** If both the Kafka broker and ZooKeeper address options are blank, Object Drive will not publish events.

### Headers
Some request and response headers may be disabled or given a different name

| Name | Description |
| --- | --- |
| OD_HEADER_BANNER_ENABLED <br />_(since v1.0.21)_ | Indicates whether a response header representing the banner field of the object ACM should be provided in the response for content streams.<br />__`Default: true`__ |
| OD_HEADER_BANNER_NAME <br />_(since v1.0.21)_ | The name of the response header representing the banner field of the object ACM.<br />__`Default: Classification-Banner`__ |
| OD_HEADER_SERVER_ENABLED <br />_(since v1.0.21)_ | Indicates whether a response header denoting the server version should be set.<br />__`Default: true`__ |
| OD_HEADER_SERVER_NAME <br />_(since v1.0.21)_ | The name of the response header denoting the server version.<br />__`Default: odrive-server`__ |
| OD_HEADER_SESSIONID_ENABLED <br />_(since v1.0.21)_ | Indicates whether a response header denoting the session identifier should be set, as well as picked up in requests for session correlation. <br />__`Default: true`__ |
| OD_HEADER_SESSIONID_NAME <br />_(since v1.0.21)_ | The name of the header used for the session identifier.<br />__`Default: Session-Id`__ |


### Logging
ObjectDrive itself just logs to stdout.  But when the service script launches it from `/etc/init.d`, it reads an `env.sh` of environment variables.  One of the things that this environment variable does is to set a default log location and will take an override in `env.sh` itself.

| Name | Description | 
| --- | --- | 
| OD_LOG_LEVEL <br />_(since v1.0)_ | Indicates what level of logging, and above importance, should be logged. <br />Supported values:<ul><li>-1</li><li>0</li><li>1</li><li>2</li><li>3</li><li>DEBUG</li><li>INFO</li><li>WARN</li><li>ERROR</li><li>FATAL</li></ul>For production environments, INFO or WARN is recommended. Setting to DEBUG level is useful for identifying issues but will incur a performance penalty of about 5-7%.<br />Recommended: INFO<br />__`Default: 0`__   | 
| OD_LOG_LOCATION <br />_(since v1.0)_ | The absolute pathname to use for the object-drive service when overriding the default location of the log file. Typically this is supplied in `env.sh`. <br />__`Default: object-drive.log`__ |
| OD_LOG_MODE <br />_(since v1.0.17)_ | Denotes whether logging is in development or production mode.  When in development mode, stack traces will be output for WARN level messages and above. For production mode, stack traces are only output in ERROR level. Supported values: <ul><li>production</li><li>development</li></ul>__`Default: production`__ |

### Server
Remaining server settings are noted here

| Name | Description | 
| --- | --- | 
| OD_ENCRYPT_ENABLED <br />_(since v1.0.19)_ | Indicates whether file content should be encrypted at rest in local cache and permanent storage. <br />__`Default: true`__ |
| OD_ENCRYPT_MASTERKEY <br />_(since v1.0)_ | The secret master key used as part of the encryption key for all files stored in the system. If this value is changed, all file keys must be adjusted at the same time. This value is required if no value is set for OD_ENCRYPT_ENABLED, or if the value of that variable is set to true. <br />Values wrappped in `ENC{...}` are decrypted using token.jar.|
| OD_SERVER_ACL_WHITELIST*n* <br />_(since v1.0.11)_ | One or more environment variable prefixes to denote distinguished name assigned to the access control whitelist that controls whether a connector can impersonate as another identity. |
| OD_SERVER_BINDADDRESS <br />_(since v1.0.19)_ | The default interface address to bind the listener to. For all interfaces, use 0.0.0.0. <br />__`Default: 0.0.0.0`__ |
| OD_SERVER_CA <br />_(since v1.0)_<br />__`Required`__ | The path to the certificate authority folder or file containing public certificate(s) in unencrypted PEM format to trust as the server. |
| OD_SERVER_CERT <br />_(since v1.0)_<br />__`Required`__ | The path to the public certificate in unencrypted PEM format for the server credentials. |
| OD_SERVER_CIPHERS <br />_(since v1.0.11)_ | A comma delimited list of ciphers to be allowed for connections. Supported values: <ul style="font-family:Arial;font-size:10pt;"><li>TLS_RSA_WITH_RC4_128_SHA</li><li>TLS_RSA_WITH_3DES_EDE_CBC_SHA</li><li>TLS_RSA_WITH_AES_128_CBC_SHA</li><li>TLS_RSA_WITH_AES_256_CBC_SHA</li><li>TLS_ECDHE_ECDSA_WITH_RC4_128_SHA</li><li>TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA</li><li>TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA</li><li>TLS_ECDHE_RSA_WITH_RC4_128_SHA</li><li>TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA</li><li>TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA</li><li>TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA</li><li>TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256</li><li>TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256</li><li>TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384</li><li>TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384</li><li>TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305</li><li>TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305</li></ul> The following values are recommended <ul><li>TLS_RSA_WITH_AES_128_CBC_SHA</li><li>TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256</li></ul><br/>If no values are set then all ciphers will be enabled and a warning will be displayed during startup. |
| OD_SERVER_KEY <br />_(since v1.0)_<br />__`Required`__ | The path to the server's private key in unencrypted PEM format.   |  |
| OD_SERVER_MAXPAGESIZE <br />_(since v1.0.23)_ | The maximum number of results per page allowed for list/search operations. <br />__`Default: 100`__ |
| OD_SERVER_PORT <br />_(since v1.0)_ | The port for which this object-drive instance will listen on. Binding to ports below 1024 typically require setting additional security settings on the system. <br />__`Default: 4430`__ |
| OD_SERVER_STATIC_ROOT <br />_(since v1.0.19)_ | The location on disk where static assets are stored. |
| OD_SERVER_TEMPLATE_ROOT <br />_(since v1.0.19)_ | The location on disk where templates are stored. | 
| OD_SERVER_TIMEOUT_IDLE <br />_(since v1.0.17)_ | This is the maximum amount of time to wait for the next request when keep-alives are enabled, in seconds <br />__`Default: 60`__ |
| OD_SERVER_TIMEOUT_READ <br />_(since v1.0.17)_ | This is the maximum duration for reading the entire request, including the body. In most scenarios, you want to set the OD_SERVER_TIMEOUT_READHEADER value instead. | |
| OD_SERVER_TIMEOUT_READHEADER <br />_(since v1.0.17)_ | This is the amount of time allowed to read request headers, in seconds <br />__`Default: 5`__ |
| OD_SERVER_TIMEOUT_WRITE <br />_(since v1.0.17) | This is the maximum duration before timing out writes of the response, in seconds. <br />__`Default: 3600`__ |
| OD_TOKENJAR_LOCATION <br />_(since v1.0.1.11)_ | If a token.jar is placed on the filesystem to support secret encryption format, then this is the full location of that jar file.  That jar is presumed to have used OD_TOKENJAR_PASSWORD in its generation <br />__`Default: /opt/services/object-drive-$MajorMinorVersion/token.jar`__ |
| OD_TOKENJAR_PASSWORD <br />_(since v1.0.1.11)_ | This is the password that is embedded into code that is authorized to decrypt secrets.  The security of the system does not lie in this password, but in the fact that each token.jar should be using a fresh sample.dat that has a fresh key per cluster.  This value generally does not need an override, but it is here in case it does get changed without recompiling the code. The default value is embedded in compiled code. |

### Zookeeper
Zookeeper is used to announce the availability of this instance of the object drive services.  At the edge, gatekeeper and nginx rely upon this information to publish availability and facilitate routing requests to the service.

| Name | Description | 
| --- | --- | 
| OD_ZK_AAC <br />_(since v1.0)_ | The announce point for AAC nodes.  Matches gatekeeper config cluster.aac.zk-location <br />__`Default: /cte/service/aac/1.0/thrift`__ |
| OD_ZK_ANNOUNCE <br />_(since v1.0)_ | The mount point for announcements where our zookeeper https node is placed. <br />__`Default: /services/object-drive/$MajorMinorVersion`__ |
| OD_ZK_MYIP <br />_(since v1.0)_ | The IP address of the Object-Drive server as reported to Zookeeper. If this environment variable is defined it will override the value detected as the server's IP address on startup.|
| OD_ZK_MYPORT <br />_(since v1.0)_ | The Port of the Object-Drive server as reported to Zookeeper. If this environment variable is defined it will override the value detected as the server's listening port on startup. <br/>__`Default: 4430`__ |
| OD_ZK_RECHECK_TIME <br />_(since v1.0.17)_ | The interval seconds between ZK health status checks (1-600) <br />__`Default: 30`__ |
| OD_ZK_RETRYDELAY <br />_(since v1.0.14)_ | The number of seconds between retry attempts when connecting to ZooKeeper (1-10) <br />__`Default: 3`__ |
| OD_ZK_TIMEOUT <br />_(since v1.0)_ | Timeout in seconds for zookeeper sessions <br />__`Default: 5`__ |
| OD_ZK_URL <br />_(since v1.0)_ | A comma delimited list of zookeeper instances to announce to. The structure of this value should be server1:port1,server2:port2,serverN:portN. If misconfigured, the server will never fully start. <br />__`Default: zk:2181`__ |

