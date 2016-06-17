#!/bin/bash
set -e

# This script will connect to mysql/mariadb on aws

MYSQL_MASTER_USER="odrivemaster"
MYSQL_MASTER_PASSWORD="cU9WiNVsGacXbWVaEB5m"
MYSQL_ENDPOINT="dev-odrive-mysql.c2bdxmcv8gbh.us-east-1.rds.amazonaws.com"
MYSQL_PORT="3306"
MYSQL_DATABASE_NAME="metadatadb"

mysql -h ${MYSQL_ENDPOINT} --protocol=tcp --database=${MYSQL_DATABASE_NAME} -u${MYSQL_MASTER_USER} -p${MYSQL_MASTER_PASSWORD}
