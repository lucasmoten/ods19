#!/bin/bash
set -e


if [ "${1:0:1}" = '-' ]; then
	set -- mysqld_safe "$@"
fi

BUILDSCHEMA="0"
DATADIR="/var/lib/mysql"
if [ "$1" = 'mysqld_safe' ]; then
	echo "Handling mysqld_safe"
	if [ ! -d "$DATADIR/mysql" ]; then
		if [ -z "$MYSQL_ROOT_PASSWORD" -a -z "$MYSQL_ALLOW_EMPTY_PASSWORD" ]; then
			echo >&2 'error: database is uninitialized and SQL_ROOT_PASSWORD not set'
			echo >&2 '  Did you forget to add -e MYSQL_ROOT_PASSWORD=... ?'
			exit 1
		fi

		echo 'Running mysql_install_db ...'
		mysql_install_db --datadir="$DATADIR"
		echo 'Finished mysql_install_db'

		tempSqlFile='/tmp/mysql-first-time.sql'
		cat > "$tempSqlFile" <<-EOSQL
			DELETE FROM mysql.user ;
			CREATE USER 'root'@'%' IDENTIFIED BY '${MYSQL_ROOT_PASSWORD}';
			GRANT ALL ON *.* TO 'root'@'%' WITH GRANT OPTION ;
			GRANT USAGE ON *.* TO 'root'@'%' REQUIRE X509;
			DROP DATABASE IF EXISTS test ;
		EOSQL

		if [ "$MYSQL_DATABASE" ]; then
			echo "CREATE DATABASE IF NOT EXISTS \`$MYSQL_DATABASE\` DEFAULT CHARACTER SET utf8 DEFAULT COLLATE utf8_general_ci ;" >> "$tempSqlFile"
		fi

		if [ "$MYSQL_USER" -a "$MYSQL_PASSWORD" ]; then
			echo "CREATE USER '$MYSQL_USER'@'%' IDENTIFIED BY '$MYSQL_PASSWORD';" >> "$tempSqlFile"
			echo "GRANT USAGE ON  *.* TO '$MYSQL_USER'@'%' REQUIRE X509;" >> "$tempSqlFile"
			if [ "$MYSQL_DATABASE" ]; then
				echo "GRANT ALL ON \`$MYSQL_DATABASE\`.* TO '$MYSQL_USER'@'%' ;" >> "$tempSqlFile"
			fi
		fi

		echo "FLUSH PRIVILEGES ;" >> "$tempSqlFile"
		echo "USE METADATADB; " >> "$tempSqlFile"

		set -- "$@" --init-file="$tempSqlFile"
		BUILDSCHEMA='1'
	fi

	echo 'Done mysqld_safe'
fi

echo 'Changing ownership of data directory to mysql:mysql'
chown -R mysql:mysql "$DATADIR"
echo 'Ownership changed. Ready to run'

if [ "$BUILDSCHEMA" = '1' ]; then
    echo "Backgrounding database initialization task"
    /usr/local/bin/odrive-database init --force=true --useEmbedded=true &
fi

exec "$@"
