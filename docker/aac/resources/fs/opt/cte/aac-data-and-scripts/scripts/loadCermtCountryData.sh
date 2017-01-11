#!/bin/sh

# default to localhost
REDIS_SERVER="localhost"

if [ ! -z $1 ] ; then
    REDIS_SERVER=$1
fi

REDIS_PORT=6379
if [ ! -z $2 ] ; then
    REDIS_PORT=$2
fi

REDIS_PASSWORD=
if [ ! -z $3 ] ; then
    REDIS_PASSWORD=$3
fi

if [ -z `which redis-cli` ] ; then
    echo "redis-cli is not in the path"
    exit 1
fi


TRI_TO_AOR_FILE="trigraph-to-aor.data"
if [ ! -f $TRI_TO_AOR_FILE ] ; then
    echo "$TRI_TO_AOR_FILE is not present in the current directory"
    exit 1
fi

COUNTRYNAME_TO_TRI_GEONAMES_FILE="countryname-to-tri-geonames.data"
if [ ! -f $COUNTRYNAME_TO_TRI_GEONAMES_FILE ] ; then
    echo "$COUNTRYNAME_TO_TRI_GEONAMES_FILE is not present in the current directory"
    exit 1
fi

COUNTRYNAME_TO_TRI_MANUAL_FILE="countryname-to-tri-manual.data"
if [ ! -f $COUNTRYNAME_TO_TRI_MANUAL_FILE ] ; then
    echo "$COUNTRYNAME_TO_TRI_MANUAL_FILE is not present in the current directory"
    exit 1
fi


if [ ! -z ${REDIS_PASSWORD} ] ; then
REDIS_CMD="redis-cli -n 2 -h $REDIS_SERVER -p ${REDIS_PORT} -a ${REDIS_PASSWORD}"
else
REDIS_CMD="redis-cli -n 2 -h $REDIS_SERVER -p ${REDIS_PORT}"
fi



echo "Loading trigraph to AOR mapping data into Redis instance on $REDIS_SERVER"
$REDIS_CMD KEYS "TRI-TO-AOR:*" | xargs $REDIS_CMD DEL
cat $TRI_TO_AOR_FILE | $REDIS_CMD
$REDIS_CMD DEL "TRI-TO-AOR-KEYSET"
$REDIS_CMD KEYS "TRI-TO-AOR:*" | xargs $REDIS_CMD SADD "TRI-TO-AOR-KEYSET"


echo "Loading countryname to trigraph mapping data, from 2 files, into Redis instance on $REDIS_SERVER"
$REDIS_CMD KEYS "COUNTRYNAME_TO_TRI:*" | xargs $REDIS_CMD DEL
sed -e '/^#/d;/^$/d' $COUNTRYNAME_TO_TRI_MANUAL_FILE | cat $COUNTRYNAME_TO_TRI_GEONAMES_FILE - | $REDIS_CMD
