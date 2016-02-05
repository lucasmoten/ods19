#!/bin/bash

# Create an s_client connection to running test instance of the audit service
# You must be on the VPN for this to work.

host=10.2.11.46
port=10443
base_cert_path="../defaultcerts/server"
cert="${base_cert_path}/server.cert.pem"
trust="${base_cert_path}/server.trust.pem"
key="${base_cert_path}/server.key.pem"

openssl s_client -connect $host:$port -cert $cert -CAfile $trust -key $key
