#!/usr/bin/env python
import os
import sys
import argparse
import shutil

import subprocess

try:
    import driver
    from driver.builder import Builder
except ImportError:
    this_script = os.path.abspath(os.path.split(__file__)[1])
    sys.path.insert(0, os.sep.join(this_script.split(os.sep)[:-3]))
    import driver
    from driver.builder import Builder

def main(args):

    print 'This script is not yet implemented.'
    sys.exit(1)
    
    # get AAC Build Home, the pwd
    pwd = os.getwd()
    # Get AAC Home, the repo's dir
    # If AAC Home exists, 
    
    pass
   
if __name__ == '__main__':

    parser = argparse.ArgumentParser()
    parser.add_argument('--tag', nargs='?', const='latest', default='latest', type=str)
    opts = parser.parse_args()
    
    main(opts)
    
    
