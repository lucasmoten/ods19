# server Certificate files

These files may be referenced by assorted docker-compose files and go client tests

## server.cert.pem
Effectively the twl-server-generic2, issued by InterCA
```
Issuer: C = us, O = u.s. government, OU = people, OU = dae, OU = chimera, CN = InterCA
Validity
    Not Before: Feb  3 23:00:12 2017 GMT
    Not After : Dec 13 23:00:12 2026 GMT
Subject: C = US, O = U.S. Government, OU = twl-server-generic2, OU = DIA, OU = DAE, CN = twl-server-generic2
```

## server.trust.pem (expired)
```
Issuer: CN = AlienCA1, O = U.S. Government, C = US
Validity
    Not Before: Jun  1 18:09:17 2007 GMT
    Not After : May 29 18:09:17 2017 GMT
Subject: CN = AlienCA1, O = U.S. Government, C = US
```
## trust.pem
```
Issuer: C = US, ST = VA, L = Chantilly, O = DIA, CN = Certificate Manager
Validity
    Not Before: Mar 20 05:00:00 2007 GMT
    Not After : Mar 24 05:00:00 2027 GMT
Subject: C = US, ST = VA, L = Chantilly, O = DIA, CN = Certificate Manager
---
Issuer: C = US, O = U.S. Government, CN = DIAS Root CA
Validity
    Not Before: Dec 20 19:57:36 2012 GMT
    Not After : Dec 18 19:57:36 2022 GMT
Subject: C = US, O = U.S. Government, CN = DIAS Root CA
---
Issuer: C = US, O = U.S. Government, CN = DIAS Root CA
Validity
    Not Before: Dec 21 14:42:12 2012 GMT
    Not After : Dec 18 19:57:36 2022 GMT
Subject: C = US, O = U.S. Government, CN = DIAS SUBCA2
---
Issuer: C = US, ST = VA, L = Chantilly, O = DIA, CN = Certificate Manager
Validity
    Not Before: Mar 20 14:30:54 2007 GMT
    Not After : Mar 20 14:30:54 2027 GMT
Subject: C = US, O = U.S. Government, OU = DoD, OU = DIA, CN = DIA Subordinate Certificate Manager
---
Issuer: CN = Six3Systems, O = US Government, C = US
Validity
    Not Before: Feb  6 23:26:10 2012 GMT
    Not After : Feb  3 23:26:10 2022 GMT
Subject: CN = Six3Systems, O = US Government, C = US
---
Issuer: C = us, O = u.s. government, OU = people, OU = dae, OU = chimera, CN = RootCA
Validity
    Not Before: Jan 27 12:03:21 2017 GMT
    Not After : Jan 25 12:03:21 2027 GMT
Subject: C = us, O = u.s. government, OU = people, OU = dae, OU = chimera, CN = InterCA
---
Issuer: C = us, O = u.s. government, OU = people, OU = dae, OU = chimera, CN = RootCA
Validity
    Not Before: Jan 27 12:01:55 2017 GMT
    Not After : Jan 22 12:01:55 2037 GMT
Subject: C = us, O = u.s. government, OU = people, OU = dae, OU = chimera, CN = RootCA
```