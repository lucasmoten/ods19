#!/bin/bash

host=10.2.11.46
port=10443
cert="server.cert.pem"
trust="server.trust.pem"
key="server.key.pem"

openssl s_client -connect $host:$port -cert $cert -CAfile $trust -key $key


