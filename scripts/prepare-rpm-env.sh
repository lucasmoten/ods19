#!/bin/bash

# Write files and directories for rpmbuild

if [ -z ${ODRIVE_BINARY_DIR+x}]; then
    echo "ODRIVE_BINARY must be set"
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
EOF
fi


cd ~/rpmbuild

mkdir ${ODRIVE_PACKAGE_NAME}
mkdir -p ${ODRIVE_PACKAGE_NAME}/usr/bin
mkdir -p ${ODRIVE_PACKAGE_NAME}/etc/odrive
install -m 755 ${ODRIVE_BINARY_DIR}/odrive ${ODRIVE_PACKAGE_NAME}/usr/bin
install -m 644 ${ODRIVE_BINARY_DIR}/odrive.yml ${ODRIVE_PACKAGE_NAME}/etc/odrive/

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

Summary: Binary distribution of object-drive-server
Name: odrive
Version: 1.0
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
%{_bindir}/*

%changelog
* Mon Jun 24 2016  Coleman McFarland <coleman.mcfarland@deciphernow.com> 1.0-1
- RPM packaging completed.

EOF


rpmbuild -ba SPECS/odrive.spec

