#!/bin/bash


# You must be logged into the private registry with LDAP credentials.
# 
#    $ docker login docker.363-283.io
#

# retag images
docker tag deciphernow/odrive docker.363-283.io/cte/object-drive-server 
docker tag deciphernow/metadatadb docker.363-283.io/coleman.mcfarland/object-drive-metadatadb 

# push images
docker push docker.363-283.io/cte/object-drive-server 
docker push docker.363-283.io/coleman.mcfarland/object-drive-metadatadb 

