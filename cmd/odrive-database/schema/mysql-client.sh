#!/bin/bash
set -e

# This script will connect to mysql/mariadb in the docker container named metadatadb
if [ $# -ne 0 ]
then
  DOCKER_CONTAINER_IP=$1
else 
  DOCKER_CONTAINER_NAME=`docker ps --format '{{.Names}}' | grep metadatadb`
  DOCKER_CONTAINER_IP=`docker inspect --format '{{ .NetworkSettings.IPAddress }}' ${DOCKER_CONTAINER_NAME}`
fi
MYSQL_USER="dbuser"
MYSQL_PASSWORD="dbPassword"
MYSQL_DATABASE="metadatadb"
CERT_PATH=$GOPATH/src/github.com/deciphernow/object-drive-server/docker/metadatadb

# Uncomment the following to require server cert verification, but this wont work unless the certs include the IP address
#SSL_VERIFY=--ssl-verify-server-cert

mysql -h ${DOCKER_CONTAINER_IP} --protocol=tcp --database=${MYSQL_DATABASE} --ssl --ssl-ca ${CERT_PATH}/ca.pem --ssl-cert ${CERT_PATH}/client-cert.pem --ssl-key ${CERT_PATH}/client-key.pem -u${MYSQL_USER} -p${MYSQL_PASSWORD} ${SSL_VERIFY}
