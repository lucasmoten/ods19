#!/bin/sh

rm ca-key.pem
rm ca.pem
rm client-cert.pem
rm client-key.pem
rm client-req.pem
rm server-cert.pem
rm server-key.pem
rm server-req.pem

# Remove client certificate, key, and certificate authority certificate from defaultcerts
rm -f $GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/client-mysql/trust/ca.pem
rm -f $GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/client-mysql/id/client-cert.pem
rm -f $GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/client-mysql/id/client-key.pem

