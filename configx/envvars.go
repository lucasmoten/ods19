package config

import (
	"fmt"
	"html/template"
	"os"
)

// Environment variables
const (
	OD_AAC_CA                  = "OD_AAC_CA"
	OD_AAC_CERT                = "OD_AAC_CERT"
	OD_AAC_HOST                = "OD_AAC_HOST"
	OD_AAC_KEY                 = "OD_AAC_KEY"
	OD_AAC_PORT                = "OD_AAC_PORT"
	OD_AWS_ACCESS_KEY_ID       = "OD_AWS_ACCESS_KEY_ID"
	OD_AWS_CLOUDWATCH_ENDPOINT = "OD_AWS_CLOUDWATCH_ENDPOINT"
	OD_AWS_CLOUDWATCH_INTERVAL = "OD_AWS_CLOUDWATCH_INTERVAL"
	OD_AWS_CLOUDWATCH_NAME     = "OD_AWS_CLOUDWATCH_NAME"
	OD_AWS_ENDPOINT            = "OD_AWS_ENDPOINT"
	OD_AWS_REGION              = "OD_AWS_REGION"
	OD_AWS_S3_BUCKET           = "OD_AWS_S3_BUCKET"
	OD_AWS_S3_FETCH_MB         = "OD_AWS_S3_FETCH_MB"
	OD_AWS_SECRET_ACCESS_KEY   = "OD_AWS_SECRET_ACCESS_KEY"
	OD_CACHE_EVICTAGE          = "OD_CACHE_EVICTAGE"
	OD_CACHE_HIGHWATERMARK     = "OD_CACHE_HIGHWATERMARK"
	OD_CACHE_LOWWATERMARK      = "OD_CACHE_LOWWATERMARK"
	OD_CACHE_PARTITION         = "OD_CACHE_PARTITION"
	OD_CACHE_ROOT              = "OD_CACHE_ROOT"
	OD_CACHE_WALKSLEEP         = "OD_CACHE_WALKSLEEP"
	OD_DB_CA                   = "OD_DB_CA"
	OD_DB_CERT                 = "OD_DB_CERT"
	OD_DB_HOST                 = "OD_DB_HOST"
	OD_DB_KEY                  = "OD_DB_KEY"
	OD_DB_CONN_PARAMS          = "OD_DB_CONN_PARAMS"
	OD_DB_MAXIDLECONNS         = "OD_DB_MAXIDLECONNS"
	OD_DB_MAXOPENCONNS         = "OD_DB_MAXOPENCONNS"
	OD_DB_PASSWORD             = "OD_DB_PASSWORD"
	OD_DB_PORT                 = "OD_DB_PORT"
	OD_DB_SCHEMA               = "OD_DB_SCHEMA"
	OD_DB_USERNAME             = "OD_DB_USERNAME"
	OD_DOCKERVM_OVERRIDE       = "OD_DOCKERVM_OVERRIDE"
	OD_DOCKERVM_PORT           = "OD_DOCKERVM_PORT"
	OD_ENCRYPT_MASTERKEY       = "OD_ENCRYPT_MASTERKEY"
	OD_EVENT_KAFKA_ADDRS       = "OD_EVENT_KAFKA_ADDRS"
	OD_LOG_LOCATION            = "OD_LOG_LOCATION"
	OD_SERVER_CA               = "OD_SERVER_CA"
	OD_SERVER_BASEPATH         = "OD_SERVER_BASEPATH"
	OD_SERVER_CERT             = "OD_SERVER_CERT"
	OD_SERVER_KEY              = "OD_SERVER_KEY"
	OD_SERVER_PORT             = "OD_SERVER_PORT"
	OD_ZK_AAC                  = "OD_ZK_AAC"
	OD_ZK_ANNOUNCE             = "OD_ZK_ANNOUNCE"
	OD_ZK_MYIP                 = "OD_ZK_MYIP"
	OD_ZK_MYPORT               = "OD_ZK_MYPORT"
	OD_ZK_TIMEOUT              = "OD_ZK_TIMEOUT"
	OD_ZK_URL                  = "OD_ZK_URL"
)

// Maintain in sync with above consts.
var vars = []string{OD_AAC_CA,
	OD_AAC_CERT,
	OD_AAC_HOST,
	OD_AAC_KEY,
	OD_AAC_PORT,
	OD_AWS_ACCESS_KEY_ID,
	OD_AWS_CLOUDWATCH_ENDPOINT,
	OD_AWS_CLOUDWATCH_INTERVAL,
	OD_AWS_CLOUDWATCH_NAME,
	OD_AWS_ENDPOINT,
	OD_AWS_REGION,
	OD_AWS_S3_BUCKET,
	OD_AWS_S3_FETCH_MB,
	OD_AWS_SECRET_ACCESS_KEY,
	OD_CACHE_EVICTAGE,
	OD_CACHE_HIGHWATERMARK,
	OD_CACHE_LOWWATERMARK,
	OD_CACHE_PARTITION,
	OD_CACHE_ROOT,
	OD_CACHE_WALKSLEEP,
	OD_DB_CA,
	OD_DB_CERT,
	OD_DB_CONN_PARAMS,
	OD_DB_HOST,
	OD_DB_KEY,
	OD_DB_MAXIDLECONNS,
	OD_DB_MAXOPENCONNS,
	OD_DB_PASSWORD,
	OD_DB_PORT,
	OD_DB_SCHEMA,
	OD_DB_USERNAME,
	OD_DOCKERVM_OVERRIDE,
	OD_DOCKERVM_PORT,
	OD_ENCRYPT_MASTERKEY,
	OD_EVENT_KAFKA_ADDRS,
	OD_LOG_LOCATION,
	OD_SERVER_BASEPATH,
	OD_SERVER_CA,
	OD_SERVER_CERT,
	OD_SERVER_KEY,
	OD_SERVER_PORT,
	OD_ZK_AAC,
	OD_ZK_ANNOUNCE,
	OD_ZK_MYIP,
	OD_ZK_MYPORT,
	OD_ZK_TIMEOUT,
	OD_ZK_URL,
}

// PrintODEnvironment prints the content of all environment variables required
// by odrive.
func PrintODEnvironment() {
	fmt.Println("odrive environment variables. Number of vars:", len(vars))
	for _, variable := range vars {
		fmt.Printf("%s=%s\n", variable, os.Getenv(variable))
	}
}

func GenerateStartScript() {
	tmpl, err := template.New("script").Parse(`#!/bin/bash

{{ range $i, $v := .Variables }}export {{ $v }}=
{{ end }}

# odrive must be on your PATH
odrive --conf /etc/odrive/odrive.yml \ 
       --staticRoot /etc/odrive/libs/server/static \
	   --templateDir /etc/odrive/libs/server/static/templates &>> /opt/odrive/log/object-drive.log 2>&1&

`)
	exitOnErr(err)
	data := struct{ Variables []string }{Variables: vars}
	tmpl.Execute(os.Stdout, data)
}
func GenerateSourceEnvScript() {
	tmpl, err := template.New("script").Parse(`#!/bin/bash

#
# Please review /etc/init.d/odrive for default logging location if OD_LOG_LOCATION is not set
#

{{ range $i, $v := .Variables }}export {{ $v }}=
{{ end }}

`)
	exitOnErr(err)
	data := struct{ Variables []string }{Variables: vars}
	tmpl.Execute(os.Stdout, data)
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
