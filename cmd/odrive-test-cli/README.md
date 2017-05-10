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
     test-connection	establish connection to odrive and check for erros
     upload		upload file to odrive

GLOBAL OPTIONS:
   --conf value		Path to yaml config
   --json		print all responses as formatted JSON
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
printed using the `--json` flag.

```bash
$ ./odrive-test-cli upload test1_data.txt

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