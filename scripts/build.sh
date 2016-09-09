#!/bin/bash

# invoked inside container

rm -rf ~/rpmbuild
cd ${ODRIVE_ROOT}/cmd/odrive-database
tar cvfz odrive-schema-${ODRIVE_VERSION}.tar.gz schema
cd ${ODRIVE_ROOT}
( cd cmd/odrive && go build )
( cd cmd/odrive && go build -o main )
( cd cmd/odutil && go build )

#build it
${ODRIVE_ROOT}/scripts/prepare-rpm-env.sh
cp ~/rpmbuild/RPMS/x86_64/odrive-${ODRIVE_VERSION}-1.x86_64.rpm $ODRIVE_ROOT

cd $ODRIVE_ROOT

#actually install it
rpm -i odrive-${ODRIVE_VERSION}-1.x86_64.rpm



