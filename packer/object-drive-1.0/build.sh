#!/bin/bash

set -e

packer build \
    -var "aws_access_key=${OD_AWS_ACCESS_KEY_ID}" \
    -var "aws_secret_key=${OD_AWS_SECRET_ACCESS_KEY}" \
    -var "rpm=object-drive-1.0.1.719.x86_64.rpm" \
    object-drive.json

