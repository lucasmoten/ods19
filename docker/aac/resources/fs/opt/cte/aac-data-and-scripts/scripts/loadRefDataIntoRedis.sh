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

TERTAGRAPH_FILE="tetragraph-to-trigraph.data"
if [ ! -f $TERTAGRAPH_FILE ] ; then
    echo "$TERTAGRAPH_FILE is not present in the current directory"
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

PROVINCE_FILE="trigraph-to-province.data"
if [ ! -f $PROVINCE_FILE ] ; then
    echo "$PROVINCE_FILE is not present in the current directory"
    exit 1
fi

CLASSIF_FILE="classif-to-rank.data"
if [ ! -f $CLASSIF_FILE ] ; then
    echo "$CLASSIF_FILE is not present in the current directory"
    exit 1
fi

MIME_TO_EXT_FILE="mime-types-to-file-ext.data"
if [ ! -f $MIME_TO_EXT_FILE ] ; then
    echo "$MIME_TO_EXT_FILE is not present in the current directory"
    exit 1
fi

PDF_CONV_FILE_TYPES_FILE="pdf-conversion-file-types.data"
if [ ! -f $PDF_CONV_FILE_TYPES_FILE ] ; then
    echo "$PDF_CONV_FILE_TYPES_FILE is not present in the current directory"
    exit 1
fi

SCI_CONTROLS_FILE="sci-controls.data"
if [ ! -f $SCI_CONTROLS_FILE ] ; then
    echo "$SCI_CONTROLS_FILE is not present in the current directory"
    exit 1
fi

WORD_CONV_FILE_TYPES_FILE="word-conversion-file-types.data"
if [ ! -f $WORD_CONV_FILE_TYPES_FILE ] ; then
    echo "$WORD_CONV_FILE_TYPES_FILE is not present in the current directory"
exit 1
fi

COUNTRIES_STATES_FILE="COUNTRIES_STATES.data"
if [ ! -f $COUNTRIES_STATES_FILE ] ; then
    echo "$COUNTRIES_STATES_FILE is not present in the current directory"
    exit 1
fi

NIPF_FILE="NIPF.data"
if [ ! -f $NIPF_FILE ] ; then
	echo "$NIPF_FILE is not present in the current directory"
	exit 1
fi

LEADTYPES_FILE="LEADTYPES.data"
if [ ! -f $LEADTYPES_FILE ] ; then
	echo "$LEADTYPES_FILE is not present in the current directory"
	exit 1
fi

ETHNIC_GROUPS_FILE="Ethnic_Groups.data"
if [ ! -f $ETHNIC_GROUPS_FILE ] ; then
	echo "$ETHNIC_GROUPS_FILE is not present in the current directory"
	exit 1
fi

LANGUAGES_FILE="Languages.data"
if [ ! -f $LANGUAGES_FILE ] ; then
	echo "$LANGUAGES_FILE is not present in the current directory"
	exit 1
fi

STATES_FILE="States.data"
if [ ! -f $STATES_FILE ] ; then
	echo "$STATES_FILE is not present in the current directory"
	exit 1
fi

NATIONALITIES_FILE="Nationalities.data"
if [ ! -f $NATIONALITIES_FILE ] ; then
	echo "$NATIONALITIES_FILE is not present in the current directory"
	exit 1
fi


AOR_FILE="aor.data"
if [ ! -f $AOR_FILE ] ; then
	echo "$AOR_FILE is not present in the current directory"
	exit 1
fi

CITY_FILE="Cities.data"
if [ ! -f $CITY_FILE ] ; then
	echo "$CITY_FILE is not present in the current directory"
	exit 1
fi

ENEMY_NETWORKS_FILE="enemy_networks.data"
if [ ! -f $ENEMY_NETWORKS_FILE ] ; then
	echo "$ENEMY_NETWORKS_FILE is not present in the current directory"
	exit 1
fi

THREAT_PROGRESSION_PHASES_FILE="threat_progression_phases.data"
if [ ! -f $THREAT_PROGRESSION_PHASES_FILE ] ; then
	echo "$THREAT_PROGRESSION_PHASES_FILE is not present in the current directory"
	exit 1
fi

IICT_TYPES_FILE="iict_types.data"
if [ ! -f $IICT_TYPES_FILE ] ; then
	echo "$IICT_TYPES_FILE is not present in the current directory"
	exit 1
fi

THREAT_LEVELS_FILE="threat_levels.data"
if [ ! -f $THREAT_LEVELS_FILE ] ; then
	echo "$THREAT_LEVELS_FILE is not present in the current directory"
	exit 1
fi

TOC_STATUSES_FILE="toc_statuses.data"
if [ ! -f $TOC_STATUSES_FILE ] ; then
	echo "$TOC_STATUSES_FILE is not present in the current directory"
	exit 1
fi

TOC_TARGETS_FILE="toc_targets.data"
if [ ! -f $TOC_TARGETS_FILE ] ; then
	echo "$TOC_TARGETS_FILE is not present in the current directory"
	exit 1
fi

DTST_TARGETS_FILE="dtst_targets.data"
if [ ! -f $DTST_TARGETS_FILE ] ; then
	echo "$DTST_TARGETS_FILE is not present in the current directory"
	exit 1
fi

ORCON_FILE="orcon.data"
if [ ! -f $ORCON_FILE ] ; then
    echo "$ORCON_FILE is not present in the current directory"
exit 1
fi


if [ ! -z ${REDIS_PASSWORD} ] ; then
REDIS_CMD="redis-cli -n 2 -h $REDIS_SERVER -p ${REDIS_PORT} -a ${REDIS_PASSWORD}"
else
REDIS_CMD="redis-cli -n 2 -h $REDIS_SERVER -p ${REDIS_PORT}"
fi

# delete existing data and add new ones
echo "Loading tetragraph to trigraph mapping data into Redis instance on $REDIS_SERVER"
$REDIS_CMD KEYS "TETRA-TO-TRI:*" | xargs $REDIS_CMD DEL
cat $TERTAGRAPH_FILE | $REDIS_CMD
$REDIS_CMD DEL "TETRA-TO-TRI-KEYSET"
$REDIS_CMD KEYS "TETRA-TO-TRI:*" | xargs $REDIS_CMD SADD "TETRA-TO-TRI-KEYSET"

echo "Loading trigraph to AOR mapping data into Redis instance on $REDIS_SERVER"
$REDIS_CMD KEYS "TRI-TO-AOR:*" | xargs $REDIS_CMD DEL
cat $TRI_TO_AOR_FILE | $REDIS_CMD
$REDIS_CMD DEL "TRI-TO-AOR-KEYSET"
$REDIS_CMD KEYS "TRI-TO-AOR:*" | xargs $REDIS_CMD SADD "TRI-TO-AOR-KEYSET"

echo "Loading countryname to trigraph mapping data, from 2 files, into Redis instance on $REDIS_SERVER"
$REDIS_CMD KEYS "COUNTRYNAME_TO_TRI:*" | xargs $REDIS_CMD DEL
sed -e '/^#/d;/^$/d' $COUNTRYNAME_TO_TRI_MANUAL_FILE | cat $COUNTRYNAME_TO_TRI_GEONAMES_FILE - | $REDIS_CMD

echo "Loading trigraph to province mapping data into Redis instance on $REDIS_SERVER"
$REDIS_CMD KEYS "TRI-TO-PROV:*" | xargs $REDIS_CMD DEL
cat $PROVINCE_FILE | $REDIS_CMD
$REDIS_CMD DEL "TRI-TO-PROV-KEYSET"
$REDIS_CMD KEYS "TRI-TO-PROV:*" | xargs $REDIS_CMD SADD "TRI-TO-PROV-KEYSET"

echo "Loading classification to rank mapping data into Redis instance on $REDIS_SERVER"
$REDIS_CMD KEYS "CLASSIF-TO-RANK:*" | xargs $REDIS_CMD DEL
cat $CLASSIF_FILE | $REDIS_CMD
$REDIS_CMD DEL "CLASSIF-TO-RANK-KEYSET"
$REDIS_CMD KEYS "CLASSIF-TO-RANK:*" | xargs $REDIS_CMD SADD "CLASSIF-TO-RANK-KEYSET"

echo "Loading MIME type to file extension mapping data into Redis instance on $REDIS_SERVER"
$REDIS_CMD KEYS "MIME-TO-FILE-EXT:*" | xargs $REDIS_CMD DEL
cat $MIME_TO_EXT_FILE | $REDIS_CMD

echo "Loading files types which can be converted to PDF into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL PDF-CONVERSION-FILE-TYPES
cat $PDF_CONV_FILE_TYPES_FILE | $REDIS_CMD

echo "Loading files types which can be converted to WORD into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL WORD-CONVERSION-FILE-TYPES
cat $WORD_CONV_FILE_TYPES_FILE | $REDIS_CMD

echo "Loading supported SCI controls into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL SCI-CONTROLS
cat $SCI_CONTROLS_FILE | $REDIS_CMD

echo "Loading supported COUNTRIES_STATES into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL COUNTRIES_STATES
cat $COUNTRIES_STATES_FILE | $REDIS_CMD

echo "Loading supported NIPF into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL NIPF
cat $NIPF_FILE | $REDIS_CMD

echo "Loading supported LEADTYPES into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL LEADTYPES
cat $LEADTYPES_FILE | $REDIS_CMD

echo "Loading supported NATIONALITIES into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL NATIONALITIES
cat $NATIONALITIES_FILE | $REDIS_CMD

echo "Loading supported AOR into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL AORS
cat $AOR_FILE | $REDIS_CMD

echo "Loading supported STATES into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL STATES
cat $STATES_FILE | $REDIS_CMD

echo "Loading supported LANGUAGES into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL LANGUAGES
cat $LANGUAGES_FILE | $REDIS_CMD

echo "Loading supported ETHNIC-GROUPS into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL ETHNIC-GROUPS
cat $ETHNIC_GROUPS_FILE | $REDIS_CMD

echo "Loading supported ENEMY-NETWORKS into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL ENEMY-NETWORKS
cat $ENEMY_NETWORKS_FILE | $REDIS_CMD

echo "Loading supported THREAT-PROGRESSION-PHASES into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL THREAT-PROGRESSION-PHASES
cat $THREAT_PROGRESSION_PHASES_FILE | $REDIS_CMD

echo "Loading supported IICT-TYPES into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL IICT-TYPES
cat $IICT_TYPES_FILE | $REDIS_CMD

echo "Loading supported THREAT-LEVELS into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL THREAT-LEVELS
cat $THREAT_LEVELS_FILE | $REDIS_CMD

echo "Loading supported TOC-STATUSES into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL TOC-STATUSES
cat $TOC_STATUSES_FILE | $REDIS_CMD

echo "Loading supported TOC-TARGETS into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL TOC-TARGETS
cat $TOC_TARGETS_FILE | $REDIS_CMD

echo "Loading supported DTST-TARGETS into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL DTST-TARGETS
cat $DTST_TARGETS_FILE | $REDIS_CMD

echo "Loading supported CITIES into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL CITIES
cat $CITY_FILE | $REDIS_CMD

echo "Loading supported ORCON into Redis instance on $REDIS_SERVER"
$REDIS_CMD DEL ORCON
cat $ORCON_FILE | $REDIS_CMD

exit 0