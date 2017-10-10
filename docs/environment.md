FORMAT: 1A

# Object Drive 1.0 

# Group Navigation

## Table of Contents

+ [Service Overview](../../)
+ [RESTful API documentation](rest.html)
+ [Environment](environment.html)
+ [Changelog](changelog.html)

# Group Environment Setup

The following environment variables can be set in the environment for usage by the object drive services

### AAC Integration
AAC Integration is used for authorization requests. At the time of this writing it is tightly coupled for CRUD type operations and uses snippets for listing/querying sets of objects.

| Name | Description | Default |
| --- | --- | --- |
| OD_AAC_CA | The path to the certificate authority folder or file containing public certificate(s) to trust as the server when connecting to AAC.  |  |
| OD_AAC_CN | The CN that we expect all AAC servers to have.  We use this when we enforce certificate verification.  This `MUST` be set in order to connect. | |
| OD_AAC_CERT | The path to the public certificate for the user credentials connecting to AAC.  |  |
| OD_AAC_KEY | The path to the private key for the user credentials connecting to AAC.  |  |
| OD_AAC_ZK_ADDRS | Comma-separated list of host:port pairs to connect to a Zookeeper cluster specific to AAC discovery  |  |
| OD_AAC_INSECURE_SKIP_VERIFY | This turns off certificate verification.  Do not do this.  Leave this value at its default. | false |

### AWS Settings
Amazon Web Services environment variables contain credentials for AWS used for S3 and other backend cloud services.

| Name | Description | Default |
| --- | --- | --- |
| OD_AWS_ACCESS_KEY_ID | The AWS Access Key. Available here: https://console.aws.amazon.com/iam/home |  |
| OD_AWS_ENDPOINT | The AWS S3 URL endpoint to use. Documented at: http://docs.aws.amazon.com/general/latest/gr/rande.html | |
| OD_AWS_REGION | The AWS region to use. (i.e. us-east-1, us-west-2).  |  |
| OD_AWS_S3_BUCKET | The S3 Bucket name to use.  The credentials used defined in OD_AWS_SECRET_ACCESS_KEY and OD_AWS_ACCESS_KEY_ID must have READ and WRITE privileges to the bucket. |  |
| OD_AWS_S3_ENDPOINT | On the high side we must override this to point to the location of S3 services. OD_AWS_ENDPOINT is a deprecated duplicate of this variable | s3.amazonaws.com |
| OD_AWS_SECRET_ACCESS_KEY | AWS secret key. Access and secret key variables override credentials stored in credential and config files.  Note that if a token.jar is installed onto the system, we can use the Bedrock encrypt format like `ENC{...}` |  |
|OD_AWS_S3_FETCH_MB| The size (in MB) of chunks to pull from S3 in cases where odrive is re-caching from S3.  This is a compromise between response time vs billing caused by S3 billing per request.|16|

### AutoScaling
CloudWatch, SQS, and AutoScale with alarms (installed in AWS) interact to produce autoscaling behavior.

| Name | Description | Default |
| --- | --- | --- |
| OD_AWS_ASG_EC2 | This is the name assigned to the AMI instance that got launched, like a host name in the autoscaling group  | no default.  should be set to the AWS EC2 InstanceId (they look like: i-d0a2e853) if SQS and ASG are enabled |
| OD_AWS_ASG_ENDPOINT | This is the location of the autoscaling service.  |  leave blank by default. which is implicitly autoscaling.amazonaws.com and redirects to autoscaling.us-east-1.amazonaws.com) |
| OD_AWS_ASG_NAME | This is the name of the autoscaling group.  | set blank to disable notifying autoscale |
| OD_AWS_CLOUDWATCH_ENDPOINT| The loction of cloudwatch monitoring | On the high side, we must override endpoint names.  Therefore, we will need an override (that begins with monitoring) for cloudwatch| monitoring.us-east-1.amazonaws.com|
| OD_AWS_CLOUDWATCH_NAME|When reporting to cloud watch, we must report into a namespace.  In production, it's the same as the zk url. | Leave blank to disable cloudwatch reports.  Usually set same as OD_ZK_ANNOUNCE is used as the value because it is unique per cluster.  If it is blank, then metrics are logged rather than sent to cloud watch. |
| OD_AWS_ENDPOINT|On the high side we must override this to point to the location of S3 services. | leave blank by default. Set on high side |
| OD_AWS_SQS_BATCHSIZE | The number of messages (1-10) to request from lifecycle queue per polling interval to examine for shutdown | 10 |
| OD_AWS_SQS_ENDPOINT | The name of the SQS service. | leave blank by default, which is implicitly sqs.us-east-1.amazonaws.com |
| OD_AWS_SQS_INTERVAL | Poll interval for the lifecycle queue in seconds | 60 |
| OD_AWS_SQS_NAME | This is the name of the lifecycle queue.  Blank to disable use of SQS for this purpose. When left blank (default), the shutdown by message is disabled |  |

### Cache Settings
Storage cache on disk as an intermediary for upload/download to and from S3

| Name | Description | Default |
| --- | --- | --- |
| OD_CACHE_EVICTAGE | Denotes the minimum age, in seconds, a file in cache before it is eligible for eviction (purge) from the cache to free up space.  | 300 |
| OD_CACHE_HIGHWATERMARK | Denotes a percentage of the file storage on the local mount point as the high size such that when the total space used exceeds the allocated percentage, a file in the cache will be purged if its age last used exceeds the eviction age time.  | 0.75 |
| OD_CACHE_LOWWATERMARK | Denotes a percentage of the file storage on the local mount point as the low size where total consumption must be at least that specified for files to be considered for purging.  | 0.50 |
| OD_CACHE_PARTITION | An optional path for prefixing folders as part of the key in S3 prior to the cache folder. Intended for delineating different environments. For example, the Jenkins Continuous Integration Build Environment uses "jenkins/build" to easily identify files that were put in by the jenkins build odrive instances that may safely be purged from the system.  |  |
| OD_CACHE_ROOT | An optional absolute or relative path to set the root of the local cache settings to override the default which beings in the same folder as working directory from which the odrive instance was started.  odrive should be run as user "odrive" rather than "root", and OD_CACHE_ROOT should be readable and writable by this user.  When installed via an rpm, launch as odrive is handled by the init script in recent builds.| . |
| OD_CACHE_WALKSLEEP | Denotes the frequency, in seconds, for which all files in the cache are examined to determine if they should be purged.  | 30 |

### Database Connection Settings
The database is used to store metadata about objects and supports querying for matching objects to drive list operations and filter for user access.

| Name | Description | Default |
| --- | --- | --- |
| OD_DB_CA | The path to the certificate authority folder or file containing public certificate(s) to trust as the server when connecting to the database over TLS.  |  |
| OD_DB_CN | The cn of the ssl cert of the database. | fqdn.for.metadatadb.local (default is for testing only) |
| OD_DB_CERT | The path to the public certificate for the user credentials connecting to the database.  |  |
| OD_DB_CONN_PARAMS | Custom parameters to include for the database connection. For MySQL/MariaDB, we are using `parseTime=true&collation=utf8_unicode_ci` |  |
| OD_DB_HOST | The name or IP address of the MySQL / MariaDB / Aurora conforming database.  |  |
| OD_DB_KEY | The path to the private key for the user credentials connecting to the database.  |  | 
| OD_DB_MAXIDLECONNS | The maximum number of database connections to keep idle. Overrides language default of 2.  | 10 |
| OD_DB_MAXOPENCONNS | The maximum number of database connections to keep open. Overrides language default of unlimited.  | 10 |
| OD_DB_PASSWORD | The password portion of credentials when connecting to the database. Note that if a token.jar is installed onto the system, we can use the Bedrock encrypt format like `ENC{...} |  |
| OD_DB_PORT | The port that the MySQL / MariaDB / Aurora instance is listening on.  |  |
| OD_DB_SCHEMA | The schema to connect to after logging into the database.  |  |
| OD_DB_USERNAME | The username portion of credentials when connecting to database.  |  |

### Database Dev AWS Settings 

In order to connect to an AWS database that is distinct from your local database, set environment variables so that you can run mysql-client.sh and mysql-aws.sh concurrently from your workstation shells:

**DEVELOPMENT ONLY** 

| Name | Description | Default |
| --- | --- | --- |
| OD_DB_AWS_MYSQL_MASTER_USER | OD_DB_USERNAME override for aws staging instance  |  |
| OD_DB_AWS_MYSQL_MASTER_PASSWORD | OD_DB_PASSWORD override for aws staging instance  |  |
| OD_DB_AWS_MYSQL_ENDPOINT | OD_DB_HOST override for aws staging instance  |  |
| OD_DB_AWS_MYSQL_PORT | OD_DB_PORT override for aws staging instance  |  |
| OD_DB_AWS_MYSQL_DATABASE_NAME | OD_DB_SCHEMA override for aws staging instance  |  |
| OD_DB_AWS_MYSQL_SSL_CA_PATH | OD_DB_CA override for aws staging instance  |  |

### Event Queue

Object Drive publishes a single event stream for client applications.

| Name | Description | Default |
| --- | --- | --- |
| OD_EVENT_KAFKA_ADDRS | A comma-separated list of **host:port** pairs.  These are Kafka brokers. | |
| OD_EVENT_ZK_ADDRS | A comma-separated list of **host:port** pairs. These are ZK nodes.  | |
| OD_EVENT_PUBLISH_FAILURE_ACTIONS | A comma delimited list of event action types that should be published to kafka if request failed. The default value * enables all failure events to be published. Permissible values are access, authenticate, create, delete, list, undelete, unknown, update, zip. | * |
| OD_EVENT_PUBLISH_SUCCESS_ACTIONS | A comma delimited list of event action types that should be published to kafka if request succeeded. The default value * enables all success events to be published. Permissible values are access, authenticate, create, delete, list, undelete, unknown, update, zip. | * |
| OD_EVENT_TOPIC | The name of the topic for which events will be published to. | odrive-event |

**NOTE:** If both Event Queue config options are blank, odrive will not publish events.

### P2P
When odrives need to contact each other to collaborate on ciphertext

| Name | Description | Default |
| --- | --- | --- |
| OD_PEER_CN | The name associated with the certificate.  This may need to change when certificates are changed, but if it works at default, leave it.  This `MUST` be set in order to connect. |  |
| OD_PEER_SIGNIFIER | This is a pseudonym used to signify a P2P client, which is set because it prevents users from accessing via nginx.  This generally doesn't need to be changed. | P2P |
| OD_PEER_INSECURE_SKIP_VERIFY | This turns off certificate verification.  Do not do this.  Leave this value at its default. | false |

### Server
Remaining server settings are noted here

| Name | Description | Default |
| --- | --- | --- |
| OD_DEADLOCK_RETRYCOUNTER | Indicates the number of times a create or update operation should be retried if the transaction fails due to a database deadlock | 30 |
| OD_DEADLOCK_RETRYDELAYMS | The duration in milliseconds between retry attempts for a create or update operation when a transaction fails due to a deadlock in the database | 55 |
| OD_DOCKERVM_OVERRIDE | **DEVELOPMENT ONLY** Allows for overriding the host name used for go tests when checking server integration tests.  | dockervm |
| OD_DOCKERVM_PORT | **DEVELOPMENT ONLY** Allows for overriding the port used for go tests when checking server integration tests to bypass nginx.  | 8080 |
| **OD_ENCRYPT_MASTERKEY** | The secret master key used as part of the encryption key for all files stored in the system. If this value is changed, all file keys must be adjusted at the same time. If you don't set this, the service will shut down.  Note that if a token.jar is installed onto the system, we can use the Bedrock encrypt format like `ENC{...} | |
| OD_SERVER_BASEPATH | The base URL root. Used in debug UIs.    | "/services/object-drive/1.0" |
| OD_SERVER_CA | The path to the certificate authority folder or file containing public certificate(s) to trust as the server.    |  |
| OD_SERVER_CERT |The path to the public certificate for the server credentials.  |  |
| OD_SERVER_KEY | The path to the server's private key.   |  |
| OD_SERVER_PORT | The port for which this object-drive instance will listen on. Binding to ports below 1024 require setting additional security settings on the system.  | 4430 |
| OD_LOG_LEVEL | Should be Info (OD_LOG_LEVEL=0, -1 is Debug, 0 is Info, 1 is Warn, 2 is Error, 3 is Fatal, etc.) for production systems | 0 |
| OD_TOKENJAR_LOCATION | If a token.jar is placed on the filesystem to support Bedrock secret encryption format, then this is the full location of that jar file.  That jar is presumed to have used OD_TOKENJAR_PASSWORD in its generation | `/opt/services/object-drive-1.0/token.jar` |
| OD_TOKENJAR_PASSWORD | This is the password that is embedded into code that is authorized to decrypt secrets that we cannot avoid writing down on the system.  The security of the system does not lie in this password, but in the fact that each token.jar should be using a fresh sample.dat that has a fresh key per cluster.  This value generally does not need an override, but it is here in case it does get changed without recompiling the code.  | Embedded in compiled code |

### Zookeeper Announcement
Zookeeper is used to announce the availability of this instance of the object drive services.  At the edge, gatekeeper and nginx rely upon this information to publish availability and facilitate routing requests to the service.

| Name | Description | Default |
| --- | --- | --- |
| OD_ZK_AAC | The announce point for AAC nodes.  Matches gatekeeper config cluster.aac.zk-location | /cte/service/aac/1.0/thrift |
| OD_ZK_ANNOUNCE|The mount point for announcements where our zookeeper https node is placed.   The point of this variable is to match the gatekeeper cluster.odrive.zk-location without the https part | /services/object-drive/1.0 |
| OD_ZK_MYIP | The IP address of the Object-Drive server as reported to Zookeeper. If this environment variable is defined it will override the value detected as the server's IP address on startup. | globalconfig.MyIP |
| OD_ZK_MYPORT | The Port of the Object-Drive server as reported to Zookeeper. If this environment variable is defined it will override the value detected as the server's listening port on startup. | serverPort _4430_ |
| OD_ZK_TIMEOUT | Timeout in seconds for zookeeper sessions | 5 |
| OD_ZK_URL | A comma delimited list of zookeeper instances to announce to. The structure of this value should be server1:port1,server2:port2,serverN:portN. If misconfigured, the server will never start.  | zk:2181 |

### Logging
ObjectDrive itself just logs to stdout.  But when the `/etc/init.d/odrive` service script launches it, it reads an `env.sh` of environment variables.  One of the things that this environment variable does is to set a default log location and will take an override in env.sh itself.

| Name | Description | Default |
| --- | --- | --- |
| OD_LOG_LOCATION | The location of a log file, supplied in `env.sh`  to override log location.  Deal with log rotation by putting a date stamp in the name.| object-drive.log |

For example, the `env.sh` on Bedrock:
```
export OD_LOG_LOCATION=/opt/bedrock/odrive/log/object-drive-`date +%FT%H_%M`.log
```
Since these servers are restarted nightly, the log will rotate to a new file every time it is restarted.  The odrive binary itself, being container-oriented will just log to stdout.  If you are not using the normal `service odrive start` to launch it, then have a bash script that sets the environment and puts a date in the log file.  There is a presumption that the service is regularly restarted, which is true in Bedrock.  If you need to do this with a cron job, then you can do so.
