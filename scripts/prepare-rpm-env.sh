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

ODRIVE_PACKAGE_NAME="object-drive-1.0-${ODRIVE_VERSION}"

ODRIVE_DATABASE_DIR="$ODRIVE_ROOT/cmd/odrive-database"

mkdir -p ~/rpmbuild/{RPMS,SRPMS,BUILD,SOURCES,SPECS,tmp}

if [ ! -f ~/.rpmmacros ]; then
  cat <<EOF >~/.rpmmacros
%_topdir   %(echo $HOME)/rpmbuild
%_tmppath  %{_topdir}/tmp
%define _unpackaged_files_terminate_build 0
EOF
fi


cd ~/rpmbuild

mkdir ${ODRIVE_PACKAGE_NAME}
mkdir -p ${ODRIVE_PACKAGE_NAME}/etc/init.d
mkdir -p ${ODRIVE_PACKAGE_NAME}/etc/logrotate.d
mkdir -p ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0
mkdir -p ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/js
mkdir -p ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/js/ajax/libs/bootstrap/3.3.6/css
mkdir -p ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/js/ajax/libs/jquery/3.0.0-beta1
mkdir -p ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/js/ajax/libs/reqwest/2.0.5
mkdir -p ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/js/ajax/libs/then-request/2.1.1
mkdir -p ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates

install -m 755 -D ${ODRIVE_BINARY_DIR}/odrive ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/object-drive-1.0
install -m 644 -D ${ODRIVE_BINARY_DIR}/odrive.yml ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/object-drive.yml
install -m 644 -D ${ODRIVE_ROOT}/server/static/js/listObjects.js ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/js/listObjects.js
install -m 644 -D ${ODRIVE_ROOT}/server/static/js/ajax/libs/bootstrap/3.3.6/css/bootstrap.min.css ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/js/ajax/libs/bootstrap/3.3.6/css/bootstrap.min.css
install -m 644 -D ${ODRIVE_ROOT}/server/static/js/ajax/libs/jquery/3.0.0-beta1/jquery.js ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/js/ajax/libs/jquery/3.0.0-beta1/jquery.js
install -m 644 -D ${ODRIVE_ROOT}/server/static/js/ajax/libs/reqwest/2.0.5/reqwest.js ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/js/ajax/libs/reqwest/2.0.5/reqwest.js
install -m 644 -D ${ODRIVE_ROOT}/server/static/js/ajax/libs/then-request/2.1.1/request.js ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/js/ajax/libs/then-request/2.1.1/request.js
install -m 644 -D ${ODRIVE_ROOT}/server/static/templates/_function.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates/_function.html
install -m 644 -D ${ODRIVE_ROOT}/server/static/templates/home.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates/home.html
install -m 644 -D ${ODRIVE_ROOT}/server/static/templates/listObjects.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates/listObjects.html
install -m 644 -D ${ODRIVE_ROOT}/server/static/templates/listObjects.js ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates/listObjects.js
install -m 644 -D ${ODRIVE_ROOT}/server/static/templates/ObjectDriveSDK.java ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates/ObjectDriveSDK.java
install -m 644 -D ${ODRIVE_ROOT}/server/static/templates/root.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates/root.html
install -m 644 -D ${ODRIVE_ROOT}/server/static/templates/APISample.html ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/templates/APISample.html
install -m 644 -D ${ODRIVE_ROOT}/server/static/favicon.ico ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/libs/server/static/favicon.ico

# schema tarball
install -m 644 -D ${ODRIVE_ROOT}/cmd/odrive-database/odrive-schema-${ODRIVE_VERSION}.tar.gz ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/odrive-schema-${ODRIVE_VERSION}.tar.gz

# odrive-database binary
install -m 755 -D ${ODRIVE_DATABASE_DIR}/odrive-database ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/database

# Install service scripts and dependencies
install -m 755 ${ODRIVE_ROOT}/scripts/init.d/object-drive-1.0 ${ODRIVE_PACKAGE_NAME}/etc/init.d/object-drive-1.0
install -m 755 ${ODRIVE_ROOT}/scripts/logrotate.d/object-drive-1.0 ${ODRIVE_PACKAGE_NAME}/etc/logrotate.d/object-drive-1.0
install -m 755 ${ODRIVE_ROOT}/scripts/env.sh ${ODRIVE_PACKAGE_NAME}/opt/services/object-drive-1.0/env.sh

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
Release: SNAPSHOT
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
if [ -f /etc/odrive/odrive.yml ]; then
    cp -f /etc/odrive/odrive.yml /tmp
fi
if [ -f /opt/services/object-drive/odrive.yml ]; then
    cp -f /opt/services/object-drive/odrive.yml /tmp
fi

if [ `grep -c '^services:' /etc/group` = 1 ] ; then
  echo services group exists
else
  groupadd -f services
  exit 0
fi

if [ `grep -c '^object-drive:' /etc/passwd` = 1 ] ; then
  echo object-drive user exists
else
  useradd --no-create-home --no-user-group --gid services object-drive
  exit 0
fi

%post
if [ -f /tmp/odrive.yml ]; then
    echo "copying old odrive.yml into object-drive.yml"
    cp -f /tmp/odrive.yml /opt/services/object-drive-1.0/object-drive.yml
fi
if [ -f /opt/odrive/env.sh.rpmsave ]; then
    echo "copying old env.sh.rpmsave"
    cp -f /opt/odrive/env.sh.rpmsave /opt/services/object-drive/env.sh
fi



%postun
userdel -r object-drive
rm -rf /var/spool/mail/object-odrive
rm -rf /home/object-drive
rm -rf /opt/services/object-drive-1.0/cache


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
/opt/services/object-drive-1.0/database
/opt/services/object-drive-1.0/object-drive-1.0
%{_sysconfdir}/init.d/object-drive-1.0
%{_sysconfdir}/logrotate.d/object-drive-1.0

EOF

rpmbuild -ba SPECS/object-drive-1.0.spec
