#!/usr/bin/env python

import json
import os
import random
from pprint import pprint

newusers = 5000

# Create a new user that is similar to an existing user, but with a new name and new security attributes
while newusers > 0:
	for subdir, dirs, files in os.walk("./users"):
		for file in files:
			if newusers >= 0:
				f = open(subdir + "/" + file, 'r')
				json_data = f.read()
				data = json.loads(json_data)
				if random.random() > 0.1:
					existing_dn = data['dn']
					first = "usey%d" % newusers
					last = "mcuser%d" % newusers
					new_dn = "CN=%s %s, OU=AAA, O=U.S. Government, C=US" % (first, last)
					data['dn'] = new_dn
					data['whitePageAttributes']['firstName'] = first
					data['whitePageAttributes']['surName'] = last
					data['whitePageAttributes']['icEmail'] = "ic%d@ic.gov" % (newusers)
					data['whitePageAttributes']['niprnetEmail'] = "niprnet%d@nn.gov" % (newusers)
					data['whitePageAttributes']['siprnetEmail'] = "%s.%s@nn.gov" % (first, last)
					data['whitePageAttributes']['telephoneNumber'] = "555-%d" % (newusers)
					newusers = newusers - 1

					with open(subdir + "/" + first + " " + last + ".json", 'w') as outf:
						json.dump(data, outf, ensure_ascii=True, indent=4)	
