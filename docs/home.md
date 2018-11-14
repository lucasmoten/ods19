FORMAT: 1A

# Object Drive 1.0 

<table style="width:100%;border:0px;padding:0px;border-spacing:0;border-collapse:collapse;font-family:Helvetica;font-size:10pt;vertical-align:center;"><tbody><tr><td style="padding:0px;font-size:10pt;">Version</td><td style="padding:0px;font-size:10pt;">--Version--</td><td style="width:20%;font-size:8pt;"> </td><td style="padding:0px;font-size:10pt;">Build</td><td style="padding:0px;font-size:10pt;">--BuildNumber--</td><td style="width:20%;font-size:8pt;"></td><td style="padding:0px;font-size:10pt;">Date</td><td style="padding:0px;font-size:10pt;">--BuildDate--</td></tr></tbody></table>

# Group Navigation

## Table of Contents

+ [Service Overview](./)
+ [RESTful API documentation](static/templates/rest.html)
+ [Emitted Events documentation](static/templates/events.html)
+ [Environment](static/templates/environment.html)
+ [Changelog](static/templates/changelog.html)

# Group Service Overview
The Object Drive Service provides for secure storage and high performance retrieval of hierarchical folder organization of objects that are named, owned and managed by users and their groups.

Clients can use this API for manipulating objects including basic CRUD operations, and use of basic filter and sort mechanisms for finding objects available to them.

Features supported:

+ Access control managed through combination of ACM for read access, with granular permissions that may be granted to individual users or groups to delegate the ability to
  + Create children of a folder
  + Read a file or folder
  + Update a file or folder metadata or content stream
  + Delete a file or folder
  + Share access to object to other users or groups
+ Automatic filtering based upon user authorization object associated with credentials.
+ Associated content stream is encrypted at rest using AES-256 CTR encryption.
+ Content streams may be retrieved via serial or partial range request operations and use of ETags.
+ Automatic versioning of objects when metadata or content stream is updated.
+ Objects may be marked deleted, restored from trash, or permanently deleted.
+ Objects created by users are owned by that user unless set to a group for which they are a member.
+ Ownership of objects may be transferred to new resource by existing owner or member of group.
+ Auxiliary operations for packaging several objects into a compressed archive (zip)
+ Optionally built integrating BoringCrypto module for FIPS 140-2 compliance.  When the service is built with Go using this module, the version identifier will have a suffix matching that of the BoringCrypto update version (currently b4)

## Service Dependencies

+ AAC Service
+ AWS S3
+ MySQL
+ Kafka
+ ZooKeeper

## Architecture Diagram

<img src="static/images/odrive-service.png" alt="Object Drive Service" width="600" align="middle" />

## Setup

Refer to the [Environment](static/templates/environment.html) variables page for the purpose of each configuration option for setting up object drive whether using an RPM installation, or Docker Image launched with Docker Compose. Several values have defaults, but you do have to configure the following at a minimum:
* MySQL/MariaDB/Aurora Database, which you can populate using the odrive-database tool, or use a premade docker image
* Database connection settings: OD_DB_CA, OD_DB_CERT, OD_DB_HOST, OD_DB_KEY, OD_DB_PASSWORD, OD_DB_PORT, OD_DB_SCHEMA, OD_DB_USERNAME
* AAC connection settings: OD_AAC_CA, OD_AAC_CERT, OD_AAC_CN, OD_AAC_KEY
* Server connection settings: OD_SERVER_CA, OD_SERVER_CERT, OD_SERVER_KEY
* Server Encryption Key: OD_ENCRYPT_MASTERKEY

Refer to the [Software Installation Procedures Guide](https://docs.google.com/document/d/1BV0mv-HePAfOJ0C1SLl1Dr6tKj1TRkgMYSOnSSbQ16s/edit#heading=h.cq93k7j2zwk3) for detailed guidance.


##  Reference Examples

Detailed code examples that use the API:

[Java Caller (create an object)](static/templates/ObjectDriveSDK.java)

The http level result of calling APIs that happens inside of SSL:

[Actual Traffic - Basic Operations](static/templates/APISample.html)

[Drive UI](/apps/drive/index.html)

## Client Libraries

### Go

A [client library](static/client.go) for projects using Go is available. It supports the following capabilities

+	ChangeOwner - Ownership grants full CRUDS, and also controls who can change ownership or move an object
+ CopyObject - Makes a full copy of an object including properties and whichever revisions the user can see and sets the calling user as the owner of the new object
+	CreateObject - For file or folder creation, with and without content streams
+	DeleteObject - Marks as deleted, sending the object to the trash
+ ExpungeObject - Deletes the object. Not restorable from trash
+	GetObject - Retrieves the properties of an object
+	GetObjectStream - Retrieves the content stream / body of an object
+	GetRevisions - Listing of object revisions
+	MoveObject - Changes the parent reference for an object
+	Search - For listing objects at root, under a folder, and filters across all objects
+	UpdateObject - Updating just the metadata and properties of an object
+	UpdateObjectAndStream - Used when also need to update the content stream

### Java

A [client library](https://gitlab.363-283.io/bedrock/object-drive-client) for projects using Java is available. It supports the following capabilities

+ Create a new document in Object Drive
+ Retrieve an existing document from Object Drive
+ Update an existing document in Object Drive
+ Delete an existing document from Object Drive
+ Retrieve a Folder by Name and create it if it doesn't already exist