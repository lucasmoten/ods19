#!/bin/bash

#set -euo pipefail

# This test script is intended to run an environment with multiple nodes of 
# object drive, populate files, and then attempt to continuously read back 
# until halted via CTRL+C.  Key parameters that should be adjusted in the
# docker-compose.yml include the docker images, the OD_CACHE_FILELIMIT, and
# whether or not OD_PEER_ENABLED is true or false. 
#
# When using image: docker-dime.di2e.net/dime/object-drive-server:1.0.20b4
# expect the following warnings and errors
#    * WARN  unable to rename - no suchh file or directory - caused by race condition between operation routine and background
#    * WARN  error draining cache - no such file or directory - caused by race condition between operation route and background
#    * ERROR there is an uploaded file that we cannot stat - no such file or directory - caused by race condition between operation routine and backgound
# if OD_PEER_ENABLED=true, or unset
# transaction finish	{"session": "0fa9ce0f", "status": 404, "message": "Resource not found p2p from address 172.29.0.9:44874 using Go-http-client/1.1 unhandled operation GET /ciphertext/S3_DEFAULT/5b28a912ccf281652c3632c8a0bab2ac371a4d11c353d66f73ee", "error": "", "file": "/go/src/bitbucket.di2e.net/dime/object-drive-server/server/AppServer.go", "line": 765}
#    caused by mismatch between expected length of rName introduced in 1.0.20
#
# Rebuilding locally and testing with the following image should mitigate
#     image: deciphernow/odrive-bc:latest


# Dynamically gets the name of this script to use in uploads as a file
thescriptname=$(basename "$0")
filestoadd=50

ODRIVE_VERSION=$(grep -m 1 '## Release' ../../../changelog.md|awk '{print $3}')
ODRIVE_VERSION_MAJOR_MINOR=$(awk '{printf "%s.%s", $1, $2}' <<< "${ODRIVE_VERSION//[^0-9]/ }")

export CA='../../../defaultcerts/clients/client.trust.pem'
export CERT='../../../defaultcerts/clients/test_0.cert.pem'
export KEY='../../../defaultcerts/clients/test_0.key.pem'

export UDN='cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us'
export SDN="cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"

# ------------------------------------------------------------------------------

echo "Checking whether docker is already running"
# PWD##*/ gets the current directory name without the leading path, which becomes the default
# suffix for docker compose images that we are comparing against
rcount=$(docker ps -a | grep ${PWD##*/} | wc | awk '{print $1}')
if [ $rcount -eq 0 ]
then
  echo "Starting Docker Compose"
  # Start docker compose
  docker-compose up -d
  # Wait 15 seconds for it to come online
  echo "Waiting 15 seconds to increase likelihood of availability"
  sleep 15  
else
  echo "Docker Compose already running"
fi

# ------------------------------------------------------------------------------

echo "Adding ${filestoadd} files"
curlsilent="--silent"
for ((i = 1; i <= $filestoadd; i++)); do
  curl ${curlsilent} --form 'ObjectMetadata={"typeName":"File","name":"'${thescriptname}'/'${i}'","namePathDelimiter":"/","acm":{"classif":"U","version":"2.1.0"}}' --form 'blob=@"'${thescriptname}'"' --insecure --cacert ${CA} --cert ${CERT} --key ${KEY} https://localhost:8080/services/object-drive/${ODRIVE_VERSION_MAJOR_MINOR}/objects >> /dev/null
done

# ------------------------------------------------------------------------------

echo "Retreiving page of results"
# Get a page of objects for file ids
pageofresults=$(curl --insecure --cacert ${CA} --cert ${CERT} --key ${KEY} https://localhost:8080/services/object-drive/${ODRIVE_VERSION_MAJOR_MINOR}/files/${thescriptname}/?pageSize=${filestoadd})

# ------------------------------------------------------------------------------
# This relies on the docker compose having OD_CACHE_FILELIMIT=1 which will effectively
# nearly empty the cache of files during cache walk, forcing a re-retrieval of files
# from the bucket when requested
echo "Keep requesting files and report every 10 seconds"
COUNTER=0
REPORTTIME=$(date +%s)
while true; do
  for ((i = 0; i < $filestoadd; i++)); do
    let COUNTER=COUNTER+1 
    objectid=$(echo $pageofresults | jq -r '.objects['${i}'].id')
    curl ${curlsilent} --insecure --cacert ${CA} --cert ${CERT} --key ${KEY} https://localhost:8080/services/object-drive/${ODRIVE_VERSION_MAJOR_MINOR}/objects/${objectid}/stream >> /dev/null

    # Report approximately every 10 seconds

    NEWREPORTTIME=$(date +%s)
    if [ ! $REPORTTIME -eq $NEWREPORTTIME ]
    then
      d=$(date +%s|rev|cut -c 1)
      if [ $d -eq 0 ] 
      then
        echo "$COUNTER file requests made"
        REPORTTIME=$NEWREPORTTIME
      fi
    fi

  done
  
done

