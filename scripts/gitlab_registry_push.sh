#!/bin/bash


# You must be logged into the private registry with LDAP credentials.
# 
#    $ docker login docker.363-283.io
#

# retag images
docker tag deciphernow/odrive docker.363-283.io/cte/object-drive-server 
docker tag deciphernow/metadatadb docker.363-283.io/cte/object-drive-metadatadb 
docker tag deciphernow/gatekeeper:latest docker.363-283.io/cte/object-drive-metadatadb:gatekeeper
docker tag deciphernow/zk:latest docker.363-283.io/cte/object-drive-metadatadb:zk
docker tag deciphernow/aac:latest docker.363-283.io/cte/object-drive-metadatadb:aac

# push images
docker push docker.363-283.io/cte/object-drive-server 
docker push docker.363-283.io/cte/object-drive-metadatadb:latest 
docker push docker.363-283.io/cte/object-drive-metadatadb:aac 
docker push docker.363-283.io/cte/object-drive-metadatadb:zk
docker push docker.363-283.io/cte/object-drive-metadatadb:gatekeeper 

echo "Finished"
