# pki Certificate files

These files may be referenced by nginx/proxier, either for listening, or as client proxy

## client.crt
Effectively the twl-server-generic2, issued by InterCA
```
Issuer: C = us, O = u.s. government, OU = people, OU = dae, OU = chimera, CN = InterCA
Validity
    Not Before: Feb  3 23:00:12 2017 GMT
    Not After : Dec 13 23:00:12 2026 GMT
Subject: C = US, O = U.S. Government, OU = twl-server-generic2, OU = DIA, OU = DAE, CN = twl-server-generic2
```

## server.public
Also the twl-server-generic2 issued by InterCA
```
Issuer: C = us, O = u.s. government, OU = people, OU = dae, OU = chimera, CN = InterCA
Validity
    Not Before: Feb  3 23:00:12 2017 GMT
    Not After : Dec 13 23:00:12 2026 GMT
Subject: C = US, O = U.S. Government, OU = twl-server-generic2, OU = DIA, OU = DAE, CN = twl-server-generic2
```

## trusted.crt
```
Issuer: C = US, O = U.S. Government, CN = DIAS Root CA
Validity
    Not Before: Dec 20 19:57:36 2012 GMT
    Not After : Dec 18 19:57:36 2022 GMT
Subject: C = US, O = U.S. Government, CN = DIAS Root CA
```
