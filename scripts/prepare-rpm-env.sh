#!/bin/bash

# Write files and directories for rpmbuild. This script expects cmd/odrive
# and cmd/odrive-database binaries to be already built.

if [ -z ${ODRIVE_BINARY_DIR+x} ]; then
    echo "ODRIVE_BINARY_DIR must be set"
    exit 1
fi

if [ -z ${ODRIVE_ROOT+x} ]; then
    echo "ODRIVE_ROOT must be set"
    exit 1
fi

if [ -z ${ODRIVE_VERSION+x} ]; then
    echo "ODRIVE_VERSION must be set"
    exit 1
fi

if [ -z ${ODRIVE_BUILDNUM+x} ]; then
    echo "ODRIVE_BUILDNUM must be set"
    exit 1
fi

# Determine major and minor
ODRIVE_VERSION_MAJOR_MINOR=$(awk '{printf "%s.%s", $1, $2}' <<< "${ODRIVE_VERSION//[^0-9]/ }")

ODRIVE_PACKAGE_NAME="object-drive-${ODRIVE_VERSION_MAJOR_MINOR}-${ODRIVE_VERSION}"

ODRIVE_DATABASE_DIR="$ODRIVE_ROOT/cmd/odrive-database"

ODRIVE_OBFUSCATE_DIR="$ODRIVE_ROOT/cmd/obfuscate"

ODRIVE_BUILDDATE=$(date +%Y%m%d)
ODRIVE_RELEASE="${ODRIVE_BUILDNUM}.${ODRIVE_BUILDDATE}"

mkdir -p ~/rpmbuild/{RPMS,SRPMS,BUILD,SOURCES,SPECS,tmp}

if [ ! -f ~/.rpmmacros ]; then
  cat <<EOF >~/.rpmmacros
%_topdir   %(echo $HOME)/rpmbuild
%_tmppath  %{_topdir}/tmp
%define _unpackaged_files_terminate_build 0
EOF
fi


cd ~/rpmbuild

mkdir -m 755 ${ODRIVE_PACKAGE_NAME}
mkdir -m 755 -p ${ODRIVE_PACKAGE_NAME}/etc/init.d
mkdir -m 755 -p ${ODRIVE_PACKAGE_NAME}/etc/logrotate.d
mkdir -m 750 -p ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}
mkdir -m 750 -p ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/images
mkdir -m 750 -p ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/js
mkdir -m 750 -p ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/templates

install -m 750 -D ${ODRIVE_BINARY_DIR}/odrive ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}
install -m 640 -D ${ODRIVE_BINARY_DIR}/odrive.yml ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/object-drive.yml
install -m 640 -D ${ODRIVE_ROOT}/server/static/client.go ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/client.go 
install -m 640 -D ${ODRIVE_ROOT}/server/static/css/source_code_pro.css ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/css/source_code_pro.css 
install -m 640 -D ${ODRIVE_ROOT}/server/static/favicon.ico ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/favicon.ico
install -m 640 -D ${ODRIVE_ROOT}/server/static/object-drive-sag.pdf ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/object-drive-sag.pdf
install -m 640 -D ${ODRIVE_ROOT}/server/static/images/odrive-service.png ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/images/odrive-service.png
install -m 640 -D ${ODRIVE_ROOT}/server/static/js/getObjectStream.png ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/js/getObjectStream.png
install -m 640 -D ${ODRIVE_ROOT}/server/static/js/etag.png ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/js/etag.png
install -m 640 -D ${ODRIVE_ROOT}/server/static/templates/APISample.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/templates/APISample.html
install -m 640 -D ${ODRIVE_ROOT}/server/static/templates/ObjectDriveSDK.java ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/templates/ObjectDriveSDK.java
install -m 640 -D ${ODRIVE_ROOT}/server/static/templates/boringcrypto.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/templates/boringcrypto.html
install -m 640 -D ${ODRIVE_ROOT}/server/static/templates/changelog.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/templates/changelog.html
install -m 640 -D ${ODRIVE_ROOT}/server/static/templates/environment.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/templates/environment.html
install -m 640 -D ${ODRIVE_ROOT}/server/static/templates/events.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/templates/events.html
install -m 640 -D ${ODRIVE_ROOT}/server/static/templates/home.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/templates/home.html
install -m 640 -D ${ODRIVE_ROOT}/server/static/templates/rest.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs/server/static/templates/rest.html

# schema tarball
install -m 640 -D ${ODRIVE_ROOT}/cmd/odrive-database/odrive-schema-${ODRIVE_VERSION}.tar.gz ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/odrive-schema-${ODRIVE_VERSION}.tar.gz

# odrive-database binary
install -m 750 -D ${ODRIVE_DATABASE_DIR}/odrive-database ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/database

# password obfuscation mechanism
install -m 750 -D ${ODRIVE_OBFUSCATE_DIR}/obfuscate ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/obfuscate

# Install service scripts and dependencies
install -m 755 ${ODRIVE_ROOT}/scripts/init.d/object-drive ${ODRIVE_PACKAGE_NAME}/etc/init.d/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}
install -m 644 ${ODRIVE_ROOT}/scripts/logrotate.d/object-drive ${ODRIVE_PACKAGE_NAME}/etc/logrotate.d/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}
install -m 640 ${ODRIVE_ROOT}/scripts/env.sh ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/env.sh

# Version replacements
if [[ "$OSTYPE" == "linux-gnu" ]]; then
    sed -i s/--MajorMinorVersion--/${ODRIVE_VERSION_MAJOR_MINOR}/ ${ODRIVE_PACKAGE_NAME}/etc/init.d/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}
    sed -i s/--MajorMinorVersion--/${ODRIVE_VERSION_MAJOR_MINOR}/ ${ODRIVE_PACKAGE_NAME}/etc/logrotate.d/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}
    sed -i s/--MajorMinorVersion--/${ODRIVE_VERSION_MAJOR_MINOR}/ ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/env.sh
elif [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' -e s/--MajorMinorVersion--/${ODRIVE_VERSION_MAJOR_MINOR}/ ${ODRIVE_PACKAGE_NAME}/etc/init.d/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}
    sed -i '' -e s/--MajorMinorVersion--/${ODRIVE_VERSION_MAJOR_MINOR}/ ${ODRIVE_PACKAGE_NAME}/etc/logrotate.d/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}
    sed -i '' -e s/--MajorMinorVersion--/${ODRIVE_VERSION_MAJOR_MINOR}/ ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/env.sh
else
    echo "Platform unknown. Unable to perform required replacements"
    exit 1
fi

tar -zcvf ${ODRIVE_PACKAGE_NAME}.tar.gz ${ODRIVE_PACKAGE_NAME}/

cp ${ODRIVE_PACKAGE_NAME}.tar.gz SOURCES/

if [ -f SPECS/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}.spec ]; then
    rm SPECS/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}.spec
fi

cat <<EOF > SPECS/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}.spec
# Be sure buildpolicy set to do nothing
%define        __spec_install_post %{nil}
%define          debug_package %{nil}
%define        __os_install_post %{_dbpath}/brp-compress
%define         _unpackaged_files_terminate_build 0

Summary: Binary distribution of object-drive-server
Name: object-drive-${ODRIVE_VERSION_MAJOR_MINOR}
Version: ${ODRIVE_VERSION}
Release: ${ODRIVE_RELEASE}
License: None
Group: Development/Tools
SOURCE0 : %{name}-%{version}.tar.gz
URL: https://bitbucket.di2e.net/dime/object-drive-server

BuildRoot: %{_tmppath}/%{name}-%{version}-%{release}-root


%description
%{summary}


%prep
%setup -q


%build
# Empty section.


%pre
# backup config for restoration
if [ -f /opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/object-drive.yml ]; then
    cp -f /opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/object-drive.yml /tmp/odrive.yml
fi
# backup environment settings for restoration
if [ -f /opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/env.sh ]; then
    cp -f /opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/env.sh /tmp
fi
if [ -f /opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/env.sh.rpmsave ]; then
    cp -f /opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/env.sh.rpmsave /tmp/env.sh
fi
# add group and user if not already present
/usr/bin/getent group services || /usr/sbin/groupadd -f -r services
/usr/bin/getent passwd object-drive || /usr/sbin/useradd --no-create-home --no-user-group --gid services object-drive 
# check if existing service running
if pgrep -x "object-drive-${ODRIVE_VERSION_MAJOR_MINOR}" > /dev/null; then
   echo "running" > /tmp/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}.runstate
   service object-drive-${ODRIVE_VERSION_MAJOR_MINOR} stop
else
   echo "stopped" > /tmp/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}.runstate
fi
exit 0

%post
if [ -f /tmp/odrive.yml ]; then
    echo "moving old odrive.yml into object-drive.yml"
    mv -f /tmp/odrive.yml /opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/object-drive.yml
fi
if [ -f /tmp/env.sh ]; then
    echo "moving old env.sh"
    mv -f /tmp/env.sh /opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/env.sh
fi
if [ -d /etc/chkconfig.d ]; then
    echo "configuring chkconfig enabling run level 3 and 5"
    chkconfig --add object-drive-${ODRIVE_VERSION_MAJOR_MINOR}
    chkconfig --level 3 object-drive-${ODRIVE_VERSION_MAJOR_MINOR} on
    chkconfig --level 5 object-drive-${ODRIVE_VERSION_MAJOR_MINOR} on
fi
if grep -q "running" /tmp/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}.runstate; then
    service object-drive-${ODRIVE_VERSION_MAJOR_MINOR} start
fi

%postun
if [ "$1" = "1" ]; then
    /usr/sbin/userdel -r object-drive
    rm -rf /var/spool/mail/object-odrive
    rm -rf /opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/cache
fi

%install
rm -rf %{buildroot}
mkdir -p  %{buildroot}

# in builddir
cp -a * %{buildroot}


%clean
rm -rf %{buildroot}


%files
%defattr(-,root,root,-)
%config(noreplace) /opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/object-drive.yml
%config(noreplace) /opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/odrive-schema-${ODRIVE_VERSION}.tar.gz
%config(noreplace) /opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/env.sh
/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/libs
/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/obfuscate
/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/database
/opt/services/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}
%{_sysconfdir}/init.d/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}
%{_sysconfdir}/logrotate.d/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}

EOF

rpmbuild -ba SPECS/object-drive-${ODRIVE_VERSION_MAJOR_MINOR}.spec
