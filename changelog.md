FORMAT: 1A

# Object Drive Changelog

## Release vNEXT
-------------
* FIX: Service process now assigned group and user when sudoing down from root.
* ENH: Orphaned files that cannot be removed due to permissions are renamed to permit service termination.

## Release v1.0.1.13
-------------
* ENH: Calculated full path and unique names for objects. Slashes are now restricted characters from updates.
* ENH: Bulk Delete objects: DELETE /objects
* ENH: Bulk Move objects: POST /objects/move
* FIX: Restore ability to update properties on objects
* ENH: Bulk Change owner objects: POST /objects/owner/{resourceString}
* ENH: Return properties in search results consistent with other list calls
* ENH: Check database schema version on startup. Must match expected. Wait for migration before terminating.
* NEW: Emit events with audit payload for all handlers to support ICS 500-27
* NEW: Build Changelog into HTML and link from API Documentation

## Release v1.0.1.12
-----------------
* ENH: Determination of content type from file extension on upload expanded to larger list
* ENH: Autoscale shutdown from lifecycle messages now handle 10 messages at a time, configured via OD_AWS_SQS_BATCHSIZE
* NEW: Empty Trash: DELETE /trashed
* NEW: Bulk Get Properties: GET /objects/properties 
* FIX: OD_ZK_ANNOUNCE no longer must be 4 parts, and default changed to /services/object-drive/1.0
* FIX: Service init script only changes ownership of certificates if they are found under OD_BASEPATH
* DOC: API Documentation updated with sectiosn for empty trash and retrieving bulk objects

## Release v1.0.1.11
-----------------
* ENH: Added more logging for AAC Client connection when receiving announce data
* FIX: RPM adds user and group if not present. Now deletes only on uninstall, not upgrades.

## Release v1.0.1.10
-----------------
* FIX: RPM updated to create services group, and change ownership to object-drive:services
* ENH: Performance improvements to database list/search operations, and additional indexing on key columns
* FIX: Object-Drive Service Init script no longer assigns group to process to prevent failure.

## Release v1.0.1.9
----------------
* ENH: Command `serviceTest` renamed to `test`
* FIX: RPM updated to use /opt/services/object-drive-1.0 installation path, object-drive-1.0 for servicename, object-drive for username
* REF: Abstract AAC authorization calls from server handlers to new interface
* NEW: Implemented API operation to change owner
* NEW: Implemented API to list root objects owned by a group
* DOC: API Documentation updated with Change Owner and List Objects at Root For Group
* ENH: ACL Impersonation Whitelist read from different location in object-drive.yml 

## Release v1.0.1.8
----------------
* NEW: CORS support in the server
* DOC: API Documentation now reflects changes where OwnedBy field is now stored and returned in serialized resource format
* NEW: Masterkey refactored down into cache layer
* FIX: Recover under race condition for user creation
* NEW: Kafka is discoverable from its own ZK cluster, not just default ZK
* NEW: RPM updated to use /opt/services/object-drive installation path
* FIX: Port announced for service in ZK is based upon actual server port selected
* FIX: Prevent non-owners from moving objects

## Release v1.0.1.7
----------------
* NEW: AAC is discoverable from its own ZK cluster, not just default ZK.
* NEW: Additional debug logging around database code for updating ACMs on objects.
* FIX: Can specify everyone (group/-Everyone/-Everyone) for read permission when creating object 
* NEW: Configuration for environment variable OD_AWS_ENDPOINT is now read from OD_AWS_S3_ENDPOINT
* NEW: Support for Peer2Peer retrieval of content streams when running multiple instances of ODrive

## Release v1.0.1.6
----------------

* FIX: odrive-database utility now allows cascade override from config file
* FIX: odrive-database migration script 2-down fixed
* DOC: API Documentation updated to reflect changes to Create/Update object, responses, improved URI examples, and search filtering.
* NEW: Search and List operations support AND filters in addition to default OR
* NEW: Update Object request supports passing updated permissions in new 1.1 format
* NEW: Create Object request supports providing permissions in new 1.1 format

## Release v1.0.1.5
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

## Release v1.0.1.4
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

## Release v1.0.1
--------------

* ENH: Connection to Zookeeper recovery improvements when timed out
* ENH: Full ACM share information captured for individual permission grants
* FIX: Old schema patch files deleted. Database will need to be dropped due to ACM share model
* DOC: API Documentation updated with detailed permissions structure
* REF: Updated zipfile endpoint internals
* FIX: odrive binary will run as user `odrive` when installed with yum package
* FIX: Major Release number bump at customer request

## Release v0.1.0
--------------

* REF: Remove broken STANDALONE flag
* FIX: Return 404 instead of 500 when retrieving an object properties and given ID is invalid.
* NEW: Allow caller to specify returned content-disposition format when requesting streams and zipped content
* NEW: Response to create object will now populate callerPermisison
* NEW: Publish Events to Kafka
* REF: Index structure used by Finder now includes docstrings
* ENH: RPMs generated will now create odrive user when installed
* ENH: All API responses returning object now populate callerPermissions
* ENH: US Persons Data and FOIA Exemption state fields now track Yes/No/Unknown instead of True/False
