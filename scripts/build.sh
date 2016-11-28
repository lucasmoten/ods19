#!/bin/bash

# invoked inside container

if yum list installed object-drive-1.0 >/dev/null 2>&1; then
  yum remove object-drive-1.0 -y
fi

rm -rf ~/rpmbuild
cd ${ODRIVE_ROOT}/cmd/odrive-database
tar cvfz odrive-schema-${ODRIVE_VERSION}.tar.gz schema
cd ${ODRIVE_ROOT}
( cd cmd/odrive && go build )
( cd cmd/odutil && go build )
( 
  cd cmd/odrive-database
  go get -u github.com/jteeuwen/go-bindata/...
  go-bindata schema migrations ../../defaultcerts/client-mysql/id ../../defaultcerts/client-mysql/trust
)

#build it
${ODRIVE_ROOT}/scripts/prepare-rpm-env.sh
cp ~/rpmbuild/RPMS/x86_64/object-drive-1.0-${ODRIVE_VERSION}-SNAPSHOT.x86_64.rpm $ODRIVE_ROOT

cd $ODRIVE_ROOT

#actually install it
rpm -i object-drive-1.0-${ODRIVE_VERSION}-SNAPSHOT.x86_64.rpm



