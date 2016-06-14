#!/bin/bash

# Write files and directories for rpmbuild

if [ -z ${ODRIVE_BINARY_DIR+x}]; then
    echo "ODRIVE_BINARY_DIR must be set"
    exit 1
fi

if [ -z ${ODRIVE_VERSION+x}]; then
    echo "ODRIVE_VERSION must be set"
    exit 1
fi

ODRIVE_PACKAGE_NAME="odrive-${ODRIVE_VERSION}"

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
mkdir -p ${ODRIVE_PACKAGE_NAME}/usr/bin
mkdir -p ${ODRIVE_PACKAGE_NAME}/etc/odrive
mkdir -p ${ODRIVE_PACKAGE_NAME}/etc/odrive/libs/server/static/js
mkdir -p ${ODRIVE_PACKAGE_NAME}/etc/odrive/libs/server/static/templates
install -m 755 -D ${ODRIVE_BINARY_DIR}/odrive ${ODRIVE_PACKAGE_NAME}/usr/bin
install -m 644 -D ${ODRIVE_BINARY_DIR}/odrive.yml ${ODRIVE_PACKAGE_NAME}/etc/odrive/
install -m 644 -D ${ODRIVE_BINARY_DIR}/libs/server/static/js/listObjects.js ${ODRIVE_PACKAGE_NAME}/etc/odrive/libs/server/static/js/listObjects.js
install -m 644 -D ${ODRIVE_BINARY_DIR}/libs/server/static/templates/_function.html ${ODRIVE_PACKAGE_NAME}/etc/odrive/libs/server/static/templates/_function.html
install -m 644 -D ${ODRIVE_BINARY_DIR}/libs/server/static/templates/home.html ${ODRIVE_PACKAGE_NAME}/etc/odrive/libs/server/static/templates/home.html
install -m 644 -D ${ODRIVE_BINARY_DIR}/libs/server/static/templates/listObjects.html ${ODRIVE_PACKAGE_NAME}/etc/odrive/libs/server/static/templates/listObjects.html
install -m 644 -D ${ODRIVE_BINARY_DIR}/libs/server/static/templates/listObjects.js ${ODRIVE_PACKAGE_NAME}/etc/odrive/libs/server/static/templates/listObjects.js
install -m 644 -D ${ODRIVE_BINARY_DIR}/libs/server/static/templates/ObjectDriveSDK.java ${ODRIVE_PACKAGE_NAME}/etc/odrive/libs/server/static/templates/ObjectDriveSDK.java
install -m 644 -D ${ODRIVE_BINARY_DIR}/libs/server/static/templates/root.html ${ODRIVE_PACKAGE_NAME}/etc/odrive/libs/server/static/templates/root.html
install -m 644 -D ${ODRIVE_BINARY_DIR}/libs/server/static/templates/TestShare.html ${ODRIVE_PACKAGE_NAME}/etc/odrive/libs/server/static/templates/TestShare.html
install -m 644 -D ${ODRIVE_BINARY_DIR}/libs/server/static/templates/TestUpdate.html ${ODRIVE_PACKAGE_NAME}/etc/odrive/libs/server/static/templates/TestUpdate.html
install -m 644 -D ${ODRIVE_BINARY_DIR}/libs/server/static/favicon.ico ${ODRIVE_PACKAGE_NAME}/etc/odrive/libs/server/static/favicon.ico

tar -zcvf ${ODRIVE_PACKAGE_NAME}.tar.gz ${ODRIVE_PACKAGE_NAME}/

cp ${ODRIVE_PACKAGE_NAME}.tar.gz SOURCES/

if [ -f SPECS/odrive.spec ]; then
    rm SPECS/odrive.spec
fi

cat <<EOF > SPECS/odrive.spec
# Be sure buildpolicy set to do nothing
%define        __spec_install_post %{nil}
%define          debug_package %{nil}
%define        __os_install_post %{_dbpath}/brp-compress
%define         _unpackaged_files_terminate_build 0

Summary: Binary distribution of object-drive-server
Name: odrive
Version: ${ODRIVE_VERSION}
Release: 1
License: None
Group: Development/Tools
SOURCE0 : %{name}-%{version}.tar.gz
URL: https://gitlab.363-283.io/cte/object-drive-server

BuildRoot: %{_tmppath}/%{name}-%{version}-%{release}-root

%description
%{summary}

%prep
%setup -q

%build
# Empty section.

%install
rm -rf %{buildroot}
mkdir -p  %{buildroot}

# in builddir
cp -a * %{buildroot}


%clean
rm -rf %{buildroot}


%files
%defattr(-,root,root,-)
%config(noreplace) %{_sysconfdir}/%{name}/%{name}.yml
%{_sysconfdir}/%{name}/libs
%{_bindir}/*

%changelog
* Tue Jun 14 2016  Coleman McFarland <coleman.mcfarland@deciphernow.com> 1.0-1
- Static files bundled. 	
* Mon May 24 2016  Coleman McFarland <coleman.mcfarland@deciphernow.com> 1.0-1
- RPM packaging completed.

EOF


rpmbuild -ba SPECS/odrive.spec

