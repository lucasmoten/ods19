FORMAT: 1A

# Object Drive 1.0 

# Group Navigation

## Table of Contents

+ [Service Overview](./)
+ [RESTful API documentation](static/templates/rest.html)
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
+ All retrieval operations are automatically filtered based upon user authorization object associated with credentials.
+ Associated content stream is encrypted at rest using AES-256 CTR encryption.
+ Content streams of objects may be retrieved via traditional serial or partial range request operations and include support for ETags.
+ Automatic versioning of objects when metadata or content stream is updated with ability to retrieve prior versions
+ Objects may be marked deleted, restored from trash, or permanently deleted.
+ Objects created by users are by default owned by that user unless assigned to a group for which they are a member.
+ Ownership of objects may be transferred from the current owner, or member of a group if owned by a group.
+ Auxiliary operations for packaging several objects into a compressed archive (zip) or performing bulk operations


## Service Dependencies

+ AAC Service
+ AWS S3
+ MySQL
+ Kafka
+ ZooKeeper

## Architecture Diagram

<img src="static/images/odrive-service.png" alt="Object Drive Service" width="600" align="middle" />

## Setup
[Environment](static/templates/environment.html)

## List of Operations 

A listing of microservice operations is summarized in the table below. 

If you want to develop against Object Drive, see the [API Documentation](static/templates/api.html)

| Name | Purpose |
| --- | --- |
| Create an Object | Main operation to add a new object to the system. |
| Get an Object | Retrieves the metadata, properties and permissions of an object. |
| Get bulk objects | Retrieves the metadata for multiple, explicitly enumerated, objects. |
| Get an Object Stream | Retrieves the content stream of an object. |
| Update Object | Used for updating the metadata of an object. |
| Update Object Stream | Used for updating the content stream and metadata of an object. |
| Delete Object | Marks an object as deleted an only available from the user's trash. |
| Delete Objects | Delete objects in bulk. |
| Delete Object Forever | Expunges an object so that it cannot be restored from the trash. |
| List Object Revisions | Retrieves a resultset of revisions for an object. |
| Get Object Revision Stream | Retrieves the content stream of a specific revision of an object. |
| Search | Retrieves a resultset of objects matching search parameters against the name and description. |
| List Objects at Root For Group | Retrieves a resultset of objects at a group's root. |
| List Objects Under Parent | Retrieves a resultset of objects contained in/under a parent object (ie., folder). |
| List Objects Shared to Everyone | Retrieves a resultset of objects that are shared to everyone. |
| Move Objects | Move objects in bulk. |
| Move Object | Changes the hierarchial placement of an object. |
| Change Owner | Change the owner of an object. |
| Change Owner Bulk | Change the owner of objects. |
| List Objects at Root For User | Retrieves a resultset of objects at the user's root. |
| List Object Shares | Retrieves a resultset of objects shared to the user. |
| List Objects Shared | Retreives a resultset of objects that the user has shared. |
| List Trashed Objects | Retrieves a resultset of objects in the user's trash. |
| Undelete Object | Restores an object from the user's trash. |
| Empty Trash | Expunges all objects in the user's trash. |
| User Stats | Retrieve information for user's storage consumption. |
| Zip Files | Get a zip of some files. |


##  Reference Examples

Detailed code examples that use the API:

[Java Caller (create an object)](static/templates/ObjectDriveSDK.java)

The http level result of calling APIs that happens inside of SSL:

[Actual Traffic - Basic Operations](static/templates/APISample.html)

[Drive UI](/apps/drive/index.html)
