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
| Update Object Stream | Used for updating the content stream and metadata of an object. |
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
| List Objects Shared to Everyone | Retrieves a resultset of objects that are shared to everyone. |
| List Trashed Objects | Retrieves a resultset of objects in the user's trash. |
| Undelete Object | Restores an object from the user's trash. |
| User Stats | Retrieve information for user's storage consumtpion. |
| Zip Files | Get a zip of some files. |


##  Reference Examples

Detailed code examples that use the API:

[Java Caller (create an object)](static/templates/ObjectDriveSDK.java)

[Javascript Caller (our simple test user interface)](static/templates/listObjects.js)

The http level result of calling APIs that happens inside of SSL:

[Actual Traffic - Basic Operations](static/templates/APISample.html)

<!--
[Share (from real traffic)](static/templates/TestShare.html)
-->
Testing interface: 

[Development (for development)](ui)

[Drive UI](/apps/drive/home)

# Group CRUD Object Operations
These basic operations provide support for creating, retrieving, updating and deleting objects. 

---

## Create an Object [/objects]

### Create an Object [POST]
Create a new object in Object Drive.

The returned json is the metadata that can be used for further operations on the data, such as update,
delete, etc.  The json representing an object is uniform so that it is a similar representation when
it comes back from creation, or from getting an object listing, or from an update.
An ACM follows guidance given here: https://confluence.363-283.io/pages/viewpage.action?ageId=557850

+ Request With Content Stream (multipart/form-data; boundary=7518615725)
    When creating a new object with a content stream, such as a file, this must be presented in multipart/form-data format, with the metadata about the object provided in a field named 'ObjectMetadata' containing a JSON structure of the following fields.

    + typeName (string, required) -  The type to be assigned to this object. Common types include 'File', 'Folder'. Custom types may be referenecd here for the purposes of rules or processing constraints and definition of properties.
    + name (string, optional) - The name to be given this object.  If no name is given, then objects are created with the default name pattern of `New <typeName>`.
    + description (string, optional) - An optional abstract of the object's contents.
    + parentId (string, optional) - Hex encoded identifier of an object, typically a folder, into which this new object is being created as a child object. If no value is specified, then the object will be created in the root location of the user who is creating it.
    + acm (object, required) - Access Control Model is the security model leveraged by the system when enforcing access control. It is based on the ISM, NTK, ACCM and Share standards, requirements and policies.  This value may be provided in either serialized string format, or nested object format.
    + contentType (string, optional) - The suggested mime type for the content stream if given for this object.
    + contentSize (int, optional) - The length of the content stream, in bytes. If there is no content stream, this value should be 0.
    + properties (properties array, optional) - Array of custom properties to be associated with the newly created object.
    + permission (ObjectShare array, optional) - Array of additional permissions to be associated with this object when created. By default, the owner is granted full access. This structure is used when permissions need to be granted beyond read access, since read access is most easily conveyed in the ACM share itself.  For convenience, the structure of the share within a permission is the same as that used for ACMs.  The sample below would grant the creator (as owner) full CRUDS, everyone would get read, and then the user `cn=user,ou=org,c=us` would get full CRUDS, while members of the groups ODrive_G1 and ODrive_G2 would get create, read and update. 
    + containsUSPersonsData (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
    + exemptFromFOIA (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.

    + Body
    
            --7518615725
            Content-Disposition: form-data; name="ObjectMetadata"
            Content-Type: application/json
            {
                "typeName": "File",
                "name": "My new file",
                "description": "This is the description for my file",
                "parentId": "",
                "acm": {
                    "classif": "u",
                    "version": "2.1.0"
                },
                "contentType": "text/plain",
                "contentSize": 31,
                "properties": [],
                "permissions": [
                    {
                        "share": {
                            "users": [
                                "cn=user,ou=org,c=us"
                            ]
                        },
                        "allowCreate": true,
                        "allowRead": true,
                        "allowUpdate": true,
                        "allowDelete": true,
                        "allowShare": true
                    },
                    {
                        "share": {
                            "projects": {
                                "dctc" : {
                                    "disp_nm": "DCTC",
                                    "groups": [
                                        "ODrive_G1",
                                        "ODrive_G2"
                                    ]
                               }
                            }
                        },
                        "allowCreate": true,
                        "allowRead": true,
                        "allowUpdate": true,
                        "allowDelete": false,
                        "allowShare": false
                    }
                ],
                "containsUSPersonsData": "No",
                "exemptFromFOIA": "No"
            }
            --7518615725
            Content-Disposition: form-data; name="filestream"; filename="test.txt"
            Content-Type: application/octet-stream
            
            This is the content of the file
            
            --7518615725--        

+ Request Without a Content Stream (application/json)
    When creating a new object without a content stream, such as a folder, the object definition may be specified directly in the request body as typified below.

    + typeName (string, required) -  The type to be assigned to this object. Common types include 'File', 'Folder'. Custom types may be referenecd here for the purposes of rules or processing constraints and definition of properties.
    + name (string, optional) - The name to be given this object.  If no name is given, then objects are created with the default name pattern of `New <typeName>`.
    + description (string, optional) - An optional abstract of the object's contents.
    + parentId (string, optional) - Hex encoded identifier of an object, typically a folder, into which this new object is being created as a child object. If no value is specified, then the object will be created in the root location of the user who is creating it.
    + acm (object, required) - Access Control Model is the security model leveraged by the system when enforcing access control. It is based on the ISM, NTK, ACCM and Share standards, requirements and policies.  This value may be provided in either serialized string format, or nested object format.
    + contentType (string, optional) - The suggested mime type for the content stream if given for this object.
    + contentSize (int, optional) - The length of the content stream, in bytes. If there is no content stream, this value should be 0.
    + properties (properties array, optional) - Array of custom properties to be associated with the newly created object.
    + permission (ObjectShare array, optional) - Array of additional permissions to be associated with this object when created. By default, the owner is granted full access. This structure is used when permissions need to be granted beyond read access, since read access is most easily conveyed in the ACM share itself.  For convenience, the structure of the share within a permission is the same as that used for ACMs.
    + containsUSPersonsData (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
    + exemptFromFOIA (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.

    + Attributes (CreateObjectRequestNoStream)


+ Response 200 (application/json)
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

+ Response 200 (application/json)
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

    + changeToken (string, required) - The current change token on the object
    + typeName (string, optional) - The display name of the type assigned this object.
    + name (string, optional) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
    + description (string, optional) - The new description to be given as an abstract of the objects content stream. If no value is provided, or this field is ommitted, then the description will not be changed.
    + acm (object, optional) -  Access Control Model (ACM) is the security model leveraged by the system when enforcing access control. It is based on the ISM, NTK, ACCM and Share standards, requirements and policies. https://confluence.363-283.io/pages/viewpage.action?pageId=557850. If no value is provided, or this field is ommitted, then the acm will not be changed. This value may be provided in either serialized string format, or nested object format.
    + properties (properties array, optional) -  An array of custom properties to be associated with this object for property changes. For the properties specified, those who do not match existing properties on the object by name will be added. For the properties that do match existing properties by name, if the value specified is blank or empty, then the existing property will be deleted, otherwise, the property will be updated to the new value. If properties are specified in the array, then existing properties on the object are retained. Properties are only removed from an object if they are provided, with their value set to an empty string.
    + containsUSPersonsData (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
    + exemptFromFOIA (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.

    + Attributes (UpdateObject)

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

## Get Object Stream [/objects/{objectId}/stream?disposition={disposition}]

The content stream for an object may be retrieved or updated at the URI designated.

+ Parameters
    + objectId (string, required) - Hex encoded identifier of the object to be retrieved.
    + disposition (string, optional) - The major content-disposition type, defaults to "inline", exists to set to "attachment"
  
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

+ Response 304

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

## Update Object Stream [/objects/{objectId}/stream]

Updates the actual file bytes associated with an objectId. This must be provided in multipart/form-data format, with the metadata about the object provided in a field named 'ObjectMetadata'.

+ Parameters
    + objectId (string, required) - Hex encoded identifier of the object to be retrieved.

### Update an Object Stream [POST]

This creates a new revision of the object.
    
+ Request (multipart/form-data; boundary=b428e6cd1933)

    The JSON object provided in the body can contain the following fields:

    + id (string, required) - The unique identifier of the object hex encoded to a string. This value must match the objectId provided in the URI.
    + changeToken (string, required) - A hash value expected to match the targeted object’s current changeToken value. This value is retrieved from get or list operations.
    + typeName (string, optional) -  The new type to be assigned to this object. Common types include 'File', 'Folder'. If no value is provided or this field is omitted, then the type will not be changed.
    + name (string, optional) - The new name to be given this object. It does not have to be unique. It may refer to a conventional filename and extension. If no value is provided, or this field is ommitted, then the name will not be changed.
    + description (string, optional) - The new description to be given as an abstract of the objects content stream. If no value is provided, or this field is ommitted, then the description will not be changed.
    + acm (string OR object, optional) -  Access Control Model (ACM) is the security model leveraged by the system when enforcing access control. It is based on the ISM, NTK, ACCM and Share standards, requirements and policies. https://confluence.363-283.io/pages/viewpage.action?pageId=557850. If no value is provided, or this field is ommitted, then the acm will not be changed.  This value may be provided in either serialized string format, or nested object format.
    + contentType (string, optional) - The suggested mime type for the content stream if given for this object.
    + contentSize (int, optional) - The length of the content stream, in bytes. If there is no content stream, this value should be 0.
    + properties (properties array, optional) -  An array of custom properties to be associated with this object for property changes. For the properties specified, those who do not match existing properties on the object by name will be added. For the properties that do match existing properties by name, if the value specified is blank or empty, then the existing property will be deleted, otherwise, the property will be updated to the new value. If properties are specified in the array, then existing properties on the object are retained. Properties are only removed from an object if they are provided, with their value set to an empty string.    
    + containsUSPersonsData (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
    + exemptFromFOIA (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.
           
    + Body

            --b428e6cd1933
            Content-Disposition: form-data; name="ObjectMetadata"
            Content-Type: application/json
            {
                "id": "11e5e4867a6e3d8389020242ac110002", 
                "changeToken": "65eea405306ed436d18b8b1c0b0b2cd3",
                "typeName": "File",
                "name": "My new file",
                "description": "This is the new description for my file",
                "acm": {
                    "classif": "u",
                    "version": "2.1.0"
                },
                "contentType": "text/plain",
                "contentSize": 36,
                "properties": [],
                "containsUSPersonsData": "No",
                "exemptFromFOIA": "No"
            }
            --b428e6cd1933
            Content-Disposition: form-data; name="filestream"; filename="test.txt"
            Content-Type: application/octet-stream
            
            This is the new contents of the file
            
            --b428e6cd1933--

+ Response 200 (application/json)

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
           
+ Request (application/json)

    + Attributes (ChangeToken)
            
+ Response 200 (application/json)

    + Attributes (ObjectDeleted)

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

+ Request (application/json)

    + Attributes (ChangeToken)

+ Response 200 (application/json)

    + Attributes (ObjectExpunged)

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

These operations permit granting and revoking capabilities on objects beyond read/view access. Capabilities are defined as follows:

* Create - The ability to create child objects beneath the object this grant is given.
* Read - The ability to read/view an object's properties, it's stream, and list its children.
* Update - The ability to make alterations to an object, including its ACM, except the share portions of the ACM.
* Delete - The ability to delete, undelete, and expunge (delete forever) an object this grant is given.
* Share - The ability to grant/revoke capabilities on objects to users and groups.

---

## Object Share [/shared/{objectId}]

Share for an object may be added or removed at the URI designated  

+ Parameters
    + objectId (string, required) - string Hex encoded identifier of the object for which a share will be added or removed.

### Add Object Share [POST]
This microservice operation is used to grant the specified permission on the target object to the grantee, as well as record that the share has been established. Regardless of sharing settings, as with standard object permissions, a user is still required to pass all other checks to be able to access objects.

+ Request (application/json)

    + Attributes (ObjectShare)

+ Response 200 (application/json)

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
This microservice operation removes permissions previously granted to users and groups for a given object.

+ Request (application/json)

    + Attributes (ACMShare)

+ Response 200 (application/json)

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

## List Objects Shared to Everyone [/sharedpublic?pageNumber={pageNumber}&pageSize={pageSize}&sortField={sortField}&sortAscending={sortAscending}&filterField={filterField}&condition={condition}&expression={expression}]

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
    
### List Objects Shared to Everyone [GET]
This microservice operation retrieves a list of objects that are shared to everyone.

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


## Get Object Stream Revision [/revisions/{objectId}/{revisionId}/stream?disposition={disposition}]

+ Parameters
    + objectId (string, required) - Hex encoded identifier of the object to be retrieved.
    + revisionId (number, required) - The revision number to be retrieved. 
    + disposition (number, optional) - The main Content-Disposition.  Defaults to "inline", exists to be set to "attachment".

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

+ Response 200 (application/json)
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

+ Request (application/json)

    The JSON object in the request body should contain a change token:

    + Attributes (ChangeToken)

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

+ Request (application/json)    

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

+ Request (application/json)

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

+ Request (application/json)

+ Response 200 (application/json)
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

+ Request (application/json)

    The JSON object in the request body should contain a change token:

    + Attributes (ChangeToken)

+ Response 200 (application/json)
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

+ Request (application/json)

+ Response 200 (application/json)
    + Attributes (UserStats)
    
+ Response 500

        Internal Server Error


# Data Structures

## ACM (object)

+ classif: `U` (string, required) - The abbreviated classification for this ACM.
+ dissem_countries: `USA` (array[string], required) - The trigraphs of countries for which this ACM can be read
+ share (ACMShare, optional) - The users and project/groups that will be granted read access to this object. If no share is specified, then the object is public.
+ version: `2.1.0` (string, required) - The version of this acm `{"version":"2.1.0","classif":"U"}` 

## ACMResponse (object)

+ banner: `UNCLASSIFIED` (string, required) - The banner marking of the overall classification.
+ classif: `U` (string, required) - The abbreviated classification for this ACM.
+ dissem_countries: `USA` (array[string], required) - The trigraphs of countries for which this ACM can be read.
+ f_accms (array, optional) - The flattened value of the ACCMs.
+ f_atom_energy (array, optional) - The flattened value of the atom energy.
+ f_clearance: `u` (string, required) - The flattened value of the classification.
+ f_macs (array, optional) - The flattened value of the MACs.
+ f_missions (array, optional) - The flattened value of the missions.
+ f_oc_org (array, optional) - The flattened value of the OC Organizations.
+ f_sci_ctrls (array, optional) - The flattened values of the SCI controls.
+ f_regions (array, optional) - The flattened values of the Regions.
+ f_share: `x`, `y`, `z` (array, optional) - The flattened value of the shares.
+ portion: `U` (string, required) - THe portion marked classification for this ACM.
+ share (ACMShare, optional) - The users and project/groups that will be granted read access to this object. If no share is specified, then the object is public.
+ version: `2.1.0` (string, required) - The version of this acm `{"version":"2.1.0","classif":"U"}` 



## ACMShare (object)

+ users: `CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US`, `CN=test tester02,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US`, `CN=test tester03,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US`, `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (array[string], optional) - Array of distinguished names for users that are targets of this share.
+ projects (array[ACMShareProjects], optional) - Array of projects with nested groups that are targets of this share.

## ACMShareProjects (object)

+ ukpn (ACMShareProject, required) - A unique keyed project name (ukpn) is specified as the fieldname for the project with display name and groups contained in object value.
+ ukpn2 (ACMShareProject2, required) - A unique keyed project name (ukpn) is specified as the fieldname for the project with display name and groups contained in object value.

## ACMShareProject (object)

+ disp_nm: `Project Name` (string, required) - The display name for the project
+ groups: `Group Name`, `Cats`, `Dogs` (array[string], required) - Array of groups to be targetted by this share within the project.

## ACMShareProject2 (object)

+ disp_nm: `Project Name 2` (string, required) - The display name for the project. This sample is `Project Name 2` for `uniquekeyprojectname2`
+ groups: `Group 1`, `Group 2`, `Group 3` (array[string], required) - Array of groups to be targetted by this share within the project.

## ACMShareCreateSample (object)

+ users: `CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (array[string], optional) - Array of distinguished names for users that are targets of this share.
+ projects (array[ACMShareCreateSampleProjectsDCTC], optional) - Array of projects with nested groups that are targets of this share.

## ACMShareCreateSampleProjectsDCTC (object)

+ dctc (ACMShareCreateSampleProjectDCTCODriveG1, required) - A unique keyed project name (dctc) is specified as the fieldname for the project with display name and groups contained in object value.

## ACMShareCreateSampleProjectDCTCODriveG1 (object)

+ disp_nm: `DCTC` (string, required) - The display name for the project
+ groups: `ODrive_G1` (array[string], required) - Array of groups to be targetted by this share within the project.


## CallerPermission (object)

+ allowCreate: false (boolean) -  Indicates whether the caller can create child objects under this object.
+ allowRead: true (boolean) -  Indicates whether the caller can view this object.
+ allowUpdate: false (boolean) -  Indicates whether the caller can modify this object.
+ allowDelete: false (boolean) -  Indicates whether the caller can delete this object.
+ allowShare: false (boolean) -  Indicates whether the caller can reshare this object.

## ChangeToken (object)

+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.

## CreateObjectRequest (object)

+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `gettysburgaddress.txt` (string) - The name for this object. 
+ description: `Description here` (string) - An abstract of the object's purpose.
+ parentId: ` ` (string, optional) - The unique identifier of the objects parent hex encoded to a string.  An empty value will result in this object being created in the user's root folder.
+ acm (ACM, required) - The acm value associated with this object in object form
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value should be empty.
+ contentSize: 1511 (string) - The length of the object's content stream, if present. For objects without a content stream, this value should be 0.
+ properties (array[PropertyCreate]) - Array of custom properties to be associated with the object.
+ permissions (array[PermissionCreate]) - Array of permissions to be associated with this object.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.


## CreateObjectRequestNoStream (object)

+ typeName: `Folder` (string) - The display name of the type assigned this object.
+ name: `Famous Speeches` (string) - The name for this object. 
+ description: `Description here` (string) - An abstract of the object's purpose.
+ parentId: ` ` (string, optional) - The unique identifier of the objects parent hex encoded to a string.  An empty value will result in this object being created in the user's root folder.
+ acm (ACM, required) - The acm value associated with this object in object form
+ contentType: ` ` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value should be empty.
+ contentSize: 0 (string) - The length of the object's content stream, if present. For objects without a content stream, this value should be 0.
+ properties (array[PropertyCreate]) - Array of custom properties to be associated with the object.
+ permissions (array[PermissionCreate]) - Array of permissions to be associated with this object.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.

## ObjectDeleted (object)

+ deletedDate: `2016-03-07T17:03:13Z` (string, optional) -  The date and time the object was deleted in the system in UTC ISO 8601 format. This field is only present if the object is in the trash.

## ObjectExpunged (object)

+ expungedDate: `2016-03-07T17:03:13Z` (string, optional) -  The date and time the object was expunged from the system in UTC ISO 8601 format. 

## ObjectResp (object)

+ id: `11e5e4867a6e3d8389020242ac110002`  (string, required) - The unique identifier of the object hex encoded to a string. This value can be used for alterations and listing on other RESTful methods.
+ createdDate: `2016-03-07T17:03:13Z`  (string) - The date and time the object was created in the system in UTC ISO 8601 format.
+ createdBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that created the object.
+ modifiedDate: `2016-03-07T17:03:13Z` (string) -  The date and time the object was last modified in the system in UTC ISO 8601 format. For unchanged objects, this will reflect the same value as the createdDate field.
+ modifiedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that last modified this object. For unchanged objects, this will reflect the same value as the createdBy field.
+ deletedDate: `0001-01-01T00:00:00Z` (string, optional) -  The date and time the object was deleted in the system in UTC ISO 8601 format. This field is only populated if the object is in the trash.
+ deletedBy: `` (string) - The user that deleted the object. This field is only populated if the object is in the trash.
+ changeCount: 42 (number) - The total count of changes that have been made to this object over its lifespan. Synonymous with version number. For unchanged objects, this will always be 0.
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.
+ ownedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that owns the object.
+ typeId: `11e5e48664f5d8c789020242ac110002` (string) - The unique identifier of the type assigned this object hex encoded to a string.
+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `gettysburgaddress.txt` (string) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
+ description: `Description here` (string) - An abstract of the object's purpose.
+ parentId: ` ` (string, optional) - The unique identifier of the objects parent hex encoded to a string. This may be used to traverse up the tree. For objects stored at the root of a user, this value will be null.
+ acm (ACMResponse, required) - The acm value associated with this object in object form
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value will be null.
+ contentSize: 1511 (string) - The length of the object's content stream, if present. For objects without a content stream, this value will be 0.
+ isPDFAvailable: `false` (boolean) - Indicates if a PDF rendition is available for this object.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.
+ properties (array[Property]) - Array of custom properties associated with the object.
+ callerPermissions (CallerPermission) - Permissions granted to the caller that resulted in this object being returned.
+ permissions (array[PermissionUser,PermissionGroup]) - Array of permissions associated with this object.

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
+ acm (ACMResponse, required) - The acm value associated with this object in object form
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value will be null.
+ contentSize: 1511 (string) - The length of the object's content stream, if present. For objects without a content stream, this value will be 0.
+ isPDFAvailable: `false` (boolean) - Indicates if a PDF rendition is available for this object.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.
+ properties (array[Property]) - Array of custom properties associated with the object.
+ callerPermissions (CallerPermission) - Permissions granted to the caller that resulted in this object being returned.
+ permissions (array[PermissionUser,PermissionGroup]) - Array of permissions associated with this object.

## ObjectResultset (object)

+ totalRows: 100 (number) - Total number of items matching the query.
+ pageCount: 10 (number) - Total rows divided by page size.
+ pageNumber: 1 (number) - Requested page number for this resultset.
+ pageSize: 10 (number) - Requested page size for this resultset.
+ pageRows: 10 (number) - Number of items included in this page of the results, which may be less than pagesize, but never greater.
+ objects (array[ObjectResp]) - Array containing objects for this page of the resultset.

## ObjectShare (object)

+ share (ACMShare, optional) - The users and project/groups that will be granted read access to this object. If no share is specified, then the object is public.
+ allowCreate: `false` (boolean, optional) -  Indicates whether the targets for the share can create child objects under the referenced object.
+ allowRead: `true` (boolean, optional) -  Indicates whether the targets for the share can view the object referenced by this permission.
+ allowUpdate: `false` (boolean, optional) -  Indicates whether the targets for the share can modify the object referenced by this permission.
+ allowDelete: `false` (boolean, optional) -  Indicates whether the targets for the share can delete the object referenced by this permission.
+ allowShare: `false` (boolean, optional) -  Indicates whether the targets for the share can reshare the object referenced by this permission.

## ObjectStorageMetric (object)

+ typeName: `File` (string) - The type of object, which is usually File or Folder.
+ objects: 24 (number) - The number of current objects that are stored.
+ objectsWithRevision: 432 (number) - The number of versioned objects that are stored.
+ objectsSize: 249234 (number) - The total size of objects in bytes, which could be a very large number.
+ objectsWithRevisionSize: 23478234 (number) - The total size of versioned objects in bytes, which may be very large.

## PermissionUser (object)

+ grantee: `cntesttester10oupeopleoudaeouchimeraou_s_governmentcus` (string) -  The flattened form of the user or group this permission targets
+ userDistinguishedName: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user for whom this permission is granted to
+ displayName: `test tester10` (string) - A representation of the grantee suitable for display in user interfaces
+ allowCreate: true (boolean) -  Indicates whether the grantee can create child objects under the referenced object of this permission.
+ allowRead: true (boolean) -  Indicates whether the grantee can view the object referenced by this permission.
+ allowUpdate: true (boolean) -  Indicates whether the grantee can modify the object referenced by this permission.
+ allowDelete: true (boolean) -  Indicates whether the grantee can delete the object referenced by this permission.
+ allowShare: true (boolean) -  Indicates whether the grantee can reshare the object referenced by this permission.

## PermissionGroup (object)

+ grantee: `dctc_odrive_g1` (string) -  The flattened form of the user or group this permission targets
+ projectName: `dctc` (string) - The project name which is also the key of a project object provided in a share.
+ projectDisplayName: `DCTC` (string) - The project display name for a project group share.
+ groupName: `ODrive_G1` (string) - The group name for a project group share.
+ displayName: `DCTC ODrive_G1` (string) - A representation of the grantee suitable for display in user interfaces
+ allowCreate: true (boolean) -  Indicates whether the grantee can create child objects under the referenced object of this permission.
+ allowRead: true (boolean) -  Indicates whether the grantee can view the object referenced by this permission.
+ allowUpdate: true (boolean) -  Indicates whether the grantee can modify the object referenced by this permission.
+ allowDelete: true (boolean) -  Indicates whether the grantee can delete the object referenced by this permission.
+ allowShare: true (boolean) -  Indicates whether the grantee can reshare the object referenced by this permission.

## PermissionCreate (object)

+ share (ACMShareCreateSample) - The share structure for this permission representing one or more targets to be granted the permissions
+ allowCreate: false (boolean) -  Indicates whether the grantee can create child objects under the referenced object of this permission.
+ allowRead: true (boolean) -  Indicates whether the grantee can view the object referenced by this permission.
+ allowUpdate: true (boolean) -  Indicates whether the grantee can modify the object referenced by this permission.
+ allowDelete: false (boolean) -  Indicates whether the grantee can delete the object referenced by this permission.
+ allowShare: false (boolean) -  Indicates whether the grantee can reshare the object referenced by this permission.

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
+ classificationPM: `U` (string) -  The portion mark classification for the value of this property

## PropertyCreate (object)

+ name: `Some Property` (string) - The name, key, field or label given to a property for usability
+ propertyValue: `Some Property Value` (string) -  The value assigned for the property
+ classificationPM: `U//FOUO` (string) -  The portion mark classification for the value of this property

## UpdateObject (object)

+ id: `11e5e4867a6e3d8389020242ac110002` (string, required) - The unique identifier of the object hex encoded to a string. 
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - The current change token on the object
+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `gettysburgaddress.txt` (string) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
+ description: `Description here` (string) - An abstract of the object's purpose.
+ acm (ACM, optional) - The acm value associated with this object in object form. If not provided, the current ACM on the object will be retained.
+ properties (array[Property]) - Array of custom properties associated with the object. New properties will be added. Properties that have the same name as existing properties will be replaced. Those with an empty value will be deleted.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.

## UserStats (object)

+ totalObjects: 24 (number) - The number of current objects that are stored.
+ totalObjectsWithRevision: 432 (number) - The number of versioned objects that are stored.
+ totalObjectsSize: 249234 (number) - The total size of objects in bytes, which could be a very large number.
+ totalObjectsWithRevisionSize: 23478234 (number) - The total size of versioned objects in bytes, which may be very large.
+ objectStorageMetrics: ObjectStorageMetric (array[ObjectStorageMetric]) - An array of ObjectStorageMetrics denoting the type of object, quantity of base object and revisions, and size used by base object and revision.


# Auxillary Operations

## Create Zip of objects [/zip]

+ objectIds (string array, required) - An array of object identifiers of files to be zipped.  
+ fileName (string, optional) - The name to give to the zip file.  Default to "drive.zip".
+ disposition (string, optional) - Either "inline" or "attachment", which is a hint to the browser for handling the result

### Create Zip of objects [POST]

Create a zip of objects from a shopping cart
The UI will accumulate a list of file ID values to include in a zip file.


+ Request (application/json)

    + Body

            {
                "objectIds": [
                        "11e65d7234b1e0668de612754e85c439",
                        "11e65d7234b1d292349612754e85c439",
                        "11e65d7234b1e0a28de612754e85c439",
                        "11e65d7234b2f0668de612754e85c439",
                        "11e65d7234b1ea9c9d9e12754e85c439",
                ],
                "fileName": "drive.zip",
                "disposition": "inline"
            }

+ Response 200

    + Headers

            Content-Type: application/zip
            Content-Disposition: {disposition}; filename={filename}

+ Response 400              

+ Response 500

