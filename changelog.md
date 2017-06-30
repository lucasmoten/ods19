FORMAT: 1A

# Object Drive Changelog

## Release vNew
* FIX: orphaned files need deletion
* ENH: log when we need to drain files up to S3 after an odrive restart

## Release v1.0.2 (June 30, 2017)
--------------------
* ENH: Instrumentation added for database, aac, and overall http calls - made visible in /stats for now
* ENH: If database schema does not match expected, service will startup in readonly mode, and switch to writeable once migration is complete.
* ENH: Add deadlock configuration parameters to global configurations.
* ENH: Database migrations for 2017 have been consolidated into a single upgrade. Schema is now 20170630
* CFG: Docker-Compose readTimeout set for database connections to 30s. Recommend values under the timeout set for the edge.
* FIX: Return useful error message when resource string not provided in correct format.
* FIX: Startup performance issue relating to scenarios with large caches
* FIX: Corrected count of groups for user.

## Release v1.0.1.26 (June 8, 2017)
--------------------
* FIX: Resolve deadlock on create/update where newly recognized ACM being inserted by multiple concurrent transactions
* CFG: Optional setting for deadlock retries can be configured via OD_DEADLOCK_RETRYCOUNTER, default is 5
* CFG: Optional setting for deadlock retry delay can be configured via OD_DEADLOCK_RETRYDELAYMS, default is 333.
* FIX: API call to change owner referencing a user that is not yet cached will no longer fail.

## Release v1.0.1.25 (June 5, 2017)
--------------------
* ENH: Clean up error message when cannot connect to AAC
* ENH: allow impersonating requests in the client library with new config field "Impersonation"
* REF: Performance improvements for #409 are always enabled and no longer toggle-able. OD_OPTION_409 no longer needs to be set.
* ENH: Allow recursive application of updates to object sharing permissions while retaining other elements of ACM.
* ENH: Add testing features to odrive-test-cli: upload random data suites, specify tester credentials, override partial configs.
* FIX: Normalization of Permissions and ACM having permissions with grantees not declared in ACM share will now retain read access. (Seen when creating objects)
* FIX: Updating object with ACM and no permissions will remove existing permissions with grantees not present in revised ACM
* FIX: Updating object without ACM but with permissions will merge provided permissions to existing ACM.
* FIX: Flatten resulting ACM after Normalizing Permissions and ACM to ensure f_* keys can be joined appropriately.
* ENH: Improve insert/update performance by only checking to associate ACM to users when a new ACM is created.

## Release v1.0.1.24 (May 10, 2017)
--------------------
* ENH: Add option for CLI tools `odrive` and `odrive-database` to print DB schema version.
* FIX: Corrected spelling of `namePathDelimiter` in API documentation for create object.
* FIX: Normalize resource string and grantee parts to lowercase. Force disp_nm in ACM to lowercase matching project key.
* FIX: Filters based upon objects user or group they are a member of owns is now fixed for search, trash, and expunge operations.
* NEW: Add CLI for client library to allow test uploads.
* FIX: Bugfix for 20170331 migration script
* DB: The database schema is now 20170508. A migration should be performed.
* NEW: Add `applyRecursively` field to API endpoint for changing object ownership.
* NEW: Add MoveObject method to the client library. 

## Release v1.0.1.23 (May 4, 2017)
--------------------
* FIX: Allow Content-Transfer-Encoding to be specified as binary, 8bit, or 7bit.
* FIX: Allow Content-Type to support charset values of utf-8 or iso-8859-1.
* NEW: Operation /groups returns information about groups for which caller is a member of that owns objects.

## Release v1.0.1.22 (May 1, 2017)
--------------------
* DOC: API Documentation now denotes length, minvalue, maxlength, maxvalue.
* FIX: Permit contentType to be specified during update without a content stream.
* CFG: Database tool compatibility with MySQL 5.7 (requires parameter show_compatibility_56 = 1)
* FIX: Add support for creating objects owned by group with pathing at time of creation.
* FIX: Security fix for client certificate checks to AAC and peer nodes in an instance.
* CFG: Certificate checks for AAC and PEER CN set in OD_AAC_CN and OD_PEER_CN in env.sh
* DOC: API Documentation correction to create object size sample denoting contentSize as a number.
* FIX: Build process will install graphviz to satisfy plantuml need for dot for diagram generation.
* DOC: API Documentation denotes that content streams should not have encoding or character sets.

## Release v1.0.1.21 (April 21, 2017)
--------------------
* CFG: Docker container for metadatadb innodb_buffer_pool_size being set to 128MB.
* FIX: Docker container for aac upgraded to 1.1.4
* FIX: Migration Script for 20161230 is now more resilient to multiple runs.
* ENH: Wait up to 10 minutes for kafka availability on startup before failing.
* FIX: Saving grantee during permission creation is now being normalized.
* FIX: Database AACFlatten function now applies lowercase.
* DB: The database schema version is now 20170421. A migration should be performed.
* CFG: Set export OD_OPTION_409=true in env.sh to enable performance improvements.

## Release v1.0.1.20 (April 19, 2017)
--------------------
* FIX: Migration for 20170331 now forces creation of aacflatten function if not present.

## Release v1.0.1.19 (April 18, 2017)
--------------------
* FIX: Spelling correction to content type assignment for files having extension xlsx.
* DOC: API Documentation now denotes Get Bulk Properties is via POST method.
* DOC: API Documentation now indicates proper default (false) for sortAscending.
* ENH: Performance Improvements for List/Search operations.
* ENH: Database Migration tool will now periodically output status for long running migrations.
* ENH: Database Migration tool will check if database parameters are setup when using binary logging.
* ENH: Search/List filters have additional experimental filter conditions begins, ends, notbegins, notends, notcontains, notequals.
* DB: The database schema version is now 20170331. A migration should be performed.
* CFG: Set export OD_OPTION_409=true in env.sh to enable performance improvements.

## Release v1.0.1.18 (March 31, 2017)
--------------------
* FIX: Check that cache files exist before attempting to remove them.
* FIX: Path Delimiter validation for Update Object to permit slashes.
* ENH: Allow owner to be specified on object create.
* FIX: Service init script will now check lock state before starting.
* FIX: Return cause of error to caller for failure to create object.
* DOC: API Documentation now denotes dates in RFC3339 format.
* FIX: Centralized how our IP address is determined.
* FIX: Init script now checks that paths are configured as absolute.
* ENH: Init script now uses logging with log levels.
* ENH: Server startup will now block forever until main ZK cluster is reachable.
* REF: Internal refactor for retrieving object revisions.
* FIX: Normalized checks on Content-Type expecting application/json to permit charset.

## Release v1.0.1.17 (March 2, 2017)
--------------------
* FIX: Path Delimiter for internal storage is now using record separator in place of forward slash.
* ENH: Create Object operation may specify namePathDelimiter to override default.
* ENH: Latest git tag is embedded in --version flag.

## Release v1.0.1.16 (February 28, 2017)
--------------------
* FIX: Bugfix to listing shared objects and trash for users with apostrophe in DN.
* ENH: Logging now renders timestamp in RFC3339 format intsead of seconds since unix epoch.
* ENH: RPM installation will now set to start service on run levels 3 and 5 via chkconfig.

## Release v1.0.1.15 (February 22, 2017)
--------------------
* ENH: Build number and git commit sha1 now exposed with the --version flag.
* FIX: Uncached large files no longer truncated at 16MB during download.

## Release v1.0.1.14 (February 9, 2017)
--------------------
* FIX: Service process now assigned group and user when sudoing down from root.
* ENH: Orphaned files that cannot be removed due to permissions are renamed to permit service termination.
* ENH: Service init script for restart handles discrepant pidfile.
* FIX: Fixed minor bug in how zip files are processed if puller can't be initialized.
* ENH: Cached files that cannot be removed due to faulty permissions are truncated if allowed to free up space.
* FIX: Cache purging of files when space consumed is above high watermark no longer considers age.
* FIX: Improve durability of connection to AAC to reduce unnecessary rpc client shutdown.
* FIX: Close connection to ZK when polling for AAC connection every 30 sec.

## Release v1.0.1.13 (January 30, 2017)
--------------------
* ENH: Calculated full path and unique names for objects. Slashes are now restricted characters from updates.
* ENH: Bulk Delete objects: DELETE /objects
* ENH: Bulk Move objects: POST /objects/move
* FIX: Restore ability to update properties on objects
* ENH: Bulk Change owner objects: POST /objects/owner/{resourceString}
* ENH: Return properties in search results consistent with other list calls
* ENH: Check database schema version on startup. Must match expected. Wait for migration before terminating.
* NEW: Emit events with audit payload for all handlers to support ICS 500-27
* NEW: Build Changelog into HTML and link from API Documentation

## Release v1.0.1.12 (December 23, 2016)
--------------------
* ENH: Determination of content type from file extension on upload expanded to larger list
* ENH: Autoscale shutdown from life cycle messages now handle 10 messages at a time, configured via OD_AWS_SQS_BATCHSIZE
* NEW: Empty Trash: DELETE /trashed
* NEW: Bulk Get Properties: GET /objects/properties
* FIX: OD_ZK_ANNOUNCE no longer must be 4 parts, and default changed to /services/object-drive/1.0
* FIX: Service init script only changes ownership of certificates if they are found under OD_BASEPATH
* DOC: API Documentation updated with sections for empty trash and retrieving bulk objects

## Release v1.0.1.11 (December 9, 2016)
-----------------
* ENH: Added more logging for AAC Client connection when receiving announce data
* FIX: RPM adds user and group if not present. Now deletes only on uninstall, not upgrades.

## Release v1.0.1.10 (December 8, 2016)
--------------------
* FIX: RPM updated to create services group, and change ownership to object-drive:services
* ENH: Performance improvements to database list/search operations, and additional indexing on key columns
* FIX: Object-Drive Service Init script no longer assigns group to process to prevent failure.

## Release v1.0.1.9 (December 2, 2016)
-------------------
* ENH: Command `serviceTest` renamed to `test`
* FIX: RPM updated to use /opt/services/object-drive-1.0 installation path, object-drive-1.0 for servicename, object-drive for username
* REF: Abstract AAC authorization calls from server handlers to new interface
* NEW: Implemented API operation to change owner
* NEW: Implemented API to list root objects owned by a group
* DOC: API Documentation updated with Change Owner and List Objects at Root For Group
* ENH: ACL Impersonation Whitelist read from different location in object-drive.yml

## Release v1.0.1.8 (November 17, 2016)
-------------------
* NEW: CORS support in the server
* DOC: API Documentation now reflects changes where OwnedBy field is now stored and returned in serialized resource format
* NEW: Masterkey refactored down into cache layer
* FIX: Recover under race condition for user creation
* NEW: Kafka is discoverable from its own ZK cluster, not just default ZK
* NEW: RPM updated to use /opt/services/object-drive installation path
* FIX: Port announced for service in ZK is based upon actual server port selected
* FIX: Prevent non-owners from moving objects

## Release v1.0.1.7 (November 4, 2016)
----------------
* NEW: AAC is discoverable from its own ZK cluster, not just default ZK.
* NEW: Additional debug logging around database code for updating ACMs on objects.
* FIX: Can specify everyone (group/-Everyone/-Everyone) for read permission when creating object
* NEW: Configuration for environment variable OD_AWS_ENDPOINT is now read from OD_AWS_S3_ENDPOINT
* NEW: Support for Peer2Peer retrieval of content streams when running multiple nodes of object drive in an instance.

## Release v1.0.1.6 (October 27, 2016)
----------------
* FIX: odrive-database utility now allows cascade override from config file
* FIX: odrive-database migration script 2-down fixed
* DOC: API Documentation updated to reflect changes to Create/Update object, responses, improved URI examples, and search filtering.
* NEW: Search and List operations support AND filters in addition to default OR
* NEW: Update Object request supports passing updated permissions in new 1.1 format
* NEW: Create Object request supports providing permissions in new 1.1 format

## Release v1.0.1.5 (October 11, 2016)
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

## Release v1.0.1.4 (September 14, 2016)
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
* NEW: CloudWatch metrics that enable the setting of alarms for AWS deployments (an auto-scaling prerequisite)

## Release v1.0.1 (September 2, 2016)
--------------
* ENH: Connection to Zookeeper recovery improvements when timed out
* ENH: Full ACM share information captured for individual permission grants
* FIX: Old schema patch files deleted. Database will need to be dropped due to ACM share model
* DOC: API Documentation updated with detailed permissions structure
* REF: Updated zipfile endpoint internals
* FIX: odrive binary will run as user `odrive` when installed with yum package
* FIX: Major Release number bump at customer request

## Release v0.1.0 (August 23, 2016)
--------------
* REF: Remove broken STANDALONE flag
* FIX: Return 404 instead of 500 when retrieving an object properties and given ID is invalid.
* NEW: Allow caller to specify returned content-disposition format when requesting streams and zipped content
* NEW: Response to create object will now populate callerPermisison
* NEW: Publish Events to Kafka
* REF: Index structure used by Finder now includes docstrings
* ENH: RPMs generated will now create os user `odrive` when installed
* ENH: All API responses returning object now populate callerPermissions
* ENH: US Persons Data and FOIA Exemption state fields now track Yes/No/Unknown instead of True/False
