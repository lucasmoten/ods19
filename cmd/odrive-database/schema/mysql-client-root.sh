#!/bin/bash
set -e

# This script will connect to mysql/mariadb in the docker container named metadatadb

MYSQL_USER="root"
MYSQL_PASSWORD="dbRootPassword"

#MYSQL_IP=`docker inspect --format '{{ .NetworkSettings.IPAddress }}' metadatadb`
#mysql -h ${MYSQL_IP} --protocol=tcp --database=metadatadb --ssl --ssl-ca ../ca.pem --ssl-cert ../client-cert.pem --ssl-key ../client-key.pem -u${MYSQL_USER} -p${MYSQL_PASSWORD}

# This uses the name assigned to the server cert and forces verification
mysql -h fqdn.for.metadatadb.local --protocol=tcp --database=metadatadb --ssl --ssl-ca ../ca.pem --ssl-cert ../client-cert.pem --ssl-key ../client-key.pem --ssl-verify-server-cert -u${MYSQL_USER} -p${MYSQL_PASSWORD}

