#!/bin/bash
set -e

# This script will connect to mysql/mariadb on aws

MYSQL_MASTER_USER="odrivemaster"
MYSQL_MASTER_PASSWORD="S4m976zGuZyH"
MYSQL_ENDPOINT="dev-odrive-mariadb.c2bdxmcv8gbh.us-east-1.rds.amazonaws.com"
MYSQL_PORT="3306"
MYSQL_DATABASE_NAME="metadatadb"
#MYSQL_SSL_CA_PATH="$GOPATH/src/decipher.com/object-drive-server/defaultcerts/aws/rds-ca-2015-us-east-1-and-root.pem"
MYSQL_SSL_CA_PATH="$GOPATH/src/decipher.com/object-drive-server/defaultcerts/aws/rds-combined-ca-bundle.pem"

mysql -h ${MYSQL_ENDPOINT} --protocol=tcp --database=${MYSQL_DATABASE_NAME} -u${MYSQL_MASTER_USER} -p${MYSQL_MASTER_PASSWORD} --ssl-ca=${MYSQL_SSL_CA_PATH}

