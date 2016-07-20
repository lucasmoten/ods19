#!/bin/bash
set -e

if [ "x$OD_DB_AWS_MYSQL_MASTER_USER" == "x" ]
then
  export OD_DB_AWS_MYSQL_MASTER_USER=$OD_DB_USERNAME
fi

if [ "x$OD_DB_AWS_MYSQL_MASTER_PASSWORD" == "x" ]
then
  export OD_DB_AWS_MYSQL_MASTER_PASSWORD=$OD_DB_PASSWORD
fi

if [ "x$OD_DB_AWS_MYSQL_ENDPOINT" == "x" ]
then
  export OD_DB_AWS_MYSQL_ENDPOINT=$OD_DB_HOST
fi

if [ "x$OD_DB_AWS_MYSQL_PORT" == "x" ]
then
  export OD_DB_AWS_MYSQL_PORT=$OD_DB_PORT
fi

if [ "x$OD_DB_AWS_MYSQL_DATABASE_NAME" == "x" ]
then
  export OD_DB_AWS_MYSQL_DATABASE_NAME=$OD_DB_SCHEMA
fi

if [ "x$OD_DB_AWS_MYSQL_SSL_CA_PATH" == "x" ]
then
  export OD_DB_AWS_MYSQL_SSL_CA_PATH=$OD_DB_CA
fi

# Uncomment the following to use SSL
SSL_CA="--ssl --ssl-ca $OD_DB_AWS_MYSQL_SSL_CA_PATH"

# Uncomment the following to verify server cert
SSL_VERIFY=--ssl-verify-server-cert

mysql -h ${OD_DB_AWS_MYSQL_ENDPOINT} --protocol=tcp --database=${OD_DB_AWS_MYSQL_DATABASE_NAME} -u${OD_DB_AWS_MYSQL_MASTER_USER} -p${OD_DB_AWS_MYSQL_MASTER_PASSWORD} ${SSL_CA} ${SSL_VERIFY}
