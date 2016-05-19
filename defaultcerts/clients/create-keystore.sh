#!/bin/bash

keytool -importkeystore -srckeystore test_0.p12 -srcstoretype PKCS12 \
  -srcstorepass password -keystore javakeystores/keystore_file_name.jks \
  -storepass password
