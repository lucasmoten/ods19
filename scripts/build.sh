#!/bin/bash

# invoked inside container

export ODRIVE_VERSION_MAJOR_MINOR=$(awk '{printf "%s.%s", $1, $2}' <<< "${ODRIVE_VERSION//[^0-9]/ }")

if yum list installed object-drive-${ODRIVE_VERSION_MAJOR_MINOR} >/dev/null 2>&1; then
  yum remove object-drive-${ODRIVE_VERSION_MAJOR_MINOR} -y
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
cp ~/rpmbuild/RPMS/x86_64/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}-${ODRIVE_VERSION}-${ODRIVE_BUILDNUM}.${ODRIVE_BUILDDATE}.x86_64.rpm ${ODRIVE_ROOT}

cd ${ODRIVE_ROOT}
#chown nobody:nogroup *.rpm
