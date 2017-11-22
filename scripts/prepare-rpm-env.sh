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

ODRIVE_PACKAGE_NAME="object-drive-1.0-${ODRIVE_VERSION}"

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
mkdir -m 750 -p ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0
mkdir -m 750 -p ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/images
mkdir -m 750 -p ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/js
mkdir -m 750 -p ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates

install -m 750 -D ${ODRIVE_BINARY_DIR}/odrive ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/object-drive-1.0
install -m 640 -D ${ODRIVE_BINARY_DIR}/odrive.yml ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/object-drive.yml
install -m 640 -D ${ODRIVE_ROOT}/server/static/css/source_code_pro.css ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/css/source_code_pro.css 
install -m 640 -D ${ODRIVE_ROOT}/server/static/favicon.ico ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/favicon.ico
install -m 640 -D ${ODRIVE_ROOT}/server/static/images/odrive-builddate.png ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/images/odrive-builddate.png
install -m 640 -D ${ODRIVE_ROOT}/server/static/images/odrive-buildnum.png ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/images/odrive-buildnum.png
install -m 640 -D ${ODRIVE_ROOT}/server/static/images/odrive-service.png ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/images/odrive-service.png
install -m 640 -D ${ODRIVE_ROOT}/server/static/images/odrive-version.png ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/images/odrive-version.png
install -m 640 -D ${ODRIVE_ROOT}/server/static/js/getObjectStream.png ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/js/getObjectStream.png
install -m 640 -D ${ODRIVE_ROOT}/server/static/js/etag.png ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/js/etag.png
install -m 640 -D ${ODRIVE_ROOT}/server/static/templates/APISample.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates/APISample.html
install -m 640 -D ${ODRIVE_ROOT}/server/static/templates/ObjectDriveSDK.java ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates/ObjectDriveSDK.java
install -m 640 -D ${ODRIVE_ROOT}/server/static/templates/api.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates/api.html
install -m 640 -D ${ODRIVE_ROOT}/server/static/templates/changelog.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates/changelog.html
install -m 640 -D ${ODRIVE_ROOT}/server/static/templates/environment.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates/environment.html
install -m 640 -D ${ODRIVE_ROOT}/server/static/templates/events.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates/events.html
install -m 640 -D ${ODRIVE_ROOT}/server/static/templates/home.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates/home.html
install -m 640 -D ${ODRIVE_ROOT}/server/static/templates/rest.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates/rest.html

# schema tarball
install -m 640 -D ${ODRIVE_ROOT}/cmd/odrive-database/odrive-schema-${ODRIVE_VERSION}.tar.gz ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/odrive-schema-${ODRIVE_VERSION}.tar.gz

# odrive-database binary
install -m 750 -D ${ODRIVE_DATABASE_DIR}/odrive-database ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/database

# password obfuscation mechanism
install -m 750 -D ${ODRIVE_OBFUSCATE_DIR}/obfuscate ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/obfuscate

# Install service scripts and dependencies
install -m 755 ${ODRIVE_ROOT}/scripts/init.d/object-drive-1.0 ${ODRIVE_PACKAGE_NAME}/etc/init.d/object-drive-1.0
install -m 644 ${ODRIVE_ROOT}/scripts/logrotate.d/object-drive-1.0 ${ODRIVE_PACKAGE_NAME}/etc/logrotate.d/object-drive-1.0
install -m 640 ${ODRIVE_ROOT}/scripts/env.sh ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/env.sh

tar -zcvf ${ODRIVE_PACKAGE_NAME}.tar.gz ${ODRIVE_PACKAGE_NAME}/

cp ${ODRIVE_PACKAGE_NAME}.tar.gz SOURCES/

if [ -f SPECS/object-drive-1.0.spec ]; then
    rm SPECS/object-drive-1.0.spec
fi

cat <<EOF > SPECS/object-drive-1.0.spec
# Be sure buildpolicy set to do nothing
%define        __spec_install_post %{nil}
%define          debug_package %{nil}
%define        __os_install_post %{_dbpath}/brp-compress
%define         _unpackaged_files_terminate_build 0

Summary: Binary distribution of object-drive-server
Name: object-drive-1.0
Version: ${ODRIVE_VERSION}
Release: ${ODRIVE_RELEASE}
License: None
Group: Development/Tools
SOURCE0 : %{name}-%{version}.tar.gz
URL: https://github.com/DecipherNow/object-drive-server

BuildRoot: %{_tmppath}/%{name}-%{version}-%{release}-root


%description
%{summary}


%prep
%setup -q


%build
# Empty section.


%pre
# backup config for restoration
if [ -f /etc/odrive/odrive.yml ]; then
    cp -f /etc/odrive/odrive.yml /tmp
fi
if [ -f /opt/services/object-drive/odrive.yml ]; then
    cp -f /opt/services/object-drive/odrive.yml /tmp
fi
if [ -f /opt/services/object-drive-1.0/object-drive.yml ]; then
    cp -f /opt/services/object-drive-1.0/object-drive.yml /tmp/odrive.yml
fi
# backup environment settings for restoration
if [ -f /etc/odrive/env.sh ]; then
    cp -f /etc/odrive/env.sh /tmp
fi
if [ -f /opt/odrive/env.sh ]; then
    cp -f /opt/odrive/env.sh /tmp
fi
if [ -f /opt/odrive/env.sh.rpmsave ]; then
    cp -f /opt/odrive/env.sh.rpmsave /tmp/env.sh
fi
if [ -f /opt/services/object-drive-1.0/env.sh ]; then
    cp -f /opt/services/object-drive-1.0/env.sh /tmp
fi
if [ -f /opt/services/object-drive-1.0/env.sh.rpmsave ]; then
    cp -f /opt/services/object-drive-1.0/env.sh.rpmsave /tmp/env.sh
fi
# add group and user if not already present
/usr/bin/getent group services || /usr/sbin/groupadd -f -r services
/usr/bin/getent passwd object-drive || /usr/sbin/useradd --no-create-home --no-user-group --gid services object-drive 
# check if existing service running
if pgrep -x "object-drive-1.0" > /dev/null; then
   echo "running" > /tmp/object-drive-1.0.runstate
   service object-drive-1.0 stop
else
   echo "stopped" > /tmp/object-drive-1.0.runstate
fi
exit 0

%post
if [ -f /tmp/odrive.yml ]; then
    echo "moving old odrive.yml into object-drive.yml"
    mv -f /tmp/odrive.yml /opt/services/object-drive-1.0/object-drive.yml
fi
if [ -f /tmp/env.sh ]; then
    echo "moving old env.sh"
    mv -f /tmp/env.sh /opt/services/object-drive-1.0/env.sh
fi
if [ -d /etc/chkconfig.d ]; then
    echo "configuring chkconfig enabling run level 3 and 5"
    chkconfig --add object-drive-1.0
    chkconfig --level 3 object-drive-1.0 on
    chkconfig --level 5 object-drive-1.0 on
fi
if grep -q "running" /tmp/object-drive-1.0.runstate; then
    service object-drive-1.0 start
fi

%postun
if [ "$1" = "1" ]; then
    /usr/sbin/userdel -r object-drive
    rm -rf /var/spool/mail/object-odrive
    rm -rf /opt/services/object-drive-1.0/cache
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
%config(noreplace) /opt/services/object-drive-1.0/object-drive.yml
%config(noreplace) /opt/services/object-drive-1.0/odrive-schema-${ODRIVE_VERSION}.tar.gz
%config(noreplace) /opt/services/object-drive-1.0/env.sh
/opt/services/object-drive-1.0/libs
/opt/services/object-drive-1.0/obfuscate
/opt/services/object-drive-1.0/database
/opt/services/object-drive-1.0/object-drive-1.0
%{_sysconfdir}/init.d/object-drive-1.0
%{_sysconfdir}/logrotate.d/object-drive-1.0

EOF

rpmbuild -ba SPECS/object-drive-1.0.spec
