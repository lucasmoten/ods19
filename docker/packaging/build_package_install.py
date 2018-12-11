#!/usr/bin/python
import os, subprocess, datetime

# This script is run during docker build.
os.environ["CGO_ENABLED"] = "0"
os.environ["GOOS"] = "linux"
os.environ["GOARCH"] = "amd64"

odrive_root = "/go/src/bitbucket.di2e.net/dime/object-drive-server"
database_root = os.path.join(odrive_root, "cmd", "odrive-database")
binary_root = os.path.join(odrive_root, "cmd", "odrive")
obfuscate_root = os.path.join(odrive_root, "cmd", "obfuscate")
os.chdir(database_root)
subprocess.check_call(["tar", "cvfz", "odrive-schema-V1.tar.gz", "schema"])
subprocess.check_call(["go", "build"])

os.chdir(obfuscate_root)
subprocess.check_call(["go", "build"])
os.chdir(binary_root)
subprocess.check_call(["go", "build"])

# Set up env for prepare_rpm_env.sh
os.environ["ODRIVE_BINARY_DIR"] = binary_root
os.environ["ODRIVE_VERSION"] = "V1"
os.environ["ODRIVE_BUILDNUM"] = "SNAPSHOT"
os.environ["ODRIVE_ROOT"]= odrive_root

os.chdir(odrive_root)
subprocess.check_call(["scripts/prepare-rpm-env.sh"])
now = datetime.datetime.now()
rpm = "/root/rpmbuild/RPMS/x86_64/object-drive-1.0-V1-SNAPSHOT." + now.strftime("%Y%m%d") + ".x86_64.rpm"


subprocess.check_call(["yum", "install", "-y", rpm])


print 'Done installing package'

