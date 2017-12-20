#!/usr/bin/python

import sys, os, subprocess, re, time, signal, logging
from stat import *

def shell_source(script):
    source_file = open(script, "r")
    source_dict = {}
    prefix = "export "
    for line in source_file:
        if line.lower().startswith(prefix):
            line = line[len(prefix):]
            line_parts = line.split("=", 1)
            line_key = line_parts[0]
            line_value = line_parts[1].rstrip("\r\n ")
            line_value = line_value.lstrip("\"")
            line_value = line_value.rstrip("\"")
            line_value = line_value.lstrip("\'")
            line_value = line_value.rstrip("\'")            
            line_value = os.path.expandvars(line_value)
            source_dict[line_key] = line_value
            os.environ.update(source_dict)
    source_file.close()

def get_pid_for_process(service):
    logging.debug('In get_pid_from_process')
    child = subprocess.Popen(['pgrep',service], stdout=subprocess.PIPE, shell=False)
    results = child.communicate()
    if len(results) > 0:
        return results[0]
    else:
        return -1

def test_var_expansion():
    print "test_var_expansion"
    cwd = os.getcwd()
    thepath = os.path.join(cwd, "env-sample.sh")
    shell_source(thepath)
    # OD_CACHE_ROOT
    ret_root = os.getenv("OD_CACHE_ROOT")
    exp_root = "/opt/services/object-drive-1.0/cache"
    if ret_root != exp_root:
        raise ValueError("OD_CACHE_ROOT=%s, expected %s" % (ret_root, exp_root))
    # OD_LOG_LOCATION
    ret_log = os.getenv("OD_LOG_LOCATION")
    exp_log = "/opt/services/object-drive-1.0/log/object-drive.log"
    if ret_log != exp_log:
        raise ValueError("OD_LOG_LOCATION=%s, expected %s" % (ret_log, exp_log))
    return 0

def test_var_quotes():
    print "test_var_quotes"
    cwd = os.getcwd()
    thepath = os.path.join(cwd, "env-sample.sh")
    shell_source(thepath)
    # remove double quotes
    ret_params = os.getenv("OD_DB_CONN_PARAMS")
    exp_params = "parseTime=true&collation=utf8_unicode_ci&readTimeout=30s"
    if ret_params != exp_params:
        raise ValueError("OD_DB_CONN_PARAMS=%s, expected %s" % (ret_params, exp_params))
    # remove single quotes
    ret_ciphers = os.getenv("OD_SERVER_CIPHERS")
    exp_ciphers = "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_RSA_WITH_AES_128_CBC_SHA"
    if ret_ciphers != exp_ciphers:
        raise ValueError("OD_SERVER_CIPHERS=%s, expected %s" % (ret_ciphers, exp_ciphers))
    return 0    

def test_pid_check():
    print "test_pid_check"
    pid = get_pid_for_process("dockerd")
    if pid < 1:
        raise ValueError("No process id found for dockerd")
    return 0

if __name__ == '__main__':
    try:
        test_pid_check()
        test_var_expansion()
        test_var_quotes()
        sys.exit(0)
    except (SystemExit):
        raise
    except:
        # Other errors
        extype, value = sys.exc_info()[:2]
        print >> sys.stderr, "ERROR: %s (%s)" % (extype, value)
        sys.exit(1)