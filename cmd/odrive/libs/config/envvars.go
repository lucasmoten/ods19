package config

import (
	"fmt"
	"os"
)

// Environment variables
const (
	OD_AAC_CA                      = "OD_AAC_CA"
	OD_AAC_CERT                    = "OD_AAC_CERT"
	OD_AAC_HOST                    = "OD_AAC_HOST"
	OD_AAC_KEY                     = "OD_AAC_KEY"
	OD_AAC_PORT                    = "OD_AAC_PORT"
	OD_AWS_ACCESS_KEY_ID           = "OD_AWS_ACCESS_KEY_ID"
	OD_AWS_REGION                  = "OD_AWS_REGION"
	OD_AWS_PROFILE                 = "OD_AWS_PROFILE"
	OD_AWS_SESSION_TOKEN           = "OD_AWS_SESSION_TOKEN"
	OD_AWS_SHARED_CREDENTIALS_FILE = "OD_AWS_SHARED_CREDENTIALS_FILE"
	OD_AWS_S3_BUCKET               = "OD_AWS_S3_BUCKET"
	OD_AWS_SECRET_ACCESS_KEY       = "OD_AWS_SECRET_ACCESS_KEY"
	OD_CACHE_EVICTAGE              = "OD_CACHE_EVICTAGE"
	OD_CACHE_HIGHWATERMARK         = "OD_CACHE_HIGHWATERMARK"
	OD_CACHE_LOWWATERMARK          = "OD_CACHE_LOWWATERMARK"
	OD_CACHE_PARTITION             = "OD_CACHE_PARTITION"
	OD_CACHE_ROOT                  = "OD_CACHE_ROOT"
	OD_CACHE_WALKSLEEP             = "OD_CACHE_WALKSLEEP"
	OD_DB_CA                       = "OD_DB_CA"
	OD_DB_CERT                     = "OD_DB_CERT"
	OD_DB_HOST                     = "OD_DB_HOST"
	OD_DB_KEY                      = "OD_DB_KEY"
	OD_DB_MAXIDLECONNS             = "OD_DB_MAXIDLECONNS"
	OD_DB_MAXOPENCONNS             = "OD_DB_MAXOPENCONNS"
	OD_DB_PASSWORD                 = "OD_DB_PASSWORD"
	OD_DB_PORT                     = "OD_DB_PORT"
	OD_DB_SCHEMA                   = "OD_DB_SCHEMA"
	OD_DB_USERNAME                 = "OD_DB_USERNAME"
	OD_DOCKERVM_OVERRIDE           = "OD_DOCKERVM_OVERRIDE"
	OD_DOCKERVM_PORT               = "OD_DOCKERVM_PORT"
	OD_ENCRYPT_MASTERKEY           = "OD_ENCRYPT_MASTERKEY"
	OD_SERVER_CA                   = "OD_SERVER_CA"
	OD_SERVER_CERT                 = "OD_SERVER_CERT"
	OD_SERVER_KEY                  = "OD_SERVER_KEY"
	OD_SERVER_PORT                 = "OD_SERVER_PORT"
	OD_STANDALONE                  = "OD_STANDALONE"
	OD_ZK_AAC                      = "OD_ZK_AAC"
	OD_ZK_BASEPATH                 = "OD_ZK_BASEPATH"
	OD_ZK_MYIP                     = "OD_ZK_MYIP"
	OD_ZK_MYPORT                   = "OD_ZK_MYPORT"
	OD_ZK_ROOT                     = "OD_ZK_ROOT"
	OD_ZK_TIMEOUT                  = "OD_ZK_TIMEOUT"
	OD_ZK_URL                      = "OD_ZK_URL"
)

// PrintODEnvironment prints the content of all environment variables required
// by odrive.
func PrintODEnvironment() {
	var vars = []string{OD_AAC_CA,
		OD_AAC_CERT,
		OD_AAC_HOST,
		OD_AAC_KEY,
		OD_AAC_PORT,
		OD_AWS_ACCESS_KEY_ID,
		OD_AWS_REGION,
		OD_AWS_PROFILE,
		OD_AWS_SESSION_TOKEN,
		OD_AWS_SHARED_CREDENTIALS_FILE,
		OD_AWS_S3_BUCKET,
		OD_AWS_SECRET_ACCESS_KEY,
		OD_CACHE_EVICTAGE,
		OD_CACHE_HIGHWATERMARK,
		OD_CACHE_LOWWATERMARK,
		OD_CACHE_PARTITION,
		OD_CACHE_ROOT,
		OD_CACHE_WALKSLEEP,
		OD_DB_CA,
		OD_DB_CERT,
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
		OD_SERVER_CA,
		OD_SERVER_CERT,
		OD_SERVER_KEY,
		OD_SERVER_PORT,
		OD_STANDALONE,
		OD_ZK_AAC,
		OD_ZK_BASEPATH,
		OD_ZK_MYIP,
		OD_ZK_MYPORT,
		OD_ZK_ROOT,
		OD_ZK_TIMEOUT,
		OD_ZK_URL,
	}
	fmt.Println("odrive environment variables. Number of vars:", len(vars))
	for _, variable := range vars {
		fmt.Printf("%s=%s\n", variable, os.Getenv(variable))
	}

}
