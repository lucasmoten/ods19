Changelog
================

Release vNEXT
-------------
* NEW: Command `serviceTest` renamed to `test`
* NEW: RPM updated to use /opt/services/object-drive-1.0 installation path, object-drive-1.0 for servicename, object-drive for username

Release v1.0.1.8
-------------
* NEW: CORS support in the server
* DOC: API Documentation now reflects chagnes where OwnedBy field is now stored and returned in serialized resource format
* NEW: Masterkey refactored down into cache layer
* FIX: Recover under race condition for user creation
* NEW: Kafka is discoverable from its own ZK cluster, not just default ZK
* NEW: RPM updated to use /opt/services/object-drive installation path
* FIX: Port announced for service in ZK is based upon actual server port selected
* FIX: Prevent non-owners from moving objects

Release v1.0.1.7
----------------
* NEW: AAC is discoverable from its own ZK cluster, not just default ZK.
* NEW: Additional debug logging around database code for updating ACMs on objects.
* FIX: Can specify everyone (group/-Everyone/-Everyone) for read permission when creating object 
* NEW: Configuration for environment variable OD_AWS_ENDPOINT is now read from OD_AWS_S3_ENDPOINT
* NEW: Support for Peer2Peer retrieval of content streams when running multiple instances of ODrive

Release v1.0.1.6
----------------

* FIX: odrive-database utility now allows cascade override from config file
* FIX: odrive-database migration script 2-down fixed
* DOC: API Documentation updated to reflect changes to Create/Update object, responses, improved URI examples, and search filtering.
* NEW: Search and List operations support AND filters in addition to default OR
* NEW: Update Object request supports passing updated permissions in new 1.1 format
* NEW: Create Object request supports providing permissions in new 1.1 format

Release v1.0.1.5
----------------

* NEW: disableS3 with an empty S3 Bucket variable.  it works with load balancing due to p2p caching.
* FIX: large stalls as load balanced clients wait for S3 ciphertext is taken care of with p2p caching. it created instability when viewing large videos.
* NEW: odrive-database utility supports migrations.
* NEW: List of objects shared to the user (/shares) will exclude those whose parent is also shared to them.
* NEW: List of objects shared to others (/shared) will exclude those whose parent is also shared to others.
* NEW: List of objects shared to everyone (/sharedpublic) will exclude those whose parent is also shared to everyone.
* FIX: Breadcrumbs will be limited to the first parents accessible to a user. No
  longer returning the complete list with redacted folder names
* NEW: Autoscaling report gets messages triggered by a CloudWatch alarm writes to SQS so we shut down and tell Autoscale 

Release v1.0.1.4
----------------

* FIX: Shared with Me now excludes objects shared to Everyone
* NEW: Allow many more config values to be specified in odrive.yml
* DOC: Endpoints to add and remove object shares are deprecated, as is ability to provide permissions when creating object.
* NEW: All events now wrapped with global event model (GEM), with odrive-specific payload field
* NEW: An array of breadcrumb objects is returned with an object's properties
* FIX: Default Kafka configuration resolved
* FIX: Update Object Properties will now carry over stream based fields to new revision
* FIX: ACM part processing will now skip empty values instead of failing to store update.
* FIX: Update Object with ACM Share now ensures owner retains read access.
* FIX: Existing objects have full CRUDS permissions assigned to owners.
* FIX: List of objects /shared to others will exclude those that are private to the user. 
* NEW: CloudWatch metrics that enable the setting of alarms by admins (an auto-scaling prerequisite)

Release v1.0.1
--------------

* !216 - Enhancement: Connection to Zookeeper recovery improvements when timed out
* !218 - Enhancement: Capture full ACM share information for individual permission grants
* Old schema patch files deleted. Database will need to be dropped due to ACM share model
* Documentation updated with detailed permissions struct
* Updated zipfile endpoint internals
* odrive binary will run as user `odrive` when installed with yum package
* Major release number bump at customer request

Release v0.1.0
--------------

* !192 – Refactor: Remove broken STANDALONE flag
* !197 – FIX: Return 404 instead of 500 when retrieving an object properties and given ID is invalid.
* !200 – NEW: Allow caller to specify returned content-disposition format when requesting streams and zipped content
* !201 – NEW: Response to create object will now populate callerPermisison
* !203 - NEW: Publish Events to Kafka
* !205 – Refactor: Docstrings on Index event fields
* !208 – Enhancement: RPMs generated will now create odrive user when installed
* !209 – Enhancement: All API responses returning object now populate callerPermissions
* !210 – Enhancement: US Persons Data and FOIA Exemption state fields now track Yes/No/Unknown instead of True/False
