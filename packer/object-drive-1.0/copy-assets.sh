#!/bin/bash

# Helper script to gather assets from other areas of the project to the
# local assets folder.

root=${GOPATH}/src/decipher.com/object-drive-server

# Make sure assets path exists for this build/trust
mkdir -p ${root}/packer/object-drive-1.0/assets/trust

cp ${root}/defaultcerts/client-aac/trust/client.trust.pem assets/aac.client.trust.pem
cp ${root}/defaultcerts/client-aac/id/client.cert.pem assets/aac.client.cert.pem
cp ${root}/defaultcerts/client-aac/id/client.key.pem assets/aac.client.key.pem
cp ${root}/defaultcerts/aws/rds-combined-ca-bundle.pem assets/rds-combined-ca-bundle.pem
cp ${root}/defaultcerts/server/server.key.pem assets/server.key.pem
cp ${root}/defaultcerts/server/server.cert.pem assets/server.cert.pem
cp ${root}/defaultcerts/server-web/trust/DIASRootCA assets/server.DIASRootCA.pem
cp ${root}/defaultcerts/server-web/trust/DIASSUBCA2 assets/server.DIASSUBCA2.pem

echo "assets copied"
