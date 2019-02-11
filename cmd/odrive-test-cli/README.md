# odrive-test-cli
A simple CLI utility to upload files to odrive.

## Usage
```bash
$ ./odrive-test-cli -help

NAME:
   odrive-test-cli - odrive CRUD operations from the command-line for testing

USAGE:
   odrive-test-cli [global options] command [command options] [arguments...]
   
VERSION:
   0.0.0
   
COMMANDS:
     example-conf	print an example configuration file
     test-connection	establish connection to odrive and check for errors
     upload		upload file to odrive
     test-fill		upload a sample of random files and directories to the server

GLOBAL OPTIONS:
   --conf value		Path to yaml config
   --json		print all responses as formatted JSON
   --yaml		print all responses as formatted YAML
   --tester value	tester credentials to use for connection
   --queue value	queue size in threaded upload (default: 1000)
   --threads value	number of threads to use in upload (default: 64)
   --help, -h		show help
   --version, -v	print the version
```

### Connecting
By default, a connection will be established using included certificates, 
values, and paths.  To override these values, a YAML configuration file 
can be specified using the `--conf` flag.  A complete example configuration
file would have all the fields shown below.

```YAML
cert:  /path/to/test.cert.pem
trust: /path/to/client.trust.pem
key: /path/to/test.key.pem
skipverify: true
remote: https://url.to.odrive
```

Partial config files can be specified as well, and any non-specified 
values will be taken from the included defaults. E.g., the below uses
all the default values, except for remote and key.

```YAML
key: /my/different/test.key.pem
remote: https://custom.odrive.path
```

### Uploading files
Uploading files will, by default only print a very concise output of 
operations being performed.
```bash
$ ./odrive-test-cli upload *_data.txt

uploading test1_data.txt...done
uploading test2_data.txt...done
uploading test3_data.txt...done

```

More verbose output, suitable for use in other applications, can be 
printed using the `--json` or `--yaml` flag.

```bash
$ ./odrive-test-cli upload test1_data.txt --json

uploading test1_data.txt...done
{
    "id": "11e734ce47b1bbe4bac30242ac120002",
    "createdDate": "2017-05-09T15:43:41.941609Z",
    "createdBy": "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
    "modifiedDate": "2017-05-09T15:43:41.941609Z",
    "modifiedBy": "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
    "deletedDate": "0001-01-01T00:00:00Z",
    "deletedBy": "",
    "changeCount": 0,
    "changeToken": "26329487b16dc05ce8a2d7767f56f0c0",
    "ownedBy": "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
    "typeId": "11e734c381f9d95abac30242ac120002",
    "typeName": "File",
    "name": "test1_data.txt",
    "description": "",
    ...
}

```

```bash
  $ ./odrive-test-cli upload test1_data.txt --yaml

  uploading test1_data.txt...done
  id: 11e741736f8db2b69ac90242ac120004
  createddate: 2017-05-25T17:56:10.668854Z
  createdby: cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us
  modifieddate: 2017-05-25T17:56:10.668854Z
  modifiedby: cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us
  deleteddate: 0001-01-01T00:00:00Z
  deletedby: ""
  changecount: 0
  changetoken: e5e3f938bd2d11920bdb1af1e7191bb9
  ownedby: user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us
  typeid: 11e74170a0c596559ac90242ac120004
  typename: File
  name: test1_data.txt
  description: ""
...
```

### Uploading as a different tester
All commands can be run as different testers to enable simulating runs with different levels of 
permission.  This can be done with the `--tester` flag as shown below.  Valid values to supply are 1-10.
Supplying no value will default to tester10.

```bash
./odrive-test-cli upload test1_data.txt --tester 5
```

### Uploading random selection of files

Odrive-test-cli can generate random paths and files to the server for use in 
testing.  Files are cleaned locally after each upload to keep the local space clean.
Simply supply a numerical argument, default is 100, and a random suite of files 
will be created an uploaded.  The concurrency of this upload can be tweaked with
the `--queue` and `--threads` flags.


  
```bash
./odrive-test-cli test-fill 10
uploading testFile_403606150...done
uploading testFile_885760557...done
uploading testFile_821608360...done
uploading testFile_571344615...done
uploading testFile_189237530...done
uploading testFile_381040049...done
uploading testFile_369997660...done
uploading testFile_123440395...done
uploading testFile_610485486...done
uploading testFile_580091253...done
```
