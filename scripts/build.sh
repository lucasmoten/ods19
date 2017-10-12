#!/bin/bash

# invoked inside container

if yum list installed object-drive-1.0 >/dev/null 2>&1; then
  yum remove object-drive-1.0 -y
fi

rm -rf ~/rpmbuild
cd ${ODRIVE_ROOT}/cmd/odrive-database
tar cvfz odrive-schema-${ODRIVE_VERSION}.tar.gz schema
cd ${ODRIVE_ROOT}
mkdir -p $GOPATH/bin
PATH=$PATH:$GOPATH/bin
echo "Building odrive server"
( cd cmd/odrive && go build -ldflags "-X main.Build=${CIRCLE_BUILD_NUM} -X main.Commit=${CIRCLE_SHA1} -X main.Version=${TAG_VERSION}" )
( cd cmd/odutil && go build )
echo "Building odrive-database"
( 
  cd cmd/odrive-database
  go get -u github.com/jteeuwen/go-bindata/...
  go-bindata schema migrations ../../defaultcerts/client-mysql/id ../../defaultcerts/client-mysql/trust
  go build -ldflags "-X main.Build=${CIRCLE_BUILD_NUM} -X main.Commit=${CIRCLE_SHA1} -X main.Version=${TAG_VERSION}"
)
( cd cmd/obfuscate && go build )

# satisfy dependency in prepare-rpm-env
export ODRIVE_BUILDNUM=${CIRCLE_BUILD_NUM}
export ODRIVE_BUILDDATE=$(date +%Y%m%d)

echo "invoking prepare-rpm-env.sh"
${ODRIVE_ROOT}/scripts/prepare-rpm-env.sh
cp ~/rpmbuild/RPMS/x86_64/object-drive-1.0-${ODRIVE_VERSION}-${ODRIVE_BUILDNUM}.${ODRIVE_BUILDDATE}.x86_64.rpm $ODRIVE_ROOT

cd $ODRIVE_ROOT

echo "installing object-drive RPM"
rpm -i object-drive-1.0-${ODRIVE_VERSION}-${ODRIVE_BUILDNUM}.${ODRIVE_BUILDDATE}.x86_64.rpm
