Execute this when your config is ready.  You WILL need to first set AWS credentials in your environment before launching this

AWS_ACCESS_KEY_ID
AWS_REGION
AWS_S3_BUCKET
AWS_S3_ENDPOINT
AWS_SECRET_ACCESS_KEY

Do NOT unpack this anywhere in the operating system's global temp directory, as it's a different kind of filesystem there,
and may not work correctly with docker (on OSX).  Untar this into a normal directory to minimize confusion.

Use this to bring up odrive
```
. ./odrive_launch up
```

You can use this to just set the environment variables, and have it complain until enough are set
```
. ./odrive_launch env
```

Use this to just stop it for a while, and start it later
```
. ./odrive_launch stop
...
. ./odrive_launch start
```

After the server comes up, you can connect with a browser from which you have installed some of the certificates.
Choose a test certificate in the ./certs directory, and use the password "password".
