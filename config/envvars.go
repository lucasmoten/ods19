package config

import (
	"fmt"
	"html/template"
	"os"
)

// Environment variables
const (
	OD_AAC_CA                        = "OD_AAC_CA"
	OD_AAC_CERT                      = "OD_AAC_CERT"
	OD_AAC_CN                        = "OD_AAC_CN"
	OD_AAC_HEALTHCHECK               = "OD_AAC_HEALTHCHECK"
	OD_AAC_HOST                      = "OD_AAC_HOST"
	OD_AAC_INSECURE_SKIP_VERIFY      = "OD_AAC_INSECURE_SKIP_VERIFY"
	OD_AAC_KEY                       = "OD_AAC_KEY"
	OD_AAC_PORT                      = "OD_AAC_PORT"
	OD_AAC_RECHECK_TIME              = "OD_AAC_RECHECK_TIME"
	OD_AAC_WARMUP_TIME               = "OD_AAC_WARMUP_TIME"
	OD_AAC_ZK_ADDRS                  = "OD_AAC_ZK_ADDRS"
	OD_AWS_ACCESS_KEY_ID             = "OD_AWS_ACCESS_KEY_ID"
	OD_AWS_ASG_EC2                   = "OD_AWS_ASG_EC2"
	OD_AWS_ASG_ENDPOINT              = "OD_AWS_ASG_ENDPOINT"
	OD_AWS_ASG_NAME                  = "OD_AWS_ASG_NAME"
	OD_AWS_CLOUDWATCH_ENDPOINT       = "OD_AWS_CLOUDWATCH_ENDPOINT"
	OD_AWS_CLOUDWATCH_INTERVAL       = "OD_AWS_CLOUDWATCH_INTERVAL"
	OD_AWS_CLOUDWATCH_NAME           = "OD_AWS_CLOUDWATCH_NAME"
	OD_AWS_REGION                    = "OD_AWS_REGION"
	OD_AWS_S3_BUCKET                 = "OD_AWS_S3_BUCKET"
	OD_AWS_S3_ENDPOINT               = "OD_AWS_S3_ENDPOINT"
	OD_AWS_S3_FETCH_MB               = "OD_AWS_S3_FETCH_MB"
	OD_AWS_SECRET_ACCESS_KEY         = "OD_AWS_SECRET_ACCESS_KEY"
	OD_AWS_SQS_BATCHSIZE             = "OD_AWS_SQS_BATCHSIZE"
	OD_AWS_SQS_ENDPOINT              = "OD_AWS_SQS_ENDPOINT"
	OD_AWS_SQS_INTERVAL              = "OD_AWS_SQS_INTERVAL"
	OD_AWS_SQS_NAME                  = "OD_AWS_SQS_NAME"
	OD_CACHE_EVICTAGE                = "OD_CACHE_EVICTAGE"
	OD_CACHE_HIGHWATERMARK           = "OD_CACHE_HIGHWATERMARK"
	OD_CACHE_LOWWATERMARK            = "OD_CACHE_LOWWATERMARK"
	OD_CACHE_PARTITION               = "OD_CACHE_PARTITION"
	OD_CACHE_ROOT                    = "OD_CACHE_ROOT"
	OD_CACHE_WALKSLEEP               = "OD_CACHE_WALKSLEEP"
	OD_DB_CA                         = "OD_DB_CA"
	OD_DB_CERT                       = "OD_DB_CERT"
	OD_DB_CONN_PARAMS                = "OD_DB_CONN_PARAMS"
	OD_DB_CONNMAXLIFETIME            = "OD_DB_CONNMAXLIFETIME"
	OD_DB_DEADLOCK_RETRYCOUNTER      = "OD_DB_DEADLOCK_RETRYCOUNTER"
	OD_DB_DEADLOCK_RETRYDELAYMS      = "OD_DB_DEADLOCK_RETRYDELAYMS"
	OD_DB_DRIVER                     = "OD_DB_DRIVER"
	OD_DB_HOST                       = "OD_DB_HOST"
	OD_DB_KEY                        = "OD_DB_KEY"
	OD_DB_MAXIDLECONNS               = "OD_DB_MAXIDLECONNS"
	OD_DB_MAXOPENCONNS               = "OD_DB_MAXOPENCONNS"
	OD_DB_PASSWORD                   = "OD_DB_PASSWORD"
	OD_DB_PORT                       = "OD_DB_PORT"
	OD_DB_PROTOCOL                   = "OD_DB_PROTOCOL"
	OD_DB_SCHEMA                     = "OD_DB_SCHEMA"
	OD_DB_USE_TLS                    = "OD_DB_USE_TLS"
	OD_DB_USERNAME                   = "OD_DB_USERNAME"
	OD_ENCRYPT_MASTERKEY             = "OD_ENCRYPT_MASTERKEY"
	OD_EVENT_KAFKA_ADDRS             = "OD_EVENT_KAFKA_ADDRS"
	OD_EVENT_PUBLISH_FAILURE_ACTIONS = "OD_EVENT_PUBLISH_FAILURE_ACTIONS"
	OD_EVENT_PUBLISH_SUCCESS_ACTIONS = "OD_EVENT_PUBLISH_SUCCESS_ACTIONS"
	OD_EVENT_TOPIC                   = "OD_EVENT_TOPIC"
	OD_EVENT_ZK_ADDRS                = "OD_EVENT_ZK_ADDRS"
	OD_EXTERNAL_HOST                 = "OD_EXTERNAL_HOST"
	OD_EXTERNAL_PORT                 = "OD_EXTERNAL_PORT"
	OD_LOG_LEVEL                     = "OD_LOG_LEVEL"
	OD_LOG_LOCATION                  = "OD_LOG_LOCATION"
	OD_LOG_MODE                      = "OD_LOG_MODE"
	OD_PEER_CN                       = "OD_PEER_CN"
	OD_PEER_INSECURE_SKIP_VERIFY     = "OD_PEER_INSECURE_SKIP_VERIFY"
	OD_PEER_SIGNIFIER                = "OD_PEER_SIGNIFIER"
	OD_SERVER_ACL_WHITELIST          = "OD_SERVER_ACL_WHITELIST"
	OD_SERVER_BASEPATH               = "OD_SERVER_BASEPATH"
	OD_SERVER_BINDADDRESS            = "OD_SERVER_BINDADDRESS"
	OD_SERVER_CA                     = "OD_SERVER_CA"
	OD_SERVER_CERT                   = "OD_SERVER_CERT"
	OD_SERVER_CIPHERS                = "OD_SERVER_CIPHERS"
	OD_SERVER_KEY                    = "OD_SERVER_KEY"
	OD_SERVER_PORT                   = "OD_SERVER_PORT"
	OD_SERVER_STATIC_ROOT            = "OD_SERVER_STATIC_ROOT"
	OD_SERVER_TEMPLATE_ROOT          = "OD_SERVER_TEMPLATE_ROOT"
	OD_SERVER_TIMEOUT_IDLE           = "OD_SERVER_TIMEOUT_IDLE"
	OD_SERVER_TIMEOUT_READ           = "OD_SERVER_TIMEOUT_READ"
	OD_SERVER_TIMEOUT_READHEADER     = "OD_SERVER_TIMEOUT_READHEADER"
	OD_SERVER_TIMEOUT_WRITE          = "OD_SERVER_TIMEOUT_WRITE"
	OD_TOKENJAR_LOCATION             = "OD_TOKENJAR_LOCATION"
	OD_TOKENJAR_PASSWORD             = "OD_TOKENJAR_PASSWORD"
	OD_ZK_AAC                        = "OD_ZK_AAC"
	OD_ZK_ANNOUNCE                   = "OD_ZK_ANNOUNCE"
	OD_ZK_MYIP                       = "OD_ZK_MYIP"
	OD_ZK_MYPORT                     = "OD_ZK_MYPORT"
	OD_ZK_RECHECK_TIME               = "OD_ZK_RECHECK_TIME"
	OD_ZK_RETRYDELAY                 = "OD_ZK_RETRYDELAY"
	OD_ZK_TIMEOUT                    = "OD_ZK_TIMEOUT"
	OD_ZK_URL                        = "OD_ZK_URL"
)

// Vars must contain every const. We should be able to use the values in this slice
// to inspect all the config in the current environment provided by env vars.
var Vars = []string{OD_AAC_CA,
	OD_AAC_CERT,
	OD_AAC_CN,
	OD_AAC_HEALTHCHECK,
	OD_AAC_HOST,
	OD_AAC_INSECURE_SKIP_VERIFY,
	OD_AAC_KEY,
	OD_AAC_PORT,
	OD_AAC_RECHECK_TIME,
	OD_AAC_WARMUP_TIME,
	OD_AAC_ZK_ADDRS,
	OD_AWS_ACCESS_KEY_ID,
	OD_AWS_ASG_EC2,
	OD_AWS_ASG_ENDPOINT,
	OD_AWS_ASG_NAME,
	OD_AWS_CLOUDWATCH_ENDPOINT,
	OD_AWS_CLOUDWATCH_INTERVAL,
	OD_AWS_CLOUDWATCH_NAME,
	OD_AWS_REGION,
	OD_AWS_S3_BUCKET,
	OD_AWS_S3_ENDPOINT,
	OD_AWS_S3_FETCH_MB,
	OD_AWS_SECRET_ACCESS_KEY,
	OD_AWS_SQS_BATCHSIZE,
	OD_AWS_SQS_ENDPOINT,
	OD_AWS_SQS_INTERVAL,
	OD_AWS_SQS_NAME,
	OD_CACHE_EVICTAGE,
	OD_CACHE_HIGHWATERMARK,
	OD_CACHE_LOWWATERMARK,
	OD_CACHE_PARTITION,
	OD_CACHE_ROOT,
	OD_CACHE_WALKSLEEP,
	OD_DB_CA,
	OD_DB_CERT,
	OD_DB_CONN_PARAMS,
	OD_DB_CONNMAXLIFETIME,
	OD_DB_DEADLOCK_RETRYCOUNTER,
	OD_DB_DEADLOCK_RETRYDELAYMS,
	OD_DB_DRIVER,
	OD_DB_HOST,
	OD_DB_KEY,
	OD_DB_MAXIDLECONNS,
	OD_DB_MAXOPENCONNS,
	OD_DB_PASSWORD,
	OD_DB_PORT,
	OD_DB_PROTOCOL,
	OD_DB_SCHEMA,
	OD_DB_USE_TLS,
	OD_DB_USERNAME,
	OD_ENCRYPT_MASTERKEY,
	OD_EVENT_KAFKA_ADDRS,
	OD_EVENT_PUBLISH_FAILURE_ACTIONS,
	OD_EVENT_PUBLISH_SUCCESS_ACTIONS,
	OD_EVENT_TOPIC,
	OD_EVENT_ZK_ADDRS,
	OD_EXTERNAL_HOST,
	OD_EXTERNAL_PORT,
	OD_LOG_LEVEL,
	OD_LOG_LOCATION,
	OD_LOG_MODE,
	OD_PEER_CN,
	OD_PEER_SIGNIFIER,
	OD_PEER_INSECURE_SKIP_VERIFY,
	OD_SERVER_ACL_WHITELIST,
	OD_SERVER_BASEPATH,
	OD_SERVER_BINDADDRESS,
	OD_SERVER_CA,
	OD_SERVER_CERT,
	OD_SERVER_CIPHERS,
	OD_SERVER_KEY,
	OD_SERVER_PORT,
	OD_SERVER_STATIC_ROOT,
	OD_SERVER_TEMPLATE_ROOT,
	OD_SERVER_TIMEOUT_IDLE,
	OD_SERVER_TIMEOUT_READ,
	OD_SERVER_TIMEOUT_READHEADER,
	OD_SERVER_TIMEOUT_WRITE,
	OD_TOKENJAR_LOCATION,
	OD_TOKENJAR_PASSWORD,
	OD_ZK_AAC,
	OD_ZK_ANNOUNCE,
	OD_ZK_MYIP,
	OD_ZK_MYPORT,
	OD_ZK_RECHECK_TIME,
	OD_ZK_RETRYDELAY,
	OD_ZK_TIMEOUT,
	OD_ZK_URL,
}

// PrintODEnvironment prints the content of all environment variables required
// by odrive. Sensitive values are redacted
func PrintODEnvironment() {
	var filtered = []string{
		OD_AWS_ACCESS_KEY_ID,
		OD_AWS_SECRET_ACCESS_KEY,
		OD_ENCRYPT_MASTERKEY,
		OD_DB_PASSWORD,
	}
	redact := func(envVar, value string) string {
		for _, restricted := range filtered {
			if envVar == restricted {
				return "<redacted>"
			}
		}
		return value
	}
	fmt.Println("object-drive environment variables. Number of vars:", len(Vars))
	for _, variable := range Vars {
		fmt.Printf("%s=%s\n", variable, redact(variable, os.Getenv(variable)))
	}
}

// GenerateStartScript creates a bash script that can be used
// as a template with all the variables exported and then running
// the odrive binary with redirected output for logging
func GenerateStartScript() {
	tmpl, err := template.New("script").Parse(`#!/bin/bash

{{ range $i, $v := .Variables }}export {{ $v }}=
{{ end }}

# odrive must be on your PATH
odrive --conf /opt/services/object-drive/odrive.yml \ 
       --staticRoot /opt/services/object-drive/libs/server/static \
	   --templateDir /opt/services/object-drive/libs/server/static/templates &>> /opt/services/object-drive/log/object-drive.log 2>&1&

`)
	exitOnErr(err)
	data := struct{ Variables []string }{Variables: Vars}
	tmpl.Execute(os.Stdout, data)
}

// GenerateSourceEnvScript creates a bash script that can be used
// as a template ith all the variables exported.
func GenerateSourceEnvScript() {
	tmpl, err := template.New("script").Parse(`#!/bin/bash

#
# Please review /etc/init.d/object-drive-1.0 for default logging location if OD_LOG_LOCATION is not set
#

{{ range $i, $v := .Variables }}export {{ $v }}=
{{ end }}

`)
	exitOnErr(err)
	data := struct{ Variables []string }{Variables: Vars}
	tmpl.Execute(os.Stdout, data)
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
