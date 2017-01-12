#!/bin/sh

# Generate a self signed CA, server key, and client key for use with this image.
openssl genrsa 2048 > ca-key.pem
openssl req -new -x509 -nodes -days 3650 -key ca-key.pem -out ca.pem -config ssl-ca.config
openssl x509 -issuer -subject -dates -noout -in ca.pem

# Use CA to make Server Cert
openssl req -newkey rsa:2048 -days 3650 -nodes -keyout server-key.pem -out server-req.pem -config ssl-server.config
openssl rsa -in server-key.pem -out server-key.pem
openssl x509 -req -in server-req.pem -days 3650 -CA ca.pem -CAkey ca-key.pem -set_serial 01 -out server-cert.pem -extfile ssl-server.config -extensions v3_req
openssl x509 -issuer -subject -dates -noout -in server-cert.pem

# Use CA to make Client Cert
openssl req -newkey rsa:2048 -days 3650 -nodes -keyout client-key.pem -out client-req.pem -config ssl-client.config
openssl rsa -in client-key.pem -out client-key.pem
openssl x509 -req -in client-req.pem -days 3650 -CA ca.pem -CAkey ca-key.pem -set_serial 01 -out client-cert.pem -extfile ssl-client.config -extensions v3_req
openssl x509 -issuer -subject -dates -noout -in client-cert.pem

# Verify certificates
openssl verify -CAfile ca.pem server-cert.pem client-cert.pem

# Copy CA for mysql to defaultcerts
mkdir -p $GOPATH/src/decipher.com/object-drive-server/defaultcerts/client-mysql/trust
cp ca.pem $GOPATH/src/decipher.com/object-drive-server/defaultcerts/client-mysql/trust/ca.pem

# Copy Client Cert + Key to defaultcerts
mkdir -p $GOPATH/src/decipher.com/object-drive-server/defaultcerts/client-mysql/id
cp client-cert.pem $GOPATH/src/decipher.com/object-drive-server/defaultcerts/client-mysql/id/client-cert.pem
cp client-key.pem $GOPATH/src/decipher.com/object-drive-server/defaultcerts/client-mysql/id/client-key.pem


