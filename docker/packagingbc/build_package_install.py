#!/usr/bin/env python3
# This script has been updated to work with python 2.7 and python 3.4
# This script is run within the docker build after installing python 3.4

import os, subprocess, datetime
#-------------------------------------------------------------------------------

def get_version():
    changelogverline = subprocess.check_output(['grep', '-m', '1', '## Release', 'changelog.md']).split()[2]
    changelogverline = changelogverline.replace(b'v',b'')
    goversion = subprocess.check_output(['go','version']).split()[2]
    if b'b' in goversion:
        result = changelogverline + b'b' + goversion.split(b'b')[1]
    else:
        result = changelogverline
    return result.decode('utf-8')

#-------------------------------------------------------------------------------
if __name__ == "__main__":
    # This script is run during docker build.
    os.environ["CGO_ENABLED"] = "1"
    os.environ["GOOS"] = "linux"
    os.environ["GOARCH"] = "amd64"

    odrive_root = "/go/src/bitbucket.di2e.net/dime/object-drive-server"
    os.chdir(odrive_root)
    version = get_version()
    major_minor = version.split('.')[0] + "." + version.split('.')[1]
    database_root = os.path.join(odrive_root, "cmd", "odrive-database")
    binary_root = os.path.join(odrive_root, "cmd", "odrive")
    obfuscate_root = os.path.join(odrive_root, "cmd", "obfuscate")
    os.chdir(database_root)
    subprocess.check_call(["tar", "cvfz", "odrive-schema-" + version + ".tar.gz", "schema"])
    subprocess.check_call(["go", "build"])
    #os.environ["LD_LIBRARY_PATH"] = "/opt/glibc-2.14/lib"
    os.chdir(obfuscate_root)
    subprocess.check_call(["go", "build"])
    os.chdir(binary_root)
    subprocess.check_call(["go", "build"])

    # Set up env for prepare_rpm_env.sh
    os.environ["ODRIVE_BINARY_DIR"] = binary_root
    os.environ["ODRIVE_VERSION"] = version
    os.environ["ODRIVE_BUILDNUM"] = "SNAPSHOT"
    os.environ["ODRIVE_ROOT"]= odrive_root

    os.chdir(odrive_root)
    subprocess.check_call(["scripts/prepare-rpm-env.sh"])
    now = datetime.datetime.now()
    rpm = "/root/rpmbuild/RPMS/x86_64/object-drive-" + major_minor + "-" + os.environ["ODRIVE_VERSION"] + "-" + os.environ["ODRIVE_BUILDNUM"] + "." + now.strftime("%Y%m%d") + ".x86_64.rpm"    

    # This is expected to fail since libc.so.6 version 2.14 isn't satisfied from the standpoint of RPM dependency check
    subprocess.check_call(["yum", "install", "-y", rpm])
    #subprocess.check_call(["yum", "--skip-broken", "install", "-y", rpm])

    print('Done installing package')

