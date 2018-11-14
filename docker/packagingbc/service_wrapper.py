#!/usr/bin/python

import os, subprocess
from pwd import getpwnam
import select
import time

"""
This file should be invoked from docker-compose. We take the env passed in
and use it to template out a new /opt/services/object-drive-1.0/env.sh file 
internal to the container. Then we start object-drive
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

    env_script = "/opt/services/object-drive-1.0/env.sh"

    if os.path.exists(env_script):
        os.remove(env_script)

    with open(env_script, "w") as f:
        f.write("#!/bin/bash \n")
        f.write("\n")
        for k, v in env_for_proc.iteritems():
            print 'Adding to env.sh: %s %s' % (k, v)
            if k == "OD_DB_CONN_PARAMS":
                f.write("export {0}=\"{1}\"\n".format(k, v))
                continue
            if k.startswith("OD"):
                f.write("export {0}={1}\n".format(k, v))

        f.write("\n")

    subprocess.check_call(["service", "object-drive-1.0", "start"], env=env_for_proc)

    # Simulate tail -f.
    for line in tail_f(open("/opt/services/object-drive-1.0/log/object-drive.log")):
        print line

    
