#!/bin/bash
set -e

# This script will connect to mysql/mariadb on aws

MYSQL_MASTER_USER=
MYSQL_MASTER_PASSWORD=
MYSQL_ENDPOINT=
MYSQL_PORT="3306"
MYSQL_DATABASE_NAME="metadatadb"
MYSQL_SSL_CA_PATH="$GOPATH/src/decipher.com/object-drive-server/defaultcerts/aws/rds-combined-ca-bundle.pem"

# Uncomment the following to use SSL
SSL_CA="--ssl --ssl-ca $MYSQL_SSL_CA_PATH"

# Uncomment the following to verify server cert
SSL_VERIFY=--ssl-verify-server-cert

mysql -h ${MYSQL_ENDPOINT} --protocol=tcp --database=${MYSQL_DATABASE_NAME} -u${MYSQL_MASTER_USER} -p${MYSQL_MASTER_PASSWORD} ${SSL_CA} ${SSL_VERIFY}
