FORMAT: 1A

# Object Drive Microservice API
A series of microservice operations are exposed on the API gateway for use of Object Drive. These services are in REST format. A listing of microservice operations is summarized in the table below

## Summary of Operations Available

| Name | Purpose |
| --- | --- |
| Create an Object | Main operation to add a new object to the system. |
| Get an Object | Retrieves the metadata, properties and permissions of an object. |
| Get an Object Stream | Retrieves the content stream of an object. |
| List Objects at Root | Retrieves a resultset of objects at the user's root. |
| List Objects Under Parent | Retrieves a resultset of objects contained in/under a parent object (ie., folder). |
| Update Object | Used for updating the metadata of an object. |
| Update Object Stram | Used for updating the content stream and metadata of an object. |
| Delete Object | Marks an object as deleted an only available from the user's trash. |
| Delete Object Forever | Expunges an object so that it cannot be restored from the trash. |
| Add Object Share | Creates a share for an object to another user or group. |
| Remove Object Share | Removes a share previously created between an object and user or groups. |
| List Object Revisions | Retrieves a resultset of revisions for an object. |
| Get Object Revision Stream | Retrieves the content stream of a specific revision of an object. |
| Search | Retrieves a resultset of objects matching search parameters against the name and description. |
| Move Object | Changes the hierarchial placement of an object |
| List Object Shares | Retrieves a resultset of objects shared to the user. |
| List Objects Shared | Retreives a resultset of objects that the user has shared. |
| List Trashed Objects | Retrieves a resultset of objects in the user's trash. |
| Undelete Object | Restores an object from the user's trash. |
| User Stats | Retrieve information for user's storage consumtpion. |


##  Reference Examples

Detailed code examples that use the API:

[Java Caller (create an object)](static/templates/ObjectDriveSDK.java)

[Javascript Caller (our simple test user interface)](static/templates/listObjects.js)

The http level result of calling APIs that happens inside of SSL:

[Update (from real traffic)](static/templates/TestUpdate.html)

[Share (from real traffic)](static/templates/TestShare.html)

[Testing interface (for development)](ui)

# Group CRUD Object Operations
These basic operations provide support for creating, retrieving, updating and deleting objects. 

---

## Create an Object [/objects]

### Create an Object [POST]
Create a new object in Object Drive.

The returned json is the metadata that can be used for further operations on the data, such as update,
delete, etc.  The json representing an object is uniform so that it is a similar representation when
it comes back from creation, or from getting an object listing, or from an update.
An acm follows guidance given here: https://confluence.363-283.io/pages/viewpage.action?ageId=557850

+ Request With Content Stream
    When creating a new object with a content stream, such as a file, this must be presented in multipart/form-data format, with the metadata about the object provided in a field named 'ObjectMetadata' containing a JSON structure of the following fields.

    + typeName (string, required) -  The type to be assigned to this object. Common types include 'File', 'Folder'. Custom types may be referenecd here for the purposes of rules or processing constraints and definition of properties.
    + name (string, optional) - The name to be given this object.  If no name is given, then objects are created with the default name pattern of `New <typeName>`.
    + description (string, optional) - An optional abstract of the object's contents.
    + parentId (string, optional) - Hex encoded identifier of an object, typically a folder, into which this new object is being created as a child object. If no value is specified, then the object will be created in the root location of the user who is creating it.
    + acm (string OR object, required) - Access Control Model is the security model leveraged by the system when enforcing access control. It is based on the ISM, NTK, ACCM and Share standards, requirements and policies.  This value may be provided in either serialized string format, or nested object format.
    + contentType (string, optional) - The suggested mime type for the content stream if given for this object.
    + contentSize (int, optional) - The length of the content stream, in bytes. If there is no content stream, this value should be 0.
    + properties (properties array, optional) - Array of custom properties to be associated with the newly created object.
    + permission (permission array, optional) - Array of additional permissions to be associated with this object when created. By default, the owner is granted full access and objects will inherit the permissions assigned to the parent object.
    + isUSPersonsData (boolean, optional) - Indicates if this object contains US Persons data.
    + isFOIAExempt (boolean, optional) - Indicates if this object is exempt from Freedom of Information Act requests.

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
                "description": "{description}",
                "parentId": "{parentId}",
                "acm": {acm},
                "contentType": "{contentType}",
                "contentSize": {contentSize},
                "properties": [{Property}],
                "permissions": [{permissions}],
                "isUSPersonsData": {isUSPersonsData},
                "isFOIAExempt": {isFOIAExempt}
            }
            --7518615725aff2855bc2024fe2ca40d3555eafbcc5a59794c866ed2c8eae
            Content-Disposition: form-data; name="filestream"; filename="test.txt"
            Content-Type: application/octet-stream
            
            asdfjklasdfjklasdfjklasdf
            
            --7518615725aff2855bc2024fe2ca40d3555eafbcc5a59794c866ed2c8eae--

+ Request Without a Content Stream
    When creating a new object without a content stream, such as a folder, the object definition may be specified directly in the request body as typified below.

    + typeName (string, required) -  The type to be assigned to this object. Common types include 'File', 'Folder'. Custom types may be referenecd here for the purposes of rules or processing constraints and definition of properties.
    + name (string, optional) - The name to be given this object.  If no name is given, then objects are created with the default name pattern of `New <typeName>`.
    + description (string, optional) - An optional abstract of the object's contents.
    + parentId (string, optional) - Hex encoded identifier of an object, typically a folder, into which this new object is being created as a child object. If no value is specified, then the object will be created in the root location of the user who is creating it.
    + acm (string OR object, required) - Access Control Model is the security model leveraged by the system when enforcing access control. It is based on the ISM, NTK, ACCM and Share standards, requirements and policies.  This value may be provided in either serialized string format, or nested object format.
    + contentType (string, optional) - The suggested mime type for the content stream if given for this object.
    + contentSize (int, optional) - The length of the content stream, in bytes. If there is no content stream, this value should be 0.
    + properties (properties array, optional) - Array of custom properties to be associated with the newly created object.
    + permission (permission array, optional) - Array of additional permissions to be associated with this object when created. By default, the owner is granted full access and objects will inherit the permissions assigned to the parent object.
    + isUSPersonsData (boolean, optional) - Indicates if this object contains US Persons data.
    + isFOIAExempt (boolean, optional) - Indicates if this object is exempt from Freedom of Information Act requests.

    + Headers
    
            Host: dockervm:8080
            User-Agent: Go-http-client/1.1
            Content-Length: 565
            Content-Type: application/json
            Accept-Encoding: gzip
            
    + Body

            {
                "typeName": "{typeName}",
                "name": "{name}",
                "description": "{description}",
                "parentId": "{parentId}",
                "acm": {acm},
                "contentType": "{contentType}",
                "contentSize": {contentSize},
                "properties": [{properties}],
                "permissions": [{permissions}]
                "isUSPersonsData": {isUSPersonsData},
                "isFOIAExempt": {isFOIAExempt}
            }

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

## Object Metadata [/objects/{objectId}/properties]

Metadata for an object may be retrieved or updated at the URI designated.  

+ Parameters
    + objectId (string, required) - string Hex encoded identifier of the object to be retrieved.

### Get an Object [GET]
This microservice operation retrieves the metadata about an object. 
This operation is used to display properties when selecting an object in the system. 
It may be called on objects int the trash which also expose additional fields in the response.

+ Response 200
    + Attributes (ObjectResp)
    This is the response format for an item that is deleted.
    + Attributes (ObjectRespDeleted)

+ Response 400

        Malformed Request

+ Response 403

        Unauthorized

+ Response 404

        Not Found

+ Response 410

        Does Not Exist

+ Response 500

        Error Retrieving Object

### Update Object [POST]
This microservice operation facilitates updating the metadata of an existing object with new settings.
This creates a new revision of the object. 

+ Request (application/json)

    The JSON object provided in the body can contain the following fields:

    + changeToken (string, required) - A hash value expected to match the targeted object’s current changeToken value. This value is retrieved from get or list operations.
    + typeName (string, optional) -  The new type to be assigned to this object. Common types include 'File', 'Folder'. If no value is provided or this field is omitted, then the type will not be changed.
    + name (string, optional) - The new name to be given this object. It does not have to be unique. It may refer to a conventional filename and extension. If no value is provided, or this field is ommitted, then the name will not be changed.
    + description (string, optional) - The new description to be given as an abstract of the objects content stream. If no value is provided, or this field is ommitted, then the description will not be changed.
    + acm (string OR object, optional) -  Access Control Model (ACM) is the security model leveraged by the system when enforcing access control. It is based on the ISM, NTK, ACCM and Share standards, requirements and policies. https://confluence.363-283.io/pages/viewpage.action?pageId=557850. If no value is provided, or this field is ommitted, then the acm will not be changed. This value may be provided in either serialized string format, or nested object format.
    + properties (properties array, optional) -  An array of custom properties to be associated with this object for property changes. For the properties specified, those who do not match existing properties on the object by name will be added. For the properties that do match existing properties by name, if the value specified is blank or empty, then the existing property will be deleted, otherwise, the property will be updated to the new value. If properties are specified in the array, then existing properties on the object are retained. Properties are only removed from an object if they are provided, with their value set to an empty string.
    + isUSPersonsData (boolean, optional) - Indicates if this object contains US Persons data.
    + isFOIAExempt (boolean, optional) - Indicates if this object is exempt from Freedom of Information Act requests.

    + Body
    
            {
                "changeToken": "{changeToken}",
                "typeName": "{typeName}",
                "name": "{name}",
                "description": "{description}",
                "acm": {acm},
                "properties": [
                    {
                        "Name": "{propertyName}",
                        "Value": "{propertyValue}",
                        "ClassificationPM": "{portionMarkedClassificationOfPropertyValue}"
                    },
                    {
                        "Name": "{propertyName}",
                        "Value": "{propertyValue}",
                        "ClassificationPM": "{portionMarkedClassificationOfPropertyValue}"
                    }
                ],
                "isUSPersonsData": {isUSPersonsData},
                "isFOIAExempt": {isFOIAExempt}
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

## Object Stream [/objects/{objectId}/stream]

The content stream for an object may be retrieved or updated at the URI designated.

+ Parameters
    + objectId (string, required) - Hex encoded identifier of the object to be retrieved.

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

    + Body
    
            The bytes representing the content stream of the object 

+ Response 204

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

### Update Object Stream [POST]

Updates the actual file bytes associated with an objectId. This must be provided in multipart/form-data format, with the metadata about the object provided in a field named 'ObjectMetadata'.

This creates a new revision of the object.
    
+ Request

    The JSON object provided in the body can contain the following fields:

    + changeToken (string, required) - A hash value expected to match the targeted object’s current changeToken value. This value is retrieved from get or list operations.
    + typeName (string, optional) -  The new type to be assigned to this object. Common types include 'File', 'Folder'. If no value is provided or this field is omitted, then the type will not be changed.
    + name (string, optional) - The new name to be given this object. It does not have to be unique. It may refer to a conventional filename and extension. If no value is provided, or this field is ommitted, then the name will not be changed.
    + description (string, optional) - The new description to be given as an abstract of the objects content stream. If no value is provided, or this field is ommitted, then the description will not be changed.
    + acm (string OR object, optional) -  Access Control Model (ACM) is the security model leveraged by the system when enforcing access control. It is based on the ISM, NTK, ACCM and Share standards, requirements and policies. https://confluence.363-283.io/pages/viewpage.action?pageId=557850. If no value is provided, or this field is ommitted, then the acm will not be changed.  This value may be provided in either serialized string format, or nested object format.
    + contentType (string, optional) - The suggested mime type for the content stream if given for this object.
    + contentSize (int, optional) - The length of the content stream, in bytes. If there is no content stream, this value should be 0.
    + properties (properties array, optional) -  An array of custom properties to be associated with this object for property changes. For the properties specified, those who do not match existing properties on the object by name will be added. For the properties that do match existing properties by name, if the value specified is blank or empty, then the existing property will be deleted, otherwise, the property will be updated to the new value. If properties are specified in the array, then existing properties on the object are retained. Properties are only removed from an object if they are provided, with their value set to an empty string.    
    + isUSPersonsData (boolean, optional) - Indicates if this object contains US Persons data.
    + isFOIAExempt (boolean, optional) - Indicates if this object is exempt from Freedom of Information Act requests.

    + Headers
    
            Host: dockervm:8080
            User-Agent: Go-http-client/1.1
            Content-Length: 565
            Content-Type: multipart/form-data; boundary=cc288bcda613e3b659a50ced49542b8676d99ce68f964346ff8a700318de
            Accept-Encoding: gzip
            
    + Body
    
            --cc288bcda613e3b659a50ced49542b8676d99ce68f964346ff8a700318de
            Content-Disposition: form-data; name="ObjectMetadata"
            Content-Type: application/json
            
            {
                "changeToken": "{changeToken}",
                "typeName": "{typeName}",
                "name": "{name}",
                "description": "{description}",
                "acm": {acm},
                "contentType": "{contentType}",
                "contentSize": {contentSize},
                "properties": [
                    {
                        "name": "<propertyName>"
                        ,"value": "<propertyValue>"
                        ,"classificationPM": "<portionMarkedClassificationOfPropertyValue>"
                    }
                ],
                "isUSPersonsData": {isUSPersonsData},
                "isFOIAExempt": {isFOIAExempt}
            }
            --cc288bcda613e3b659a50ced49542b8676d99ce68f964346ff8a700318de
            Content-Disposition: form-data; name="filestream"; filename="test.txt"
            Content-Type: application/octet-stream
            
            asdfjklasdfjklasdfjklasdf
            
            --cc288bcda613e3b659a50ced49542b8676d99ce68f964346ff8a700318de--    

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

## List Objects At Root [/objects?pageNumber={pageNumber}&pageSize={pageSize}&sortField={sortField}&sortAscending={sortAscending}&filterField={filterField}&condition={condition}&expression={expression}]

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

### List Objects At Root [GET]

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

## List Objects Under Parent [/objects/{objectId}?pageNumber={pageNumber}&pageSize={pageSize}&sortField={sortField}&sortAscending={sortAscending}&filterField={filterField}&condition={condition}&expression={expression}]

+ Parameters
    + objectId (string, required) - Hex encoded unique identifier of the folder or other object for which to return a list of child objects. 
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

## Delete Object [/objects/{objectId}/trash]

+ Parameters
    + objectId (string, required) - Hex encoded identifier of the object to be deleted.

### Delete Object [POST]
This microservice operation handles the deletion of an object within Object Drive. When objects are deleted, they are marked as such but remain intact for auditing purposes and the ability to restore (remove from trash). All other operations that pertain to retrieval or updating filter deleted objects internally. The exception to this is when viewing the contents of the trash via List Trashed Objects, or performing Undelete Object, and Delete Object Forever operations.

This creates a new revision of the object.

When an object is deleted, a recursive action is performed on all natural children to set an internal flag denoting an ancestor as deleted, unless that child object is deleted, in which case, the recursion down that branch terminates.

+ Request

    The JSON object in the request body should contain a change token:
    + changeToken (string, required) - A hash value expected to match the targeted object's current changeToken value.  This value should be set with the same value returned from any list operation or the Get Object operation.

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

### Delete Object Forever [DELETE]
This microservice operation will remove an object from the trash and delete it forever.  

+ Request
    The JSON object in the request body should contain a change token:
    + changeToken (string, required) - A hash value expected to match the targeted object's current changeToken value.  This value should be set with the same value returned from any list operation or the Get Object operation.

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

# Group Access Control Operations

---

## Object Share [/shared/{objectId}]

Share for an object may be added or removed at the URI designated  

+ Parameters
    + objectId (string, required) - string Hex encoded identifier of the object for which a share will be added or removed.

### Add Object Share [POST]
This microservice operation is used to grant the specified permission on the target object to the grantee, as well as record that the share has been established, and optionally perform propagated creation of the same permissions to existing children. Regardless of sharing settings, as with standard object permissions, a user is still required to pass all other checks to be able to access objects.

+ Request

    The JSON object in the request body may contain the following fields:
    + share (object, required) - Nested object representing the targets of the share in the same format as would be presented in an ACM.
    + allowCreate (boolean, optional) - Denotes whether the grantee will have permission to create new objects as children of this object. Defaults to false if not specified.
    + allowRead (boolean, optional) - Denotes whether the grantee will have permission to read the object. Defaults to false if not specified.
    + allowUpdate (boolean, optional) - Denotes whether the grantee will have permission to modify the object, but not delete it or change sharing settings. Defaults to false if not specified.
    + allowDelete (boolean, optional) - Denotes whether the grantee will have permission to delete an object and its children. Defaults to false if not specified.
    + allowShare (boolean, optional) - Denotes whether the grantee will have permission to share this object, and its children, limited to the same permissions this grantee has been given. Defaults to false if not specified.
    + propagateToChildren (boolean, optional) - Denotes whether this grant will be applied recursively to all existing children of the referenced object. Defaults to false if not specified.

    + Headers
    
            Host: dockervm:4430
            User-Agent: Go-http-client/1.1
            Content-Length: 206
            Content-Type: application/json
            Accept-Encoding: gzip

    + Body
    
            {
                "share": {share},
                "allowCreate": {allowCreate},
                "allowRead": {allowRead},
                "allowUpdate": {allowUpdate},
                "allowDelete": {allowDelete},
                "allowShare": {allowShare},
                "propagateToChildren": {propagateToChildren}
            }

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

### Remove Object Share [DELETE]
This microservice operation removes a previously defined object share.

+ Request

    The JSON object in the request body should contain the following fields
    + share (object, required) - Nested object representing the targets of the permissions to revoke in the same format as would be presented in an ACM.
    + revokeCreate (boolean, optional) - Denotes whether to revoke permission to create child objects from those targets in the share. Defaults to false if not specified.
    + revokeRead (boolean, optional) - Denotes whether to revoke permission to read the object from those targets in the share. Defaults to false if not specified.
    + revokeUpdate (boolean, optional) - Denotes whether to revoke permission to update the object from those targets in the share. Defaults to false if not specified.
    + revokeDelete (boolean, optional) - Denotes whether to revoke permission to delete the object from those targets in the share. Defaults to false if not specified.
    + revokeShare (boolean, optional) - Denotes whether to revoke permission to share the object from those targets in the share. Defaults to false if not specified.
    + propagateToChildren (boolean, optional) - Denotes whether this revokation will be applied recursively to all existing children of the referenced object. Defaults to false if not specified.

    + Headers
    
            Content-Type: application/json
            Content-Length: nnn

    + Body
    
            {
                "share": {share},
                "revokeCreate": {revokeCreate},
                "revokeRead": {revokeRead},
                "revokeUpdate": {revokeUpdate},
                "revokeDelete": {revokeDelete},
                "revokeShare": {revokeShare},
                "propagateToChildren": {propagateToChildren}
            }

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

# Group Versioning Operations

---

## List Object Revisions [/revisions/{objectId}?pageNumber={pageNumber}&pageSize={pageSize}&sortField={sortField}&sortAscending={sortAscending}]

+ Parameters
    + objectId - Hex encoded identifier of the object for which revisions are being requested.
    + pageNumber (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize (number, optional) - The number of results to return per page.
    + sortField (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
    + sortAscending (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.

### List Object Revisions [GET]

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
    + objectId (string, required) - Hex encoded identifier of the object to be retrieved.
    + revisionId (number, required) - The revision number to be retrieved. 

### Get Object Stream Revision [GET]

+ Response 200

    + Headers

              Content-Length: 26
              Cache-Control: no-cache
              Connection: keep-alive
              Content-Type: text
              Date: Wed, 24 Feb 2016 23:34:20 GMT
              Expires: Wed, 24 Feb 2016 23:34:19 GMT
              Server: nginx/1.8.1

    + Body
    
              The bytes representing the content stream of the requested object revision 

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

# Group Search Operations

---

## Search [/search/{searchPhrase}?pageNumber={pageNumber}&pageSize={pageSize}&sortField={sortField}&sortAscending={sortAscending}&filterField={filterField}&condition={condition}&expression={expression}]

+ Parameters
    + searchPhrase (string, required) - The phrase to look for inclusion within the name or description of objects. This will be overridden if parameters for filterField are set.
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
  
+ Response 400

        Malformed request.
        
+ Response 500

        * Error retrieving object
        * Error determining user.

# Group Filing Operations

---

## Move Object [/objects/{objectId}/move/{folderId}]

+ Parameters
    + objectId (string, required) - Hex encoded identifier of the object to be moved.
    + folderId (string, optional) - Hex encoded identifier of the folder into which this object should be moved.  If no identifier is provided, then the object will be moved to the owner's root folder.

### Move Object [POST]
This microservice operation supports moving an object such as a file or folder from one location to another. By default, all objects are created in the ‘root’ as they have no parent folder given.

This creates a new revision of the object.

+ Request

    The JSON object in the request body should contain a change token:
    + changeToken (string, required) - A hash value expected to match the targeted object's current changeToken value.  This value should be set with the same value returned from any list operation or the Get Object operation.

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

## List User Objects Shared [/shared?pageNumber={pageNumber}&pageSize={pageSize}&sortField={sortField}&sortAscending={sortAscending}&filterField={filterField}&condition={condition}&expression={expression}]

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

### List User Objects Shared [GET]
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
        

## List Trashed Objects [/trashed?pageNumber={pageNumber}&pageSize={pageSize}&sortField={sortField}&sortAscending={sortAscending}&filterField={filterField}&condition={condition}&expression={expression}]

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

### List Trashed Objects [GET]

+ Response 200
    + Attributes (ObjectResultset)

+ Response 400

        Unable to decode request
        
+ Response 500

        Error storing metadata or stream

## Undelete Object [/objects/{objectId}/untrash]

This operation restores a previously deleted object from the trash. Recursively, children of the previously deleted object will also be restored.

This creates a new revision of the object.

+ Parameters
     + objectId (string, required) - Hex encoded identifier of the object to be deleted.

### Undelete Object [POST]

+ Request

    The JSON object in the request body should contain a change token:
    + changeToken (string, required) - A hash value expected to match the targeted object's current changeToken value.  This value should be set with the same value returned from any list operation or the Get Object operation.

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

## User Stats [/userstats]

User Stats provides metrics information for the user's total number of objects and revisions and the amount of size consumed in the system.

### User Stats [GET]

+ Response 200
    + Attributes (UserStats)
    
+ Response 500

        Internal Server Error


# Data Structures

## ChangeToken (object)

+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.

## ObjectResp (object)

+ id: `11e5e4867a6e3d8389020242ac110002`  (string, required) - The unique identifier of the object hex encoded to a string. This value can be used for alterations and listing on other RESTful methods.
+ createdDate: `2016-03-07T17:03:13Z`  (string) - The date and time the object was created in the system in UTC ISO 8601 format.
+ createdBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that created the object.
+ modifiedDate: `2016-03-07T17:03:13Z` (string) -  The date and time the object was last modified in the system in UTC ISO 8601 format. For unchanged objects, this will reflect the same value as the createdDate field.
+ modifiedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that last modified this object. For unchanged objects, this will reflect the same value as the createdBy field.
+ changeCount: 42 (number) - The total count of changes that have been made to this object over its lifespan. Synonymous with version number. For unchanged objects, this will always be 0.
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.
+ ownedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that owns the object.
+ typeId: `11e5e48664f5d8c789020242ac110002` (string) - The unique identifier of the type assigned this object hex encoded to a string.
+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `gettysburgaddress.txt` (string) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
+ description: `Description here` (string) - An abstract of the object's purpose.
+ parentId: ` ` (string, optional) - The unique identifier of the objects parent hex encoded to a string. This may be used to traverse up the tree. For objects stored at the root of a user, this value will be null.
+ acm: ACM (ACM, required) - The acm value associated with this object in object form
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value will be null.
+ contentSize: 1511 (string) - The length of the object's content stream, if present. For objects without a content stream, this value will be 0.
+ properties: Property (array[Property]) - Array of custom properties associated with the object.
+ permissions: Permission (array[Permission]) - Array of permissions associated with this object.
+ isPDFAvailable: `false` (boolean) - Indicates if a PDF rendition is available for this object.
+ isUSPersonsData: `false` (boolean) - Indicates if this object contains US Persons data.
+ isFOIAExempt: `false` (boolean) - Indicates if this object is exempt from Freedom of Information Act requests.

## ACM (object)

+ version: `2.1.0` (string, required) - The version of this acm `{"version":"2.1.0","classif":"U"}` 
+ classif: `U` (string, required) - The portion marked classification for this ACM
+ dissem_countries: `USA` (array[string], required) - The trigraphs of countries for which this ACM can be read

## ObjectRespDeleted (object)

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
+ acm: ACM (ACM, required) - The acm value associated with this object in object form
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value will be null.
+ contentSize: 1511 (string) - The length of the object's content stream, if present. For objects without a content stream, this value will be 0.
+ properties: Property (array[Property]) - Array of custom properties associated with the object.
+ permissions: Permission (array[Permission]) - Array of permissions associated with this object.
+ isPDFAvailable: `false` (boolean) - Indicates if a PDF rendition is available for this object.
+ isUSPersonsData: `false` (boolean) - Indicates if this object contains US Persons data.
+ isFOIAExempt: `false` (boolean) - Indicates if this object is exempt from Freedom of Information Act requests.

## ObjectResultset (object)

+ totalRows: 100 (number) - Total number of items matching the query.
+ pageCount: 10 (number) - Total rows divided by page size.
+ pageNumber: 1 (number) - Requested page number for this resultset.
+ pageSize: 10 (number) - Requested page size for this resultset.
+ pageRows: 10 (number) - Number of items included in this page of the results, which may be less than pagesize, but never greater.
+ objects: ObjectResp (array[ObjectResp])

## ObjectStorageMetric (object)

+ typeName: `File` (string) - The type of object, which is usually File or Folder.
+ objects: 24 (number) - The number of current objects that are stored.
+ objectsWithRevision: 432 (number) - The number of versioned objects that are stored.
+ objectsSize: 249234 (number) - The total size of objects in bytes, which could be a very large number.
+ objectsWithRevisionSize: 23478234 (number) - The total size of versioned objects in bytes, which may be very large.

## Permission (object)

+ id: `11e5e4867a6e3d8389020242ac110002` (string) -  The unique identifier of the permission associated to the object hex encoded to a string. This value can be used for deleting the permission.
+ createdDate: `2016-03-07T17:03:13Z` (string) -  The date and time the permission was created in the system in UTC ISO 8601 format.
+ createdBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) -  The user that created the permission.
+ modifiedDate: `2016-03-07T17:03:13Z` (string) -  The date and time the permission was last modified in the system in UTC ISO 8601 format.
+ modifiedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) -  The user that last modified this permission.
+ changeCount: 2 (number) - number The total count of changes that have been made to this permission over its lifespan. Synonymous with version number.
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) -  A hash of the permissions's unique identifier and last modification date and time.
+ objectId: `11e5e4867a6e3d8389020242ac110002` (string) -  The unique identifier of the object that this permission is associated with.
+ grantee: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) -  The user for whom this permission is granted to
+ allowCreate: false (boolean) -  Indicates whether the grantee can create child objects under the referenced object of this permission.
+ allowRead: true (boolean) -  Indicates whether the grantee can view the object referenced by this permission.
+ allowUpdate: false (boolean) -  Indicates whether the grantee can modify the object referenced by this permission.
+ allowDelete: false (boolean) -  Indicates whether the grantee can delete the object referenced by this permission.
+ allowShare: false (boolean) -  Indicates whether the grantee can reshare the object referenced by this permission.
+ explicitShare: false (boolean) - Indicates whether this permission was explicitly created by a call to add the grant, or if it was inherited from the parent object for which a child was made or propagated creation on an existing object.

## Property (object)

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

## UserStats (object)

+ totalObjects: 24 (number) - The number of current objects that are stored.
+ totalObjectsWithRevision: 432 (number) - The number of versioned objects that are stored.
+ totalObjectsSize: 249234 (number) - The total size of objects in bytes, which could be a very large number.
+ totalObjectsWithRevisionSize: 23478234 (number) - The total size of versioned objects in bytes, which may be very large.
+ objectStorageMetrics: ObjectStorageMetric (array[ObjectStorageMetric]) - An array of ObjectStorageMetrics denoting the type of object, quantity of base object and revisions, and size used by base object and revision.
