#!/bin/bash

set -e

# This script is used by the docker container to initialize the schema
# Developers logging in locally should use mysql-local.sh for 127.0.0.1, or mysql-client.sh to connect to the container

# TODO: Route the MYSQL_USER and MYSQL_PASSWORD from the environment variables
# TODO: Set the host name to match the cert instead of 127.0.0.1 and turn on --ssl-verify-server-cert


MYSQL_USER="dbuser"
MYSQL_PASSWORD="dbPassword"
# These credentials used to perform a repair on the mysql database for resolving issues
# if the docker container is halted that result in the following error:
# Error 145: Table './mysql/proc' is marked as crashed and should be repaired
# The command can only be performed as root: repair table mysql.proc
# This script performs a repair via mysqlcheck on startup.
MYSQL_ROOT_USER="root"
MYSQL_ROOT_PASSWORD="dbRootPassword"


echo "Handling schema"

echo "Pinging database to see if available"
until mysqladmin ping -h 127.0.0.1 --protocol=tcp --ssl --ssl-ca /ca.pem --ssl-cert /client-cert.pem --ssl-key /client-key.pem -u${MYSQL_USER} -p${MYSQL_PASSWORD}
do
  /bin/sleep 2;
done;

echo "Repairing tables"
mysqlcheck -h 127.0.0.1 --protocol=tcp --repair --databases mysql --ssl --ssl-ca /ca.pem --ssl-cert /client-cert.pem --ssl-key /client-key.pem -u${MYSQL_ROOT_USER} -p${MYSQL_ROOT_PASSWORD}


SCHEMAPOPULATED=0
mysqlshow "metadatadb" "object" "encryptIV" -h 127.0.0.1 --protocol=tcp --ssl --ssl-ca /ca.pem --ssl-cert /client-cert.pem --ssl-key /client-key.pem -u${MYSQL_USER} -p${MYSQL_PASSWORD} | grep encryptIV > /dev/null 2>&1 && SCHEMAPOPULATED=1
if [ $SCHEMAPOPULATED -lt 1 ]
then
  echo "Installing schema"
  cd /schema
  mysql -h 127.0.0.1 --protocol=tcp --database=metadatadb --ssl --ssl-ca /ca.pem --ssl-cert /client-cert.pem --ssl-key /client-key.pem -u${MYSQL_USER} -p${MYSQL_PASSWORD} --vertical < triggers.drop.sql
  mysql -h 127.0.0.1 --protocol=tcp --database=metadatadb --ssl --ssl-ca /ca.pem --ssl-cert /client-cert.pem --ssl-key /client-key.pem -u${MYSQL_USER} -p${MYSQL_PASSWORD} --vertical < functions.drop.sql
  mysql -h 127.0.0.1 --protocol=tcp --database=metadatadb --ssl --ssl-ca /ca.pem --ssl-cert /client-cert.pem --ssl-key /client-key.pem -u${MYSQL_USER} -p${MYSQL_PASSWORD} --vertical < constraints.drop.sql
  mysql -h 127.0.0.1 --protocol=tcp --database=metadatadb --ssl --ssl-ca /ca.pem --ssl-cert /client-cert.pem --ssl-key /client-key.pem -u${MYSQL_USER} -p${MYSQL_PASSWORD} --vertical < tables.create.sql
  mysql -h 127.0.0.1 --protocol=tcp --database=metadatadb --ssl --ssl-ca /ca.pem --ssl-cert /client-cert.pem --ssl-key /client-key.pem -u${MYSQL_USER} -p${MYSQL_PASSWORD} --vertical < constraints.create.sql
  mysql -h 127.0.0.1 --protocol=tcp --database=metadatadb --ssl --ssl-ca /ca.pem --ssl-cert /client-cert.pem --ssl-key /client-key.pem -u${MYSQL_USER} -p${MYSQL_PASSWORD} --vertical < functions.create.sql
  mysql -h 127.0.0.1 --protocol=tcp --database=metadatadb --ssl --ssl-ca /ca.pem --ssl-cert /client-cert.pem --ssl-key /client-key.pem -u${MYSQL_USER} -p${MYSQL_PASSWORD} --vertical < triggers.create.sql
  mysql -h 127.0.0.1 --protocol=tcp --database=metadatadb --ssl --ssl-ca /ca.pem --ssl-cert /client-cert.pem --ssl-key /client-key.pem -u${MYSQL_USER} -p${MYSQL_PASSWORD} --vertical < dbstate-init.sql
  if [ $? -eq 0 ]
  then
    echo "Schema population complete."
  else
    echo "Schema failed to initialize."
  fi
  cd /
else
  echo "Schema already present"
fi

