#!/usr/bin/env python
# This script has been updated to work with python 2.7 and python 3.4
# This script is run within the docker container after installing python 3.4

import os, subprocess
from pwd import getpwnam
import select
import time

"""
This file should be invoked from docker-compose. We take the env passed in
and use it to template out a new env.sh file internal to the container. 
Then we start object-drive
"""
def tail_f(file):
    interval = 1.0
  
    while True:
        where = file.tell()
        line = file.readline()
        if not line:
            time.sleep(interval)
            file.seek(where)
        else:
            yield line

if __name__ == '__main__':

    env_for_proc = dict()
    for k in os.environ.keys():
        env_for_proc[k] = os.getenv(k, "")

    version = "--MajorMinorVersion--"

    env_script = "/opt/services/object-drive-" + version + "/env.sh"

    if os.path.exists(env_script):
        os.remove(env_script)

    with open(env_script, "w") as f:
        f.write("#!/bin/bash \n")
        f.write("\n")
        for k, v in env_for_proc.iteritems():
            print('Adding to env.sh: %s %s' % (k, v))
            if k == "OD_DB_CONN_PARAMS":
                f.write("export {0}=\"{1}\"\n".format(k, v))
                continue
            if k.startswith("OD"):
                f.write("export {0}={1}\n".format(k, v))

        f.write("\n")

    # centos7 support
    systemdscript = "/etc/systemd/object-drive-" + version + ".service"
    if os.path.exists(systemdscript):
        os.remove(systemdscript)
    with open(systemdscript, "w") as f2:
        f2.write("[Unit]\n")
        f2.write("Description=Object Drive " + version + "\n")
        f2.write("After=network.target\n")
        f2.write("\n")
        f2.write("[Service]\n")
        f2.write("Type=simple\n")
        f2.write("User=object-drive-" + version + "\n")
        f2.write("Group=services\n")
        f2.write("ExecStart=/etc/init.d/object-drive-" + version + " start\n")
        f2.write("ExecStop=/etc/init.d/object-drive-" + version + " stop\n")
        f2.write("ExecReload=/etc/init.d/object-drive-" + version + " restart\n")
        f2.write("Restart=on-abort\n")
        f2.write("\n")
        f2.write("[Install]\n")
        f2.write("WantedBy=multi-user.target\n")

    subprocess.check_call(["systemctl", "enable", "object-drive-" + version + ".service"])
    subprocess.check_call(["systemctl", "start", "object-drive-" + version + ".service"], env=env_for_proc)

    # Simulate tail -f.
    for line in tail_f(open("/opt/services/object-drive-" + version + "/log/object-drive.log")):
        print(line)

    
