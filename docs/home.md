FORMAT: 1A

# Object Drive Microservice API
A series of microservices will be exposed on the API gateway for use of Object Drive. These services will be available in Thrift and REST formats to support access by a variety of end user application types. A listing of microservice operations is summarized in the table below

##  Reference Examples

[Update (from real traffic)](TestUpdate.html)

[Share (from real traffic)](TestShare.html)

# Group CRUD Object Operations

---

## Get an Object [/objects/{objectId}/properties]

+ Parameters
    + objectId (string, required) - string Hex encoded identifier of the object to be retrieved.

### Get an Object [GET]
This microservice operation retrieves the metadata about an object. This operation is used to display properties when selecting an object in the system. It may be called on deleted objects which also expose additional fields in the response.

+ Response 200
    + Attributes (ObjectResp)

+ Response 400
    Malformed Request
+ Response 403
    Unauthorized
+ Response 404
    Not Found
+ Response 405
    Deleted
+ Response 410
    Does Not Exist
+ Response 500
    Error Retrieving Object

## Get an Object Stream [/objects/{objectId}/stream]

+ Parameters
    + objectId (string, required) - string Hex encoded identifier of the object to be retrieved.

### Get an Object Stream [GET]
This microservice operation retrieves the content stream of an object as an array of bytes

Note that the content length returned could be very large.  So the client must be prepared to handle
files that are too large to buffer in memory.

+ Response 200

    + Headers

          Content-Length: 26
          Cache-Control: no-cache
          Connection: keep-alive
          Content-Type: text
          Date: Wed, 24 Feb 2016 23:34:20 GMT
          Expires: Wed, 24 Feb 2016 23:34:19 GMT
          Server: nginx/1.8.1

+ Response 400
  Unable to decode request
+ Response 403
  Unauthorized
+ Response 405
  Deleted
+ Response 410
  Does Not Exist
+ Response 500
  Error storing metadata or stream


## List Objects [/objects?pageNumber={pageNumber}&pageSize={pageSize}&sortField={sortField}&sortAscending={sortAscending}&filterField={filterField}&condition={condition}&expression={expression}]

+ Parameters
    + pageNumber (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize (number, optional) - The number of results to return per page.
    + sortField (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
    + sortAscending (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
    + filterField (string, optional) - Denotes a field that the results should be filtered on. Can be specified multiple times.
        If filterField is set, condition and expression must also be set to complete the tupled filter query.
    + condition (enum[string], optional) 

        The type of filter to apply.

        + Members
            + `equals`
            + `contains`
            
    + expression (string, optional) - A phrase that should be used for the match against the field value

### List Objects [GET]

 This microservice operation retrieves a list of objects contained within the specified parent, with optional settings for pagination, sorting, and filtering.

+ Response 200
    + Attributes (ObjectResultset)

+ Response 400
  Unable to decode request
+ Response 403
  Unauthorized
+ Response 405
  Deleted
+ Response 410
  Does Not Exist
+ Response 500
  Error storing metadata or stream

## List Objects Trashed [/trashed?pageNumber={pageNumber}&pageSize={pageSize}&sortField={sortField}&sortAscending={sortAscending}&filterField={filterField}&condition={condition}&expression={expression}]

+ Parameters
    + pageNumber (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize (number, optional) - The number of results to return per page.
    + sortField (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
    + sortAscending (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
    + filterField (string, optional) - Denotes a field that the results should be filtered on. Can be specified multiple times.
        If filterField is set, condition and expression must also be set to complete the tupled filter query.
    + condition (enum[string], optional) 

        The type of filter to apply.

        + Members
            + `equals`
            + `contains`
            
    + expression (string, optional) - A phrase that should be used for the match against the field value

### List Objects Trashed [GET]

+ Response 200
    + Attributes (ObjectResultset)

+ Response 400
  Unable to decode request
+ Response 500
  Error storing metadata or stream

## Create an Object [/objects]

### Create an Object [POST]
Create a new object in Object Drive.
Requires multipart form-data. Send the json metadata before sending
the (possibly very large) payload.

Note that the real file name shows up in the multipart form-data filename attribute, but it
can be overridden with the name attribute in the json data.

The returned json is the metadata that can be used for further operations on the data, such as update,
delete, etc.  The json representing an object is uniform so that it is a similar representation when
it comes back from creation, or from getting an object listing, or from an update.

+ Request
    + Headers
        Host: dockervm:8080
        User-Agent: Go-http-client/1.1
        Content-Length: 565
        Content-Type: multipart/form-data; boundary=7518615725aff2855bc2024fe2ca40d3555eafbcc5a59794c866ed2c8eae
        Accept-Encoding: gzip
    + Body
        --7518615725aff2855bc2024fe2ca40d3555eafbcc5a59794c866ed2c8eae
        Content-Disposition: form-data; name="ObjectMetadata"
        Content-Type: application/json

        {
        "typeName": "{typeName}",
        "name": "{name}",
        "acm": "{acm}",
        }
        --7518615725aff2855bc2024fe2ca40d3555eafbcc5a59794c866ed2c8eae
        Content-Disposition: form-data; name="filestream"; filename="test.txt"
        Content-Type: application/octet-stream

        asdfjklasdfjklasdfjklasdf

        --7518615725aff2855bc2024fe2ca40d3555eafbcc5a59794c866ed2c8eae--

+ Response 200
    + Attributes (ObjectResp)

+ Response 400
  Unable to decode request
+ Response 403
  Unauthorized
+ Response 405
  Deleted
+ Response 410
  Does Not Exist
+ Response 500
  Error storing metadata or stream

## List Objects Under Parent [/objects/{objectId}?pageNumber={pageNumber}&pageSize={pageSize}&sortField={sortField}&sortAscending={sortAscending}&filterField={filterField}&condition={condition}&expression={expression}]

+ Parameters
    + pageNumber (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize (number, optional) - The number of results to return per page.
    + sortField (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
    + sortAscending (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
    + filterField (string, optional) - Denotes a field that the results should be filtered on. Can be specified multiple times.
        If filterField is set, condition and expression must also be set to complete the tupled filter query.
    + condition (enum[string], optional) 

        The type of filter to apply.

        + Members
            + `equals`
            + `contains`
            
    + expression (string, optional) - A phrase that should be used for the match against the field value


### List Object Under Parent [GET]
Purpose: This microservice operation retrieves a list of objects contained within the specified parent, with optional settings for pagination. By default, this operation only returns metadata about the first 20 items.

+ Request
    + Headers
            Host: fully.qualified.domain.name
            Content-Type: application/json

+ Response 200 (application/json)
    + Attributes (ObjectResultset)

+ Response 400
  Unable to decode request
+ Response 403
  Unauthorized
+ Response 405
  Deleted
+ Response 410
  Does Not Exist
+ Response 500
  Error retrieving object represented as the parent to retrieve children, or some other error.

## Update Object [/objects/{objectId}/properties]

+ Parameters
    + objectId (string, required) - Unique identifier of the object to be updated.

### Update Object [POST]
This microservice operation facilitates updating the metadata of an existing object with new settings.
This creates a new version of the object. A changeToken must always be provided for updates.

+ Request (application/json)

        {
            "changeToken": "<changeToken>",
            "name": "<name>",
            "description": "<description>",
            "acm": "{\"version\":\"2.1.0\",\"classif\":\"S\"}",
            "properties": [
                {
                    "Name": "<propertyName>",
                    "Value": "<propertyValue>",
                    "ClassificationPM": "<portionMarkedClassificationOfPropertyValue>"
                },
                {
                    "Name": "<propertyName>",
                    "Value": "<propertyValue>",
                    "ClassificationPM": "<portionMarkedClassificationOfPropertyValue>"
                }
            ]
        }

+ Response 200 (application/json)
    + Attributes (ObjectResp)

+ Response 400
  Unable to decode request
+ Response 403
  Unauthorized
+ Response 404
  The requested object is not found.
+ Response 405
  Deleted
+ Response 410
  Does Not Exist
+ Response 500
  Error storing metadata or stream

## Update Object Stream [/objects/{objectId}/stream]

### Update Object Stream [POST]

Updates the actual file bytes associated with an objectId. Requires a multipart upload.

This creates a new version of the object.

Passing a changeToken is mandatory.

+ Request
    + Headers
        Host: dockervm:8080
        User-Agent: Go-http-client/1.1
        Content-Length: 565
        Content-Type: multipart/form-data; boundary=cc288bcda613e3b659a50ced49542b8676d99ce68f964346ff8a700318de
        Accept-Encoding: gzip

+ Response 200
    + Attributes (ObjectResp)

+ Response 400
  Unable to decode request
+ Response 403
  Unauthorized
+ Response 404
  The object was not found
+ Response 405
  Deleted
+ Response 410
  Does Not Exist
+ Response 500
  Error storing metadata or stream

## Delete Object [/objects/{objectId}/trash]

+ Parameters
    + objectId (string, required) - Hex encoded identifier of the object to be deleted.
    + changeToken (string, required) - A hash value expected to match the targeted object's current changeToken value.

### Delete Object [POST]
This microservice operation handles the deletion of an object within Object Drive. When objects are deleted, they are marked as such but remain intact for auditing purposes and the ability to restore (remove from trash). All other operations that pertain to retrieval or updating filter deleted objects internally. The exception to this is when viewing the contents of the trash via listObjectsTrashed operation, removeObjectFromTrash, and deleteObjectForever.

When an object is deleted, a recursive action is performed on all natural children to set their isAncestorDeleted to true, unless that child object is deleted, in which case, the recursion down that branch terminates.

+ Request
    + Headers
        Content-Type: application/json
        Content-Length: nnn

    + Body
        {
            "changeToken": "{changeToken}"
        }

+ Response 200
    + Headers
        Content-Type: application/json
        Content-Length: nnn

    + Body
        {
            "deletedDate": "{deletedDate}"
        }

+ Response 400
  Unable to decode request
+ Response 403
  Unauthorized
+ Response 405
  Deleted
+ Response 410
  Does Not Exist
+ Response 500
  Error storing metadata or stream

## Delete Object Forever [/objects/{objectId}]  

+ Parameters
     + objectId (string, required) - Hex encoded identifier of the object to be deleted.
     + changeToken (string, required) - A hash value expected to match the targeted object's current changeToken value.

### Delete Object Forever [DELETE]
This microservice operation will remove an object from the trash and delete it forever.  

+ Request
    + Headers
        Content-Type: application/json
        Content-Length: nnn

    + Body
        {
            "changeToken": "{changeToken}"   
        }

+ Response 200
    + Headers
        Content-Type: application/json
        Content-Length: nnn

    + Body
        {
            "expungedDate": "{expungedDate}"
        }

+ Response 400
  Unable to decode request
+ Response 403
  Unauthorized
+ Response 405
  Deleted
+ Response 410
  Does Not Exist
+ Response 500
  Error storing metadata or stream

## Undelete Object [/objects/{objectId}/untrash]

+ Parameters
     + objectId (string, required) - Hex encoded identifier of the object to be deleted.
     + changeToken (string, required) - A hash value expected to match the targeted object's current changeToken value.

### Undelete Object [POST]

+ Request

    + Headers
            Content-Type: application/json
            Content-Length: nnn

    + Body

        {
            "changeToken": "{changeToken}"   
        }


+ Response 200
    + Attributes (ObjectResp)

+ Response 403
  User is unauthorized
+ Response 405
  Referenced object is deleted.  
+ Response 410
  Referenced object no longer exists
+ Response 500
          Error storing metadata or stream

# Group Access Control Operations

---

## Add Object Share [/shared/{objectId}]

+ Parameters
    + objectId (string, required) - Unique identifier of the object to be shared.
    + grantee (string, required) - Unique identifier of a user or group of the system that will be given shared access to the object.
    + allowRead (boolean, required) - Denotes whether the grantee will have permission to read the object.
    + allowWrite (boolean, required) - Denotes whether the grantee will have permission to modify the object, and where appropriate, create child objects.
    + allowDelete (boolean, required) - Denotes whether the grantee will have permission to delete an object and its children.
    + allowShare (required, boolean) - Denotes whether the grantee will have permission to share this object, and its children, limited to the same permissions this grantee has been given.

### Add Object Share [POST]
This microservice operation is intended to provide a wrapper around updateObjectPermissions to grant the specified permission, as well as record that the share has been established, and notify the grantee of the share that it is available for them. Regardless of sharing settings, as with standard object permissions, a user is still required to pass all other checks to be able to access objects.
The user to which the object is to be granted must exist in the object-drive system.
Users are added when they visit this API, so the grantee user must visit object-drive in order to have a record in the system.

+ Request
    + Headers
        Host: dockervm:4430
        User-Agent: Go-http-client/1.1
        Content-Length: 206
        Content-Type: application/json
        Accept-Encoding: gzip

    + Body
        {
            "grantee": "CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
            "create": true,
            "read": true,
            "update": true,
            "delete": true,
            "share": false,
            "propagateToChildren": false
        }

+ Response 200
    + Attributes (Permission)

+ Response 400
  Unable to decode request
+ Response 403
  Unauthorized
+ Response 405
  Deleted
+ Response 410
  Does Not Exist
+ Response 500
  Error storing metadata or stream

## List User Object Shares [/shares?pageNumber={pageNumber}&pageSize={pageSize}&sortField={sortField}&sortAscending={sortAscending}&filterField={filterField}&condition={condition}&expression={expression}]

+ Parameters
    + pageNumber (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize (number, optional) - The number of results to return per page.
    + sortField (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
    + sortAscending (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
    + filterField (string, optional) - Denotes a field that the results should be filtered on. Can be specified multiple times.
        If filterField is set, condition and expression must also be set to complete the tupled filter query.
    + condition (enum[string], optional) 

        The type of filter to apply.

        + Members
            + `equals`
            + `contains`
            
    + expression (string, optional) - A phrase that should be used for the match against the field value

### List User Object Shares [GET]
This microservice operation retrieves a list of objects that the user has shared to them, by others.

+ Request    
    + Headers
        Content-Type: application/json

+ Response 200
    + Attributes (ObjectResultset)

+ Response 400
  Unable to decode request
+ Response 403
  Unauthorized
+ Response 405
  Deleted
+ Response 410
  Does Not Exist
+ Response 500
  Error storing metadata or stream

## List User Object Shared [/shared]

+ Parameters
    + pageNumber (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize (number, optional) - The number of results to return per page.
    + sortField (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
    + sortAscending (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
    + filterField (string, optional) - Denotes a field that the results should be filtered on. Can be specified multiple times.
        If filterField is set, condition and expression must also be set to complete the tupled filter query.
    + condition (enum[string], optional) 

        The type of filter to apply.

        + Members
            + `equals`
            + `contains`
            
    + expression (string, optional) - A phrase that should be used for the match against the field value

### List User Object Shared [GET]
This microservice operation retrieves a list of objects that the user has shared to others.

+ Request
    + Header
        Content-Type: application/json

+ Response 200
    + Attributes (ObjectResultset)

+ Response 400
  Unable to decode request
+ Response 403
  Unauthorized
+ Response 405
  Deleted
+ Response 410
  Does Not Exist
+ Response 500
  Error storing metadata or stream

## Remove Object Share [/shared/{objectId}/{shareId}]

+ Parameters
    + objectId (string, required) - Unique identifier of the object for which a share is to be deleted.
    + shareId (string, required) - Unique identifier of the share on the object that is to be deleted.
    + changeToken (string, required) - A hash value expected to match the targeted object share's current changeToken value.
    + propagateToChildren (boolean, required) - Indicates whether matching inherited shares on any child objects of the targeted object should also be removed recursively.

### Remove Object Share [DELETE]
This microservice operation removes a previously defined object share.

+ Request
    + Headers
        Content-Type: application/json
        Content-Length: nnn

    + Body
        {
            "changeToken": "{changeToken}",
            "propagateToChildren": "{propagateToChildren}"
        }

+ Response 200


    + Headers
        Content-Type: application/json
        Content-Length: nnn

    + Body
        {
            "deletedDate": {deletedDate}
        }

+ Response 400
  Unable to decode request
+ Response 403
  Unauthorized
+ Response 405
  Deleted
+ Response 410
  Does Not Exist
+ Response 500
  Error storing metadata or stream

# Group Filing Operations

---

## Move Object [/objects/{objectId}/move/{folderId}]

+ Parameters
    + objectId (string, required) - Unique identifier of the object to be moved.
    + folderId (string, required) - Unique identifier of the folder into which this object should be moved.

### Move Object [POST]
This microservice operation supports moving an object such as a file or folder from one location to another. By default, all objects are created in the ‘root’ as they have no parent folder given.

+ Request
    + Headers
        Content-Type: application/json
        Content-Length: nnn

    + Body
        {
            "changeToken": "a32836052f06abb6a5e5c2d91bdcb27ece440512"
        }

+ Response 200
    + Attributes (ObjectResp)

+ Response 400
  Unable to decode request
+ Response 403
  Unauthorized
+ Response 404
  The requested object is not found.
+ Response 405
  Deleted
+ Response 410
  Does Not Exist
+ Response 428
  If the changeToken does not match expected value.
+ Response 500
  Error storing metadata or stream

# Group User Centric Operations

---

## List User Object Shares [/shares?pageNumber={pageNumber}&pageSize={pageSize}&sortField={sortField}&sortAscending={sortAscending}&filterField={filterField}&condition={condition}&expression={expression}]

+ Parameters
    + pageNumber (number, required) - The page number of results to be returned to support chunked output.
    + pageSize (number, required) - The number of results to return per page.
    + sortField (string) - Denotes a field that the results should be sorted on.
    + sortAscending (boolean) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
    + filterField (string) - Denotes a field that the results should be filtered on.

### List User Object Shares [GET]
This microservice operation retrieves a list of objects that the user has shared to them.

+ Request    
    + Headers
        Content-Type: application/json

    + Body
        {
            "pageNumber": "{pageNumber}",
            "pageSize": "{pageSize}"
        }

+ Response 200
    + Attributes (ObjectResultset)

+ Response 400
  Unable to decode request
+ Response 500
  Error retrieving shared objects.

## List User Objects Shared [/shared?pageNumber={pageNumber}&pageSize={pageSize}&sortField={sortField}&sortAscending={sortAscending}&filterField={filterField}&condition={condition}&expression={expression}]

+ Parameters
    + pageNumber (number, required) - The page number of results to be returned to support chunked output.
    + pageSize (number, required) - The number of results to return per page.
    + sortField (string) - Denotes a field that the results should be sorted on.
    + sortAscending (boolean) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
    + filterField (string) - Denotes a field that the results should be filtered on.
### List User Object Shared [POST]
This microservice operation retrieves a list of objects that the user has shared to others.

+ Request
    + Header
        Content-Type: application/json

    + Body
        {
            "pageNumber": "{pageNumber}",
            "pageSize": "{pageSize}"
        }

+ Response 200
    + Attributes (ObjectResultset)

+ Response 400
  Unable to decode request
+ Response 403
  Unauthorized
+ Response 405
  Deleted
+ Response 410
  Does Not Exist
+ Response 500
  Error storing metadata or stream

## User Stats [/userstats]

### User Stats [GET]

+ Response 200
    + Attributes (UserStats)
    
+ Response 500
  Internal Server Error

# Group Versioning Operations

---

## Get Object Revisions [/revisions/{objectId}?pageNumber={pageNumber}&pageSize={pageSize}&sortField={sortField}&sortAscending={sortAscending}]

+ Parameters
    + objectId - The object id.
    + pageNumber (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize (number, optional) - The number of results to return per page.
    + sortField (string, optional) - Denotes a field that the results should be sorted on.
    + sortAscending (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.

### Get Object Revision [GET]

+ Response 200
    + Attributes (ObjectResultset)

+ Response 204
+ Response 403
  If the user is unauthorized to perform the request because they lack permissions to view the object.
+ Response 405
  If the object or an ancestor is deleted.
+ Response 410
  If the object referenced no longer exists
+ Response 500
  * Malformed JSON
  * Error retrieving object
  * Error determining user.


## Get Object Stream Revision [/revisions/{objectId}/{revisionId}/stream]

+ Parameters
    + revisionId (number, required) - The incrementing integer revision count
    + objectId (string, required) - The object id.

### Get Object Stream Revision [GET]

+ Response 200
    + Attributes (ObjectResultset)

+ Response 204
  If the object has no content stream, a successful response indicating no content will be returned

+ Response 403
  If the user is unauthorized to perform the request because they lack permissions to view the object.
+ Response 405
  If the object or an ancestor is deleted.
+ Response 410
  If the object referenced no longer exists
+ Response 500
  * Malformed JSON
  * Error retrieving object
  * Error determining user.

# Group Search Operations

---

## Search [/search/{searchPhrase}?pageNumber={pageNumber}&pageSize={pageSize}&sortField={sortField}&sortAscending={sortAscending}&filterField={filterField}&condition={condition}&expression={expression}]

+ Parameters
    + pageNumber (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize (number, optional) - The number of results to return per page.
    + sortField (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
    + sortAscending (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
    + filterField (string, optional) - Denotes a field that the results should be filtered on. Can be specified multiple times.
        If filterField is set, condition and expression must also be set to complete the tupled filter query.
    + condition (enum[string], optional) 

        The type of filter to apply.

        + Members
            + `equals`
            + `contains`
            
    + expression (string, optional) - A phrase that should be used for the match against the field value

### Search [GET]

+ Response 200
    + Attributes (ObjectResultset)

+ Response 204
  If the object has no content stream, a successful response indicating no content will be returned
+ Response 400
  Malformed request.
+ Response 500
  * Error retrieving object
  * Error determining user.

# Data Structures

## Permissions (object)

+ id: `11e5e4867a6e3d8389020242ac110002` (string) -  The unique identifier of the permission associated to the object hex encoded to a string. This value can be used for deleting the permission.
+ createdDate: `2016-03-07T17:03:13Z` (string) -  The date and time the permission was created in the system in UTC ISO 8601 format.
+ createdBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) -  The user that created the permission.
+ modifiedDate: `2016-03-07T17:03:13Z` (string) -  The date and time the permission was last modified in the system in UTC ISO 8601 format.
+ modifiedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) -  The user that last modified this permission.
+ changeCount: 2 (number) - number The total count of changes that have been made to this permission over its lifespan. Synonymous with version number.
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) -  A hash of the permissions's unique identifier and last modification date and time.
+ objectId: `11e5e4867a6e3d8389020242ac110002` (string) -  The unique identifier of the object that this permission is associated with.
+ grantee: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) -  The user for whom this permission is granted to
+ allowCreate: (boolean) -  Indicates whether the grantee can create child objects under the referenced object of this permission.
+ allowRead: (boolean) -  Indicates whether the grantee can view the object referenced by this permission.
+ allowUpdate: (boolean) -  Indicates whether the grantee can modify the object referenced by this permission.
+ allowDelete: (boolean) -  Indicates whether the grantee can delete the object referenced by this permission.
+ allowShare: (boolean) -  Indicates whether the grantee can reshare the object referenced by this permission.
+ explicitShare: (boolean) -  Indicates whether this permission was explicitly created by a call to add the grant, or if it was inherited from the parent object for which a child was made or propagated creation on an existing object.

## ObjectResp (object)

+ id: `11e5e4867a6e3d8389020242ac110002`  (string, required) - string The unique identifier of the object hex encoded to a string. This value can be used for alterations and listing on other RESTful methods.
+ createdDate: `2016-03-07T17:03:13Z`  (string) - The date and time the object was created in the system in UTC ISO 8601 format.
+ createdBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that created the object.
+ modifiedDate: `2016-03-07T17:03:13Z` (string) -  The date and time the object was last modified in the system in UTC ISO 8601 format. For unchanged objects, this will reflect the same value as the createdDate field.
+ modifiedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that last modified this object. For unchanged objects, this will reflect the same value as the createdBy field.
+ deletedDate: `2016-03-07T17:03:13Z` (string, optional) -  The date and time the object was deleted in the system in UTC ISO 8601 format. This field is only present if the object is in the trash.
+ deletedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that deleted the object. This field is only present if the object is in the trash.
+ changeCount: 42 (number) - The total count of changes that have been made to this object over its lifespan. Synonymous with version number. For unchanged objects, this will always be 0.
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.
+ ownedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that owns the object.
+ typeId: `11e5e48664f5d8c789020242ac110002` (string) - The unique identifier of the type assigned this object hex encoded to a string.
+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `gettysburgaddress.txt` (string) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
+ description: `Description here` (string) - An abstract of the object's purpose.
+ parentId: ` ` (string, optional) - The unique identifier of the objects parent hex encoded to a string. This may be used to traverse up the tree. For objects stored at the root of a user, this value will be null.
+ acm: `{\"version\":\"2.1.0\",\"classif\":\"S\"}` (string) - The raw acm value associated with this object.
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value will be null.
+ contentSize: 1511 (string) - The length of the object's content stream, if present. For objects without a content stream, this value will be 0.
+ properties: (array[Properties]) - Array of custom properties associated with the object.
+ permissions: (array[Permissions]) - Array of permissions associated with this object.

## Properties (object)

+ id: `11e5e4867a6e3d8389020242ac110002` (string) - The unique identifier of the property associated to the object hex encoded to a string.
+ createdDate: `2016-03-07T17:03:13Z` (string) - The date and time the property was created in the system in UTC ISO 8601 format.
+ createdBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) -  The user that created the property .
+ modifiedDate: `2016-03-07T17:03:13Z` (string) - The date and time the property was last modified in the system in UTC ISO 8601 format.
+ modifiedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) -  The user that last modified this property .
+ changeCount: 1 (number) - The total count of changes that have been made to this property over its lifespan. Synonymous with version number.
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) -  A hash of the property's unique identifier and last modification date and time.
+ name: `Some Property` (string) - The name, key, field or label given to a property for usability
+ propertyValue: `Some Property Value` (string) -  The value assigned for the property
+ classificationPM: `TS` (string) -  The portion mark classification for the value of this property

## ObjectResultset (object)

+ totalRows: 100 (number) - Total number of items matching the query.
+ pageCount: 10 (number) - Total rows divided by page size.
+ pageNumber: 1 (number) - Requested page number for this resultset.
+ pageSize: 10 (number) - Requested page size for this resultset.
+ pageRows: 10 (number) - Number of items included in this page of the results, which may be less than pagesize, but never greater.
+ objects: (array[ObjectResp])

## ObjectStorageMetric (object)

+ typeName: `File` (string) - The type of object, which is usually File or Folder.
+ objects: 24 (number) - The number of current objects that are stored.
+ objectsWithRevision: 432 (number) - The number of versioned objects that are stored.
+ objectsSize: 249234 (number) - The total size of objects in bytes, which could be a very large number.
+ objectsWithRevisionSize: 23478234 (number) - The total size of versioned objects in bytes, which may be very large.

## OserStats (object)

+ totalObjects: 24 (number) - The number of current objects that are stored.
+ totalObjectsWithRevision: 432 (number) - The number of versioned objects that are stored.
+ totalObjectsSize: 249234 (number) - The total size of objects in bytes, which could be a very large number.
+ totalObjectsWithRevisionSize: 23478234 (number) - The total size of versioned objects in bytes, which may be very large.
+ objectStorageMetrics: (array[ObjectStorageMetric])