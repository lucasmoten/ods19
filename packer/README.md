# Packer builds

These builds should be able to create AMIs that are launchable and configurable.

## object-drive-1.0

The "standalone" AMI for our service. If you have `packer` installed, you can
invoke the build with **build.sh**. This simply passes some data to the build 
definition in **object-drive.json**.

Every AMI is different. To build an AMI for your environment, please edit the following:

* **build.sh** needs to pass in the required AWS credentials.
* **build.sh** needs to pass in the filename of the RPM (in same dir)
* An SSH user and password, or an SSH key must be configured for packer to do
  its work.
* An **/assets** folder must be created local to this build, in the same directory
  as **build.sh**
* **/assets** must contain the configurations and certs required for your environment


Here is a list of files that the build expects in **assets**. Note that they have
names that the actual build defined in **object-drive.json** expects.

```
## Configs
env.sh

## Certs
aac.client.trust.pem
aac.client.cert.pem
aac.client.key.pem
rds-combined-ca-bundle.pem
server.cert.pem
server.DIASRootCA.pem
server.DIASSUBCA2.pem
server.key.pem
```

When building from the **object-drive-server** repo, the **copy-assets.sh** script
can be used to populate an **assets** folder with the standard test certificates.

## Running sudo during builds

You _must_ specify `"communicator": "ssh"` and `"ssh_pty": true` in your `builders` config stanza
to successfully run commands as **sudo** during build. [Please see this GitHub issue.](https://github.com/mitchellh/packer/issues/2420#issuecomment-258047050)

# Known Failure Modes

* If a new version of object-drive is shipped **and** a schema update needs to be applied,
  the migration must be applied manually. A launched AMI will not do it for you.

