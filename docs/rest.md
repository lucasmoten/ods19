FORMAT: 1A

# Object Drive 1.0 

<table style="width:100%;border:0px;padding:0px;border-spacing:0;border-collapse:collapse;font-family:Helvetica;font-size:10pt;vertical-align:center;"><tbody><tr><td style="padding:0px;font-size:10pt;">Version</td><td style="padding:0px;font-size:10pt;">--Version--</td><td style="width:20%;font-size:8pt;"> </td><td style="padding:0px;font-size:10pt;">Build</td><td style="padding:0px;font-size:10pt;">--BuildNumber--</td><td style="width:20%;font-size:8pt;"></td><td style="padding:0px;font-size:10pt;">Date</td><td style="padding:0px;font-size:10pt;">--BuildDate--</td></tr></tbody></table>

# Group Navigation

## Table of Contents

+ [Service Overview](../../)
+ [RESTful API documentation](rest.html)
+ [Emitted Events documentation](events.html)
+ [Environment](environment.html)
+ [Changelog](changelog.html)

# Group RESTful API

## Request Headers

Headers may be provided as part of the request for all API calls.  Those that
are understood by the service are as follows

### Application Logging Headers

* APPLICATION - If this value is set, then it will be assigned in the 
  payload.audit_event.Creator of the global event message. If no value
  is set in this header, the value reported defaults to `Object Drive`
* X-Forwarded-For - If this value is set, then it will be assigned in the
  xForwardedForIp of the global event message

### Cross Origin Resource Sharing (CORS) Headers

The standard headers for CORS are supported.  The CORS policy exposed by the
server is a permissive policy.

### Impersonation Headers

These headers are only used when a request is being proxied or otherwise
authorized to perform requests on behalf of other users.

* EXTERNAL_SYS_DN - If provided, the service expects this value to be in the
  access control list for impersonation.
* USER_DN
* SSL_CLIENT_S_DN - The service expects this to be set by an edge node, match
  the current PKI for the request, and be included in the access control list
  for impersonation.

## Data Type Guidance

Dates are serialized in responses in RFC3339 format. RFC3339 is an ISO 8601 
format where the date and time are shown to at least the second. Portions of a
second are optional and may be given to nanosecond precision. Trailing zeros
in portions of a second are truncated.

An unset date, as is common for deleted date, may be represented as January 1 
in year 1. This will appear as `0001-01-01T00:00:00Z`

Identifiers are hex encoded UUID values with a length of 32 when in string 
format.  These UUID values are generated at the database backing store.  

Boolean values represented as `true` or `false` are stored internally as
a 1 or a 0.  Trinary values, with settings of `Yes`, `No`, and `Unknown`
are treated as strings.

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
    When creating a new object with a content stream, such as a file, this must be presented in multipart/form-data format, with the metadata about the object provided in a field named 'ObjectMetadata' containing a JSON structure of the following fields.  The content stream for the object should be the second part, as the native bytes without use of encoding or character sets.

    + typeName (string, maxlength=255, required) -  The type to be assigned to this object.  Custom types may be referenced here for the purposes of rules or processing constraints and definition of properties.
       * File - This type may be assigned if no type is given, and you are creating an object with a stream
       * Folder - This type may be assigned if no type is given, and you are creating an object without a stream
    + name: `New File` (string, maxlength=255, optional) - The name to be given this object.  If no name is given, then objects are created with the default name pattern of `New <typeName>`.
    + namePathDelimiter: `:::` (string, optional) - An optional alternate path delimiter for which the name given should be assessed to generate intermediate objects when establishing a folder/file structure hierarchy. By default, the name will be split on the record separator (ASCII character 30).
       * Example splitting `abc:::def:::ghi` on `:::`
         * creates object `abc` if it does not already exist
         * creates object `def` if it does not already exist as a child of `abc`
         * creates object `ghi` as a child of `def`
       * Example splitting `abc/def/ghi` on `/`
         * creates object `abc` if it does not already exist
         * creates object `def` if it does not already exist as a child of `abc`
         * creates object `ghi` as a child of `def`
       * Example splitting `(U//FOUO) Turnip Greens` on `/`
         * creates object `(U` if it does not already exist
         * creates object `FOUO) Turnip Greens` as a child of `(U`
       * Example splitting `(U//FOUO) Turnip Greens` with default record separator
         * creates object `(U//FOUO) Turnip Greens`       
    + description (string, maxlength=10240, optional) - An optional abstract of the object's contents.
    + parentId (string, length=32, optional) - Hex encoded identifier of an object, typically a folder, into which this new object is being created as a child object. If no value is specified, then the object will be created in the root location of the user who is creating it.
    + acm (object, required) - Access Control Model is the security model leveraged by the system when enforcing access control. It is based on the ISM, NTK, ACCM and Share standards, requirements and policies.  This value may be provided in either serialized string format, or nested object format.
    + permission (PermissionRequest, optional) - [1.1] The permissions associated with this object by capability and resource allowed.  Resources take the following form:
       * {resourceType}/{serialized-representation}/{optional-display-name}
       * Examples for Users
         * user/{distinguishedName}/{displayName}
         * user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10
       * Examples for groups
         * group/{projectName}/{groupName}
         * group/{projectName}/{projectDisplayName}/{groupName}/{displayName}
         * group/dctc/odrive_g1
         * group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1
         * group/-Everyone
    + contentType: `text/plain` (string, maxlength=255, optional) - The suggested mime type for the content stream if given for this object.
    + contentSize: 0 (number, maxvalue=9223372036854775807, optional) - The length of the content stream, in bytes. If there is no content stream, this value should be 0.  The maxvalue given here is theoretical based upon the maximum allowable value represented in 8 bytes. The actual maximum size of an object is initially constrained by free disk storage space in the local cache on the instance the object is being created.
    + containsUSPersonsData: `Yes` (string, maxlength=255, optional) - Indicates if this object contains US Persons data.
        + Default: `Unknown`  
        + Members
            + `Yes`
            + `No`
            + `Unknown`
    + exemptFromFOIA: `No` (string, maxlength=255, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  
        + Default: `Unknown`  
        + Members
            + `Yes`
            + `No`
            + `Unknown`
    + ownedBy: `user/{distinguishedName}/{displayName}` (string, maxlength=255, optional) - Permits assigning the ownership to a group during create
       * Groups that we are in are allowed
         * group/{projectName}/{projectDisplayName}/{groupName}/{displayName}
    + properties (properties array, optional) - Array of custom properties to be associated with the newly created object.
    + permissions (array[PermissionUserCreate,PermissionGroupCreate]) - **[1.0, Deprecated]** - Array of permissions associated with this object.

    + Body
    
            --7518615725
            Content-Disposition: form-data; name="ObjectMetadata"
            Content-Type: application/json
            
            {
              "typeName": "File",
              "name": "gettysburgaddress.txt",
              "description": "Description here",
              "parentId": "",
              "acm": {
                "classif": "U",
                "dissem_countries": [
                  "USA"
                ],
                "share": {
                  "users": [
                    "CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                    "CN=test tester02,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                    "CN=test tester03,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                    "CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US"
                  ],
                  "projects": [
                    {
                      "ukpn": {
                        "disp_nm": "Project Name",
                        "groups": [
                          "Group Name",
                          "Cats",
                          "Dogs"
                        ]
                      },
                      "ukpn2": {
                        "disp_nm": "Project Name 2",
                        "groups": [
                          "Group 1",
                          "Group 2",
                          "Group 3"
                        ]
                      }
                    }
                  ]
                },
                "version": "2.1.0"
              },
              "permission": {
                "create": {
                  "allow": [
                    "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"
                  ]
                },
                "read": {
                  "allow": [
                    "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10",
                    "group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1"
                  ]
                },
                "update": {
                  "allow": [
                    "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"
                  ]
                },
                "delete": {
                  "allow": [
                    "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"
                  ]
                },
                "share": {
                  "allow": [
                    "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"
                  ]
                }
              },
              "contentType": "text",
              "contentSize": 1511,
              "properties": [
                {
                  "name": "Some Property",
                  "value": "Some Property Value",
                  "classificationPM": "U//FOUO"
                }
              ],
              "containsUSPersonsData": "No",
              "exemptFromFOIA": "No",
              "permissions": [
                {
                  "share": {
                    "users": [
                      "CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US"
                    ]
                  },
                  "allowCreate": false,
                  "allowRead": true,
                  "allowUpdate": true,
                  "allowDelete": false,
                  "allowShare": false
                },
                {
                  "share": {
                    "projects": [
                      {
                        "dctc": {
                          "disp_nm": "DCTC",
                          "groups": [
                            "ODrive_G1"
                          ]
                        }
                      }
                    ]
                  },
                  "allowCreate": false,
                  "allowRead": true,
                  "allowUpdate": false,
                  "allowDelete": false,
                  "allowShare": false
                }
              ]
            }
            --7518615725
            Content-Disposition: form-data; name="filestream"; filename="test.txt"
            Content-Type: application/octet-stream
            
            This is the content of the file
            
            --7518615725--        

+ Request Without a Content Stream (application/json)
    When creating a new object without a content stream, such as a folder, the object definition may be specified directly in the request body as typified below.

    + typeName (string, maxlength=255, required) -  The type to be assigned to this object.  Custom types may be referenced here for the purposes of rules or processing constraints and definition of properties.
       * File - This type may be assigned if no type is given, and you are creating an object with a stream
       * Folder - This type may be assigned if no type is given, and you are creating an object without a stream
    + name: `New Folder` (string, maxlength=255, optional) - The name to be given this object.  If no name is given, then objects are created with the default name pattern of `New <typeName>`.
    + namePathDelimiter: `:::` (string, optional) - An optional alternate path delimiter for which the name given should be assessed to generate intermediate objects when establishing a folder/file structure hierarchy. By default, the name will be split on the record separator (ASCII character 30).
       * Example splitting `abc:::def:::ghi` on `:::`
         * creates object `abc` if it does not already exist
         * creates object `def` if it does not already exist as a child of `abc`
         * creates object `ghi` as a child of `def`
       * Example splitting `abc/def/ghi` on `/`
         * creates object `abc` if it does not already exist
         * creates object `def` if it does not already exist as a child of `abc`
         * creates object `ghi` as a child of `def`
       * Example splitting `(U//FOUO) Turnip Greens` on `/`
         * creates object `(U` if it does not already exist
         * creates object `FOUO) Turnip Greens` as a child of `(U`
       * Example splitting `(U//FOUO) Turnip Greens` with default record separator
         * creates object `(U//FOUO) Turnip Greens`                
    + description (string, maxlength=10240, optional) - An optional abstract of the object's contents.
    + parentId (string, length=32, optional) - Hex encoded identifier of an object, typically a folder, into which this new object is being created as a child object. If no value is specified, then the object will be created in the root location of the user who is creating it.
    + acm (object, required) - Access Control Model is the security model leveraged by the system when enforcing access control. It is based on the ISM, NTK, ACCM and Share standards, requirements and policies.  This value may be provided in either serialized string format, or nested object format.
    + permission (PermissionRequest, optional) - [1.1] The permissions associated with this object by capability and resource allowed.  Resources take the following form:
       * {resourceType}/{serialized-representation}/{optional-display-name}
       * Examples for Users
         * user/{distinguishedName}/{displayName}
         * user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10
       * Examples for groups
         * group/{projectName}/{groupName}
         * group/{projectName}/{projectDisplayName}/{groupName}/{displayName}
         * group/dctc/odrive_g1
         * group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1
         * group/-Everyone
    + contentType: `` (string, maxlength=255, optional) - The suggested mime type for the content stream if given for this object.
    + contentSize: 0 (number, maxvalue=9223372036854775807, optional) - The length of the content stream, in bytes. If there is no content stream, this value should be 0.  The maxvalue given here is theoretical based upon the maximum allowable value represented in 8 bytes. The actual maximum size of an object is initially constrained by free disk storage space in the local cache on the instance the object is being created.
    + containsUSPersonsData: `Yes` (string, maxlength=255, optional) - Indicates if this object contains US Persons data.
        + Default: `Unknown`  
        + Members
            + `Yes`
            + `No`
            + `Unknown`
    + exemptFromFOIA: `No` (string, maxlength=255, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  
        + Default: `Unknown`  
        + Members
            + `Yes`
            + `No`
            + `Unknown`
    + ownedBy: `user/{distinguishedName}/{displayName}` (string, maxlength=255, optional) - Permits assigning the ownership to a group during create
       * Groups that we are in are allowed
         * group/{projectName}/{projectDisplayName}/{groupName}/{displayName}
    + properties (properties array, optional) - Array of custom properties to be associated with the newly created object.
    + permissions (array[PermissionUserCreate,PermissionGroupCreate]) - **[1.0, Deprecated]** - Array of permissions associated with this object.

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

### Bulk Delete Objects [DELETE]

Delete a set of objects to the trash.  It requires the id and the change token for each one.  This operation is limited to a maximum of 1000 items deleted per request.  

+ Request (application/json)

    + Body

            [
                {"ObjectId":"11e5e4867a6e3d8389020242ac110002", "ChangeToken":"e18919"},
                {"ObjectId":"11e5e4867a6f3d8389020242ac110002", "ChangeToken":"a38919"}
            ]

+ Response 200

    + Body

            [
                {"objectId":"11e5e4867a6e3d8389020242ac110002","code":200},
                {"objectId":"11e5e4867a6f3d8389020242ac110002","code":400, "error":"unable to find object", "msg":"cannot delete object"}
            ]

## Object Metadata [/objects/{objectId}/properties]

Metadata for an object may be retrieved or updated at the URI designated.  

+ Parameters
    + objectId: `11e5e48664f5d8c789020242ac110002` (string(length=32), required) - string Hex encoded identifier of the object to be retrieved.

### Get an Object [GET]
This microservice operation retrieves the metadata about an object. 
This operation is used to display properties when selecting an object in the system. 
It may be called on objects int the trash which also expose additional fields in the response.

+ Response 200 (application/json)
    + Attributes (GetObjectResponse)

+ Response 400

        Malformed Request

+ Response 403

        Forbidden

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

    + id: `11e5e48664f5d8c789020242ac110002` (string, length=32, required) - The unique identifier of the object hex encoded to a string. This value must match the objectId provided in the URI.
    + changeToken (string, length=32, required) - The current change token on the object
    + typeName: `Folder` (string, maxlength=255, optional) -  The type to be assigned to this object.  During update if no typeName is given, then the existing type will be retained
    + name (string, maxlength=255, optional) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
    + description (string, maxlength=10240, optional) - The new description to be given as an abstract of the objects content stream. If no value is provided, or this field is omitted, then the description will not be changed.
    + acm (object, optional) -  Access Control Model (ACM) is the security model leveraged by the system when enforcing access control. It is based on the ISM, NTK, ACCM and Share standards, requirements and policies. https://confluence.363-283.io/pages/viewpage.action?pageId=557850. If no value is provided, or this field is omitted, then the acm will not be changed. This value may be provided in either serialized string format, or nested object format.
    + permission (PermissionRequest, optional) - [1.1] The permissions associated with this object by capability and resource allowed.  Resources take the following form:
       * {resourceType}/{serialized-representation}/{optional-display-name}
       * Examples for Users
         * user/{distinguishedName}/{displayName}
         * user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10
       * Examples for groups
         * group/{projectName}/{groupName}
         * group/{projectName}/{projectDisplayName}/{groupName}/{displayName}
         * group/dctc/odrive_g1
         * group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1
         * group/-Everyone
    + contentType: `text/plain` (string, maxlength=255, optional) - The suggested mime type for the content stream for this object.
    + containsUSPersonsData: `Yes` (string, maxlength=255, optional) - Indicates if this object contains US Persons data.
        + Default: `Unknown`  
        + Members
            + `Yes`
            + `No`
            + `Unknown`
    + exemptFromFOIA: `No` (string, maxlength=255, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  
        + Default: `Unknown`  
        + Members
            + `Yes`
            + `No`
            + `Unknown`
    + properties (properties array, optional) -  An array of custom properties to be associated with this object for property changes. For the properties specified, those who do not match existing properties on the object by name will be added. For the properties that do match existing properties by name, if the value specified is blank or empty, then the existing property will be deleted, otherwise, the property will be updated to the new value. If properties are not specified in the array, then existing properties on the object are retained. Properties are only removed from an object if they are provided, with their value set to an empty string.
    + recursiveShare (boolean, optional) - If set to true, updates to sharing and permissions are applied to an objects children. Note that this initiates an asynchronous operation on the server, in the background. Clients may need to wait for changes to take effect.

    + Attributes (UpdateObject)

+ Response 200 (application/json)

    + Attributes (ObjectResp)

+ Response 400

        Unable to decode request
    
+ Response 403

        Forbidden
        
+ Response 404

        The requested object is not found.
        
+ Response 405

        Deleted
        
+ Response 410

        Does Not Exist
        
+ Response 500

        Error storing metadata or stream

## Get Object Stream [/objects/{objectId}/stream{?disposition}]

When retrieving the bytes for an object, the behavior of real web browsers needs to be taken into account.
A request to an object that is successfully found will return a 200 HTTP code and keep streaming back bytes.
It is up to the client to be reasonable when it handles it.
Expect the number of bytes coming back to possibly use an unacceptably large amount of memory (many GB in the response), particularly when writing to the API for the purposes of indexing.
The content-length will state how many bytes can come back, so the client can simply stop reading the stream when if has seen everything it needs to see.
Otherwise, the client must read the data in a streaming fashion to ensure that only a small in-memory buffer is required, generally to forward to an indexing call (ie: Elastic Search).
Use of this feature is not just for performance, as not following this may keep the client from working correctly at all (by getting timeout and out of memory errors).

Actual browser behavior is far more complex than simply retrieving the files, and it is a good idea for custom clients to follow these conventions to get good performance.
When an object stream comes back, it will specify an `ETag` value.  The client should remember this value, and resend an `If-None-Match` which contains the known `ETag` value.
If the content has not been changed (ie: there is no new version for this file), then a `304 Not Modified` will be sent back instead of a stream.
For situations where there are a lot of URLs that resolve to images, and the browser is fetching them just because it doesn't know if they changed, the speedup is going to be dramatic.

![Get Object Stream](static/js/getObjectStream.png)

In addition to ETag, browsers may do range requesting for larger resources.  If a browser goes to get a stream, it will look at its mime type and content-length.
The browser may decide to simply read a small amount of the file from a 200 OK response, and cut off reading the stream early (close the connection).
Then when the browser needs more bytes, it will put in a header like `Range: 23433-432178` to select a specific chunk for more bytes, or leave the range open-ended like `Range:23433-`,
where the browser may again close the connection at its leisure.  When it does this, it will get a `206 Partial Content` code rather than a 200 OK.
It is the client's use of the `Range` tag that allows the response to be 206.

![Get Object Stream](static/js/etag.png)

+ Parameters
    + objectId: `11e5e48664f5d8c789020242ac110002` (string(length=32), required) - Hex encoded identifier of the object to be retrieved.
    + disposition: `attachment` (string, optional) - The Content-Disposition to be set in the header of the response to control UI/Browser operation
        + Default: `inline`
        + Members
            + `inline`
            + `attachment`
  
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

        Forbidden
        
+ Response 405

        Deleted
        
+ Response 410

        Does Not Exist

+ Response 500

        Error storing metadata or stream

## Update Object Stream [/objects/{objectId}/stream]

Updates the actual file bytes associated with an objectId. This must be provided in multipart/form-data format, with the metadata about the object provided in a field named 'ObjectMetadata'.

+ Parameters
    + objectId: `11e5e48664f5d8c789020242ac110002` (string(length=32), required) - Hex encoded identifier of the object to be retrieved.

### Update an Object Stream [POST]

This creates a new revision of the object.

+ Request (multipart/form-data; boundary=b428e6cd1933)

    The JSON object provided in the body can contain the following fields:

    + id: `11e5e48664f5d8c789020242ac110002` (string, length=32, required) - The unique identifier of the object hex encoded to a string. This value must match the objectId provided in the URI.
    + changeToken (string, length=32, required) - A hash value expected to match the targeted object’s current changeToken value. This value is retrieved from get or list operations.
    + typeName (string, maxlength=255, optional) -  The new type to be assigned to this object. Common types include 'File', 'Folder'. If no value is provided or this field is omitted, then the type will not be changed.
    + name (string, maxlength=255, optional) - The new name to be given this object. It does not have to be unique. It may refer to a conventional filename and extension. If no value is provided, or this field is omitted, then the name will not be changed.
    + description (string, maxlength=10240, optional) - The new description to be given as an abstract of the objects content stream. If no value is provided, or this field is omitted, then the description will not be changed.
    + acm (object, optional) -  Access Control Model (ACM) is the security model leveraged by the system when enforcing access control. It is based on the ISM, NTK, ACCM and Share standards, requirements and policies. https://confluence.363-283.io/pages/viewpage.action?pageId=557850. If no value is provided, or this field is omitted, then the acm will not be changed.  This value may be provided in either serialized string format, or nested object format.
    + permission (PermissionRequest, optional) - [1.1] The permissions associated with this object by capability and resource allowed.  Resources take the following form:
       * {resourceType}/{serialized-representation}/{optional-display-name}
       * Examples for Users
         * user/{distinguishedName}/{displayName}
         * user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10
       * Examples for groups
         * group/{projectName}/{groupName}
         * group/{projectName}/{projectDisplayName}/{groupName}/{displayName}
         * group/dctc/odrive_g1
         * group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1
         * group/-Everyone
    + contentType: `text/html` (string, maxlength=255, optional) - The suggested mime type for the content stream for this object.
    + contentSize: 0 (number, maxvalue=9223372036854775807, optional) - The length of the content stream, in bytes. If there is no content stream, this value should be 0.  The maxvalue given here is theoretical based upon the maximum allowable value represented in 8 bytes. The actual maximum size of an object is initially constrained by free disk storage space in the local cache on the instance the object is being created.
    + containsUSPersonsData: `Yes` (string, maxlength=255, optional) - Indicates if this object contains US Persons data.
        + Default: `Unknown`  
        + Members
            + `Yes`
            + `No`
            + `Unknown`
    + exemptFromFOIA: `No` (string, maxlength=255, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  
        + Default: `Unknown`  
        + Members
            + `Yes`
            + `No`
            + `Unknown`
    + properties (properties array, optional) -  An array of custom properties to be associated with this object for property changes. For the properties specified, those who do not match existing properties on the object by name will be added. For the properties that do match existing properties by name, if the value specified is blank or empty, then the existing property will be deleted, otherwise, the property will be updated to the new value. If properties are not specified in the array, then existing properties on the object are retained. Properties are only removed from an object if they are provided, with their value set to an empty string.
    + recursiveShare (boolean, optional) - If set to true, updates to sharing and permissions are applied to an objects children. Note that this initiates an asynchronous operation on the server, in the background. Clients may need to wait for changes to take effect.

    The content stream for the object should be the second part, as the native bytes without use of encoding or character sets.
           
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

        Forbidden
        
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
    + objectId: `11e5e4867a6e3d8389020242ac110002` (string(length=32), required) - Hex encoded identifier of the object to be deleted.

### Delete Object [POST]
This microservice operation handles the deletion of an object within Object Drive. When objects are deleted, they are marked as such but remain intact for auditing purposes and the ability to restore (remove from trash). All other operations that pertain to retrieval or updating filter deleted objects internally. The exception to this is when viewing the contents of the trash via List Trashed Objects, or performing Undelete Object, and Delete Object Forever operations.

This creates a new revision of the object.

When an object is deleted, a recursive action is performed on all natural children to set an internal flag denoting an ancestor as deleted, unless that child object is deleted, in which case, the recursion down that branch terminates.
           
+ Request (application/json)

    + Attributes (DeleteObjectRequest)
            
+ Response 200 (application/json)

    + Attributes (ObjectDeleted)

+ Response 400

        Unable to decode request
        
+ Response 403

        Forbidden
        
+ Response 405

        Deleted

+ Response 410

        Does Not Exist
        
+ Response 500

        Error storing metadata or stream

## Delete Object Forever [/objects/{objectId}]  

+ Parameters
     + objectId: `11e5e4867a6e3d8389020242ac110002` (string(length=32), required) - Hex encoded identifier of the object to be deleted.

### Delete Object Forever [DELETE]
This microservice operation will remove an object from the trash and delete it forever.  

+ Request (application/json)

    + Attributes (DeleteObjectRequest)

+ Response 200 (application/json)

    + Attributes (ObjectExpunged)

+ Response 400

        Unable to decode request
        
+ Response 403

        Forbidden
        
+ Response 405

        Deleted
        
+ Response 410

        Does Not Exist
        
+ Response 500

        Error storing metadata or stream


# Group Versioning Operations

---

## List Object Revisions [/revisions/{objectId}{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

+ Parameters
    + objectId: `11e5e4867a6e3d8389020242ac110002` (string(length=32), required) - Hex encoded identifier of the object for which revisions are being requested.
    + pageNumber: 1 (number(minvalue=1), optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number(minvalue=1, maxvalue=10000), optional) - The number of results to return per page.
    + sortField: `contentsize` (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
        + Default: `createddate`
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + sortAscending: true (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: false
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition). Introduced in v1.0.16, field names specified that do not appear in the list below will be compared to custom properties of the given name for potential removal from the returned page of the resultset.
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + condition: `equals` (enum[string], optional) - **experimental** - The match type for filtering
        + Members
            + `begins`
            + `contains`
            + `ends`
            + `equals`
            + `lessthan`
            + `morethan`
            + `notbegins`
            + `notcontains`
            + `notends`
            + `notequals`
    + expression: `0` (string, optional) - **experimental** - A phrase that should be used for the match against the field value


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


## Get Older Object Stream [/revisions/{objectId}/{revisionId}/stream{?disposition}]

+ Parameters
    + objectId: `11e5e4867a6e3d8389020242ac110002` (string(length=32), required) - Hex encoded identifier of the object to be retrieved.
    + revisionId: 2 (number(minvalue=0), required) - The revision number to be retrieved. 
    + disposition: `attachment` (string, optional) - The value to assign the Content-Disposition in the response.
        + Default: `inline`
        + Members
            + `inline` - The default disposition
            + `attachment` - Supports browser prompting the user to save the response as a file.

### Get Older Object Stream [GET]

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

        If the user is forbidden to perform the request because they lack permissions to view the object.
        
+ Response 405

        If the object or an ancestor is deleted.
        
+ Response 410

        If the object referenced no longer exists
        
+ Response 500

        * Malformed JSON
        * Error retrieving object
        * Error determining user.`

## Restore Revision as Current [/revisions/{objectId}/{revisionId}/restore]

+ Parameters
    + objectId: `11e5e4867a6e3d8389020242ac110002` (string(length=32), required) - Hex encoded identifier of the object to be updated.
    + revisionId: 2 (number(minvalue=0), required) - The revision number to be restored as the current version. 

### Restore Revision as Current [POST]
This microservice operation makes a copy of the prior specified version and makes it the current version as a new revision without the need for uploading a file or specifying properties, fields or permissions.  The version being restored must have the same location and owner, and the caller of the operation must have update privileges on the current revision, and at least read access on the prior revision.  Current properties are replaced by the properties on the revision being restored.  Permissions and ACM remain the same as the current version.  You cannot restore a deleted version, or any version if the current version is deleted.

+ Request (application/json)

    The JSON object in the request body should contain the change token of the current revision:

    + Attributes (ChangeToken)

+ Response 200 (application/json)
    + Attributes (GetObjectResponse)

+ Response 400

        Malformed Request

+ Response 403

        Forbidden

+ Response 404

        Not Found

+ Response 410

        Does Not Exist

+ Response 500

        Error Retrieving Object



# Group Search & List Operations

---

## Search [/search/{searchPhrase}{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

**EXPERIMENTAL** - Search operations are an experimental feature.

The **id** field values for sortField and filterFiled refer to the object id only.  While there are other identifiers for metadata fields, these are not directly supported here.  If you want to search on objects specific to a single folder identified by parentid, make use of the List Folder Objects and List User Objects at Root operations defined elsewhere.

+ Parameters
    + searchPhrase: `image/gif` (string, required) - The phrase to look for inclusion within the name or description of objects. This will be overridden if parameters for filterField are set.
    + pageNumber: 1 (number(minvalue=1), optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number(minvalue=1, maxvalue=10000), optional) - The number of results to return per page.
    + sortField: `contentsize` (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
        + Default: `createddate`
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + sortAscending: true (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: false
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition). Introduced in v1.0.16, field names specified that do not appear in the list below will be compared to custom properties of the given name for potential removal from the returned page of the resultset.
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + condition: `equals` (enum[string], optional) - **experimental** - The match type for filtering
        + Members
            + `begins`
            + `contains`
            + `ends`
            + `equals`
            + `lessthan`
            + `morethan`
            + `notbegins`
            + `notcontains`
            + `notends`
            + `notequals`
    + expression: `0` (string, optional) - **experimental** - A phrase that should be used for the match against the field value

### Search [GET]

+ Response 200 (application/json)
    + Attributes (ObjectResultset)

+ Response 204
  
+ Response 400

        Malformed request.
        
+ Response 500

        * Error retrieving object
        * Error determining user.

## List User Objects At Root [/objects{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

+ Parameters

    + pageNumber: 1 (number(minvalue=1), optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number(minvalue=1, maxvalue=10000), optional) - The number of results to return per page.
    + sortField: `contentsize` (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
        + Default: `createddate`
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + sortAscending: true (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: false
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition) Introduced in v1.0.16, field names specified that do not appear in the list below will be compared to custom properties of the given name for potential removal from the returned page of the resultset.
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + condition: `equals` (enum[string], optional) - **experimental** - The match type for filtering
        + Members
            + `begins`
            + `contains`
            + `ends`
            + `equals`
            + `lessthan`
            + `morethan`
            + `notbegins`
            + `notcontains`
            + `notends`
            + `notequals`
    + expression: `0` (string, optional) - **experimental** - A phrase that should be used for the match against the field value

### List User Objects At Root [GET]

This microservice operation retrieves a list of objects at the root owned by the caller, with optional settings for pagination, sorting, and filtering.

+ Response 200 (application/json)

    + Attributes (ObjectResultset)

+ Response 400

        Unable to decode request
        
+ Response 403

        Forbidden

+ Response 405

        Deleted
        
+ Response 410

        Does Not Exist

## List Groups having Objects [/groups]

### List Groups having Objects [GET]

This microservice operation retrieves a list of groups for which the user is a member that have objects at the root.  As users may be members of an undeterminate number of groups, this eliminates the need to list the objects of every group to determine if a group currently contains objects.

+ Response 200 (application/json)

    + Attributes (GroupSpaceResultset)
       
+ Response 500

        Error retrieving groupspaces

## List Group Objects At Root [/groupobjects/{groupName}{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

+ Parameters

    + groupName: dctc_odrive_g1 (string, required) - The flattened name of a group for which the user is a member and objects owned by the group should be returned.
        * The flattened values for user identity are also acceptable
        * Psuedogroups, such as `_everyone` are not acceptable for this request, but are forbidden from owning objects anyway.
    + pageNumber: 1 (number(minvalue=1), optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number(minvalue=1, maxvalue=10000), optional) - The number of results to return per page.
    + sortField: `contentsize` (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
        + Default: `createddate`
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + sortAscending: true (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: false
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition). Introduced in v1.0.16, field names specified that do not appear in the list below will be compared to custom properties of the given name for potential removal from the returned page of the resultset.
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + condition: `equals` (enum[string], optional) - **experimental** - The match type for filtering
        + Members
            + `begins`
            + `contains`
            + `ends`
            + `equals`
            + `lessthan`
            + `morethan`
            + `notbegins`
            + `notcontains`
            + `notends`
            + `notequals`
    + expression: `0` (string, optional) - **experimental** - A phrase that should be used for the match against the field value

### List Group Objects At Root [GET]

This microservice operation retrieves a list of objects with no parent owned by the specified group, with optional settings for pagination, sorting, and filtering.

+ Response 200 (application/json)

    + Attributes (ObjectResultset)

+ Response 400

        Unable to decode request.
        No groupName was provided
        
+ Response 403

        Forbidden if the user is not a member of the provided group name.
       
+ Response 500

        Error retrieving objects

## List Folder Objects [/objects/{objectId}{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

+ Parameters
    + objectId: `11e5e4867a6e3d8389020242ac110002` (string(length=32), required) - Hex encoded unique identifier of the folder or other object for which to return a list of child objects. 
    + pageNumber: 1 (number(minvalue=1), optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number(minvalue=1, maxvalue=10000), optional) - The number of results to return per page.
    + sortField: `contentsize` (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
        + Default: `createddate`
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + sortAscending: true (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: false
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition). Introduced in v1.0.16, field names specified that do not appear in the list below will be compared to custom properties of the given name for potential removal from the returned page of the resultset.
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + condition: `equals` (enum[string], optional) - **experimental** - The match type for filtering
        + Members
            + `begins`
            + `contains`
            + `ends`
            + `equals`
            + `lessthan`
            + `morethan`
            + `notbegins`
            + `notcontains`
            + `notends`
            + `notequals`
    + expression: `0` (string, optional) - **experimental** - A phrase that should be used for the match against the field value

### List Folder Objects [GET]
Purpose: This microservice operation retrieves a list of objects contained within the specified parent, with optional settings for pagination. By default, this operation only returns metadata about the first 20 items.

+ Response 200 (application/json)

    + Attributes (ObjectResultsetChildren)

+ Response 400

        Unable to decode request
        
+ Response 403

        If the user is forbidden from listing children of an object because they don't have read access to it
        
+ Response 405

        Deleted
        
+ Response 410

        Does Not Exist

+ Response 500

        Error retrieving object represented as the parent to retrieve children, or some other error.

## List Public Objects [/sharedpublic{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

+ Parameters
    + pageNumber: 1 (number(minvalue=1), optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number(minvalue=1, maxvalue=10000), optional) - The number of results to return per page.
    + sortField: `contentsize` (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
        + Default: `createddate`
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + sortAscending: true (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: false
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition). Introduced in v1.0.16, field names specified that do not appear in the list below will be compared to custom properties of the given name for potential removal from the returned page of the resultset.
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + condition: `equals` (enum[string], optional) - **experimental** - The match type for filtering
        + Members
            + `begins`
            + `contains`
            + `ends`
            + `equals`
            + `lessthan`
            + `morethan`
            + `notbegins`
            + `notcontains`
            + `notends`
            + `notequals`
    + expression: `0` (string, optional) - **experimental** - A phrase that should be used for the match against the field value
    
### List Public Objects [GET]

This microservice operation retrieves a list of objects that are shared to everyone.

+ Request

    + Header
    
            Content-Type: application/json

+ Response 200
    + Attributes (ObjectResultset)

+ Response 400

        Unable to decode request
        
+ Response 500

        Error storing metadata or stream

# Group File Retrieval

---

## Download File By Path [/files/{path}{?disposition}]

+ Parameters
    + path: `folder/subfolder/file.txt` (string, optional) - The path to a file to be retrieved
    + disposition: `attachment` (string, optional) - The Content-Disposition to be set in the header of the response to control UI/Browser operation
        + Default: `inline`
        + Members
            + `inline`
            + `attachment`
### Download File By Path [GET]
This microservice operation is experimental (since v1.0.14). This microservice operation retrieves a file content stream given a standard URI path. For each component of the path the service will identify matching file or folder of that name for which the user has read access to iteratively until it reaches the final node.  If a user does not have read access to any component of the path, an error code will be returned.  This is a convenience function around manually making calls to list/search for objects with a given name to accomplish the same.

Headers are passed along to support range requests, ETag values, and so forth.

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

        Forbidden

+ Response 404

        File Not Found

+ Response 405

        Deleted
        
+ Response 410

        Does Not Exist


## List Files By Path [/files/{path}/{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

+ Parameters
    + path: `folder/subfolder/` (string, optional) - The path to a folder to retrieve a directory listing
    + pageNumber: 1 (number(minvalue=1), optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number(minvalue=1, maxvalue=10000), optional) - The number of results to return per page.
    + sortField: `contentsize` (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
        + Default: `createddate`
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + sortAscending: true (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: false
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition). Introduced in v1.0.16, field names specified that do not appear in the list below will be compared to custom properties of the given name for potential removal from the returned page of the resultset.
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + condition: `equals` (enum[string], optional) - **experimental** - The match type for filtering
        + Members
            + `begins`
            + `contains`
            + `ends`
            + `equals`
            + `lessthan`
            + `morethan`
            + `notbegins`
            + `notcontains`
            + `notends`
            + `notequals`
    + expression: `0` (string, optional) - **experimental** - A phrase that should be used for the match against the field value

### List Files By Path [GET]
This microservice operation is experimental (since v1.0.14). This microservice operation retrieves a list of objects contained within the specified path, with optional settings for pagination. By default, this operation only returns metadata about the first 20 items.  For each component of the path the service will identify matching file or folder of that name for which the user has read access to iteratively until it reaches the final node.  If a user does not have read access to any component of the path, an error code will be returned.  This is a convenience function around manually making calls to list/search for objects with a given name 
to accomplish the same.

+ Response 200 (application/json)

    + Attributes (ObjectResultsetChildren)

+ Response 400

        Unable to decode request
        
+ Response 403

        If the user is forbidden from listing children of an object or a component of the path because they don't have read access to it
        
+ Response 405

        Deleted
        
+ Response 410

        Does Not Exist

+ Response 500

        Error retrieving object represented as the parent to retrieve children, or some other error.

## Download Group File By Path [/files/groupobjects/{groupName}/{path}{?disposition}]

+ Parameters
    + groupName: dctc_odrive_g1 (string, required) - The flattened name of a group for which to base initial object ownership under when looking at the path for file to retrieve
        * The flattened values for user identity are also acceptable
        * Psuedogroups, such as `_everyone` are not acceptable for this request, but are forbidden from owning objects anyway.
    + path: `folder/subfolder/file.txt` (string, optional) - The path to a file to be retrieved
    + disposition: `attachment` (string, optional) - The Content-Disposition to be set in the header of the response to control UI/Browser operation
        + Default: `inline`
        + Members
            + `inline`
            + `attachment`
### Download Group File By Path [GET]
This microservice operation is experimental (since v1.0.14). This microservice operation retrieves a file content stream given a standard URI path. For each component of the path the service will identify matching file or folder of that name for which the user has read access to iteratively until it reaches the final node.  If a user does not have read access to any component of the path, an error code will be returned.  This is a convenience function around manually making calls to list/search for objects with a given name to accomplish the same.

Headers are passed along to support range requests, ETag values, and so forth.

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

        Forbidden

+ Response 404

        File Not Found

+ Response 405

        Deleted
        
+ Response 410

        Does Not Exist


## List Group Files By Path [/files/groupobjects/{groupName}/{path}/{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

+ Parameters
    + groupName: dctc_odrive_g1 (string, required) - The flattened name of a group for which to base initial object ownership under when looking at the path for list of objects to retrieve
        * The flattened values for user identity are also acceptable
        * Psuedogroups, such as `_everyone` are not acceptable for this request, but are forbidden from owning objects anyway.
    + path: `folder/subfolder/` (string, optional) - The path to a folder to retrieve a directory listing
    + pageNumber: 1 (number(minvalue=1), optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number(minvalue=1, maxvalue=10000), optional) - The number of results to return per page.
    + sortField: `contentsize` (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
        + Default: `createddate`
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + sortAscending: true (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: false
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition). Introduced in v1.0.16, field names specified that do not appear in the list below will be compared to custom properties of the given name for potential removal from the returned page of the resultset.
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + condition: `equals` (enum[string], optional) - **experimental** - The match type for filtering
        + Members
            + `begins`
            + `contains`
            + `ends`
            + `equals`
            + `lessthan`
            + `morethan`
            + `notbegins`
            + `notcontains`
            + `notends`
            + `notequals`
    + expression: `0` (string, optional) - **experimental** - A phrase that should be used for the match against the field value

### List Group Files By Path [GET]
This microservice operation is experimental (since v1.0.14). This microservice operation retrieves a list of objects contained within the specified path, with optional settings for pagination. By default, this operation only returns metadata about the first 20 items.  For each component of the path the service will identify matching file or folder of that name for which the user has read access to iteratively until it reaches the final node.  If a user does not have read access to any component of the path, an error code will be returned.  This is a convenience function around manually making calls to list/search for objects with a given name 
to accomplish the same.

+ Response 200 (application/json)

    + Attributes (ObjectResultsetChildren)

+ Response 400

        Unable to decode request
        
+ Response 403

        If the user is forbidden from listing children of an object or a component of the path because they don't have read access to it
        
+ Response 405

        Deleted
        
+ Response 410

        Does Not Exist

+ Response 500

        Error retrieving object represented as the parent to retrieve children, or some other error.



# Group Filing Operations

---

## Copy Object [/objects/{objectId}/copy]

+ Parameters
    + objectId: `11e5e4867a6e3d8389020242ac110002` (string(length=32), required) - Hex encoded identifier of the object to be copied.

### Copy Object [POST]
This microservice operation supports copying an object and its revisions, including permissions and any dynamic properties therein to a new object of the same general characteristics and location, but owned by the user initiating the operation.

Only those revisions for which the caller has permission to retrieve are copied to the new object. Thus, there may be fewer revisions then the original object.  The object created references the same underlying content stream as the source object
and its revisions (A copy of such is not made in permanent storage, since it already exists).  The user initiating the call,
by virtue of being owner, will receive full CRUDS permissions added to the copied object.

This operation does not require a content type header to be set, nor a body to be provided. Such attributes are reserved for future use.

+ Response 200 (application/json)
    + Attributes (ObjectResp)

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


## Move Object [/objects/{objectId}/move/{folderId}]

+ Parameters
    + objectId: `11e5e4867a6e3d8389020242ac110002` (string(length=32), required) - Hex encoded identifier of the object to be moved.
    + folderId: `30211e5e48ac110067a6e3d802420289` (string(length=32), optional) - Hex encoded identifier of the folder into which this object should be moved.  If no identifier is provided, then the object will be moved to the owner's root folder.

### Move Object [POST]
This microservice operation supports moving an object such as a file or folder from one location to another. By default, all objects are created in the ‘root’ as they have no parent folder given.

This creates a new revision of the object.

Only the owner of an object is allowed to move it.

+ Request (application/json)

    The JSON object in the request body should contain a change token:

    + Attributes (MoveObjectRequest)

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

## Change Owner [/objects/{objectId}/owner/{newOwner}]

+ Parameters
    + objectId: `11e5e4867a6e3d8389020242ac110002` (string(length=32), required) - Hex encoded identifier of the object to be moved.
    + newOwner: `group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1` (string(maxlength=255), required) - A resource string compliant value representing the new owner. Resources take the following form:
       * {resourceType}/{serialized-representation}/{optional-display-name}
       * Examples for Users
         * user/{distinguishedName}/{displayName}
         * user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10
       * Examples for groups
         * group/{projectName}/{groupName}
         * group/{projectName}/{projectDisplayName}/{groupName}/{displayName}
         * group/dctc/odrive_g1
         * group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1
         * group/-Everyone

### Change Owner [POST]
This microservice operation supports tranferring ownership to a new user or group identified by common resource string format.

This creates a new revision of the object.

The object hierarchy will be reset to the root of the new owner.

Only the current owner of an object is allowed to change ownership.  If the object is owned by a group, then any member of that group may change its owner.

The transferee will be granted full (CRUDS) permissions. The transferor will retain existing permissions. 

Although it is not permitted to assign ownership to Everyone, ownership may be assigned to unverified users or groups for which the user is not a member.

+ Request (application/json)

    The JSON object in the request body should contain a change token and whether to apply the operation recursively:

    + Attributes (ChangeOwnerRequest)

+ Response 200 (application/json)
    + Attributes (ObjectResp)

+ Response 400

        Unable to decode request
        A new owner is required when changing owner
        Value provided for new owner could not be parsed
        
+ Response 403

        Forbidden
        
+ Response 404

        The requested object is not found
        
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

## List User Object Shares [/shares{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

+ Parameters
    + pageNumber: 1 (number(minvalue=1), optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number(minvalue=1, maxvalue=10000), optional) - The number of results to return per page.
    + sortField: `contentsize` (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
        + Default: `createddate`
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + sortAscending: true (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: false
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition). Introduced in v1.0.16, field names specified that do not appear in the list below will be compared to custom properties of the given name for potential removal from the returned page of the resultset.
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + condition: `equals` (enum[string], optional) - **experimental** - The match type for filtering
        + Members
            + `begins`
            + `contains`
            + `ends`
            + `equals`
            + `lessthan`
            + `morethan`
            + `notbegins`
            + `notcontains`
            + `notends`
            + `notequals`            
    + expression: `0` (string, optional) - **experimental** - A phrase that should be used for the match against the field value

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

## List User Objects Shared [/shared{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

+ Parameters
    + pageNumber: 1 (number(minvalue=1), optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number(minvalue=1, maxvalue=10000), optional) - The number of results to return per page.
    + sortField: `contentsize` (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
        + Default: `createddate`
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + sortAscending: true (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: false
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition). Introduced in v1.0.16, field names specified that do not appear in the list below will be compared to custom properties of the given name for potential removal from the returned page of the resultset.
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + condition: `equals` (enum[string], optional) - **experimental** - The match type for filtering
        + Members
            + `begins`
            + `contains`
            + `ends`
            + `equals`
            + `lessthan`
            + `morethan`
            + `notbegins`
            + `notcontains`
            + `notends`
            + `notequals`
    + expression: `0` (string, optional) - **experimental** - A phrase that should be used for the match against the field value

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
        

## List Trashed Objects [/trashed{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

+ Parameters
    + pageNumber: 1 (number(minvalue=1), optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number(minvalue=1, maxvalue=10000), optional) - The number of results to return per page.
    + sortField: `contentsize` (string, optional) - Denotes a field that the results should be sorted on. Can be specified multiple times for complex sorting.
        + Default: `createddate`
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + sortAscending: true (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: false
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition). Introduced in v1.0.16, field names specified that do not appear in the list below will be compared to custom properties of the given name for potential removal from the returned page of the resultset.
        + Members
            + `changecount`
            + `createdby`
            + `createddate`
            + `contentsize`
            + `contenttype`
            + `description`
            + `foiaexempt`
            + `id`
            + `modifiedby`
            + `modifieddate`
            + `name`
            + `ownedby`
            + `typename`
            + `uspersons`
    + condition: `equals` (enum[string], optional) - **experimental** - The match type for filtering
        + Members
            + `begins`
            + `contains`
            + `ends`
            + `equals`
            + `lessthan`
            + `morethan`
            + `notbegins`
            + `notcontains`
            + `notends`
            + `notequals`
    + expression: `0` (string, optional) - **experimental** - A phrase that should be used for the match against the field value

### List Trashed Objects [GET]

+ Request (application/json)

+ Response 200 (application/json)
    + Attributes (ObjectResultsetDeleted)

+ Response 400

        Unable to decode request
        
+ Response 500

        Error storing metadata or stream

## Undelete Object [/objects/{objectId}/untrash]

This operation restores a previously deleted object from the trash. Recursively, children of the previously deleted object will also be restored.

This creates a new revision of the object.

+ Parameters
     + objectId: `30211e5e48ac110067a6e3d802420289` (string, required) - Hex encoded identifier of the object to be deleted.

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



## Empty Trash [/trashed{?pageSize}]

Objects that have been put into the trash can be expunged until the trash is emptied.
This is effectively the same as calling the operation Delete Object Forever for everything that has been trashed.

+ Parameters
    + pageSize: 10000 (number(minvalue=1, maxvalue=10000), optional) - The batch size to expunge objects in

### Empty Trash [DELETE]

+ Response 200 (application/json)

    + Body

            {
                "expunged_count": 20
            }	        

## User Stats [/userstats]

User Stats provides metrics information for the user's total number of objects and revisions and the amount of size consumed in the system.

### User Stats [GET]

+ Request (application/json)

+ Response 200 (application/json)
    + Attributes (UserStats)
    
+ Response 500

        Internal Server Error


# Group Auxiliary &amp; Bulk Operations

---

## Bulk Get Objects Properties [/objects/properties]

### Bulk Get Objects [POST]
Get multiple objects at once

This returns an object result set.  Note that because this gets
objects in bulk, it is 
possible a list of Errors coming back with the objects that came back successfully.

+ Request (application/json)

    + Body

            {
                "objectIds" : [
                        "11e5e4867a6e3d8389020242ac110002",
                        "11e5e4867a6e3d8389020242ac189124",
                        "11e5e4867a6e3f8389020242ac110002"
                ]
            }

+ Response 200

    + Body
    
            {
            totalRows: 2,
            pageNumber: 1,
            pageRows: 1,
            pageNumber: 1,
            objects:[    
                {
                "id": "11e5e4867a6e3d8389020242ac110002",
                "createdDate": "2016-03-07T17:03:13Z",
                "createdBy": "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
                "modifiedDate": "2016-03-07T17:03:13Z",
                "modifiedBy": "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
                "deletedDate": "0001-01-01T00:00:00Z",
                "deletedBy": "``",
                "changeCount": 42,
                "changeToken": "65eea405306ed436d18b8b1c0b0b2cd3",
                "ownedBy": "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
                "typeId": "11e5e48664f5d8c789020242ac110002",
                "typeName": "File",
                "name": "gettysburgaddress.txt",
                "description": "Description here",
                "parentId": "11e5e4867a6e3d8489020242ac110002",
                "acm": {
                    "banner": "UNCLASSIFIED",
                    "classif": "U",
                    "dissem_countries": [
                    "USA"
                    ],
                    "f_accms": [],
                    "f_atom_energy": [],
                    "f_clearance": "u",
                    "f_macs": [],
                    "f_missions": [],
                    "f_oc_org": [],
                    "f_sci_ctrls": [],
                    "f_regions": [],
                    "f_share": [
                    "x",
                    "y",
                    "z"
                    ],
                    "portion": "U",
                    "share": {
                    "users": [
                        "CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                        "CN=test tester02,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                        "CN=test tester03,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                        "CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US"
                    ],
                    "projects": [
                        {
                        "ukpn": {
                            "disp_nm": "Project Name",
                            "groups": [
                            "Group Name",
                            "Cats",
                            "Dogs"
                            ]
                        },
                        "ukpn2": {
                            "disp_nm": "Project Name 2",
                            "groups": [
                            "Group 1",
                            "Group 2",
                            "Group 3"
                            ]
                        }
                        }
                    ]
                    },
                    "version": "2.1.0"
                },
                "permission": {
                    "create": {
                    "allow": [
                        "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"
                    ],
                    "deny": [
                        "``"
                    ]
                    },
                    "read": {
                    "allow": [
                        "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10",
                        "group/dctc/dctc/odrive_g1"
                    ],
                    "deny": [
                        "``"
                    ]
                    },
                    "update": {
                    "allow": [
                        "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"
                    ],
                    "deny": [
                        "``"
                    ]
                    },
                    "delete": {
                    "allow": [
                        "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"
                    ],
                    "deny": [
                        "``"
                    ]
                    },
                    "share": {
                    "allow": [
                        "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"
                    ],
                    "deny": [
                        "``"
                    ]
                    }
                },
                "contentType": "text",
                "contentSize": 1511,
                "containsUSPersonsData": "No",
                "exemptFromFOIA": "No",
                "properties": [
                    {
                    "id": "11e5e4867a6e3d8389020242ac110002",
                    "createdDate": "2016-03-07T17:03:13.1234Z",
                    "createdBy": "CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                    "modifiedDate": "2016-03-07T17:03:13.1234Z",
                    "modifiedBy": "CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                    "changeCount": 1,
                    "changeToken": "65eea405306ed436d18b8b1c0b0b2cd3",
                    "name": "Some Property",
                    "value": "Some Property Value",
                    "classificationPM": "U"
                    }
                ],
                "callerPermissions": {
                    "allowCreate": false,
                    "allowRead": true,
                    "allowUpdate": false,
                    "allowDelete": false,
                    "allowShare": false
                },
                "permissions": [
                    {
                    "grantee": "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
                    "userDistinguishedName": "CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                    "displayName": "test tester10",
                    "allowCreate": true,
                    "allowRead": true,
                    "allowUpdate": true,
                    "allowDelete": true,
                    "allowShare": true
                    },
                    {
                    "grantee": "dctc_odrive_g1",
                    "projectName": "dctc",
                    "projectDisplayName": "dctc",
                    "groupName": "odrive_g1",
                    "displayName": "dctc odrive_g1",
                    "allowCreate": true,
                    "allowRead": true,
                    "allowUpdate": true,
                    "allowDelete": true,
                    "allowShare": true
                    }
                ],
                "breadcrumbs": [
                    {
                    "id": "11e0202427a6eac115e4863d83890002",
                    "parentId": "",
                    "name": "parentFolderA"
                    },
                    {
                    "id": "11e5e4867a6e3d8489020242ac110002",
                    "parentId": "11e0202427a6eac115e4863d83890002",
                    "name": "folderA"
                    }
                ]
                },{
                "id": "11e5e4867a6e3d8389020242ac189124",
                "createdDate": "2016-03-07T17:03:13.123Z",
                "createdBy": "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
                "modifiedDate": "2016-03-07T17:03:13.123456Z",
                "modifiedBy": "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
                "deletedDate": "0001-01-01T00:00:00Z",
                "deletedBy": "``",
                "changeCount": 42,
                "changeToken": "65eea405306ed436d18b8b1c0b0b2cd3",
                "ownedBy": "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
                "typeId": "11e5e48664f5d8c789020242ac110002",
                "typeName": "File",
                "name": "gettysburgaddress.txt",
                "description": "Description here",
                "parentId": "11e5e4867a6e3d8489020242ac110002",
                "acm": {
                    "banner": "UNCLASSIFIED",
                    "classif": "U",
                    "dissem_countries": [
                    "USA"
                    ],
                    "f_accms": [],
                    "f_atom_energy": [],
                    "f_clearance": "u",
                    "f_macs": [],
                    "f_missions": [],
                    "f_oc_org": [],
                    "f_sci_ctrls": [],
                    "f_regions": [],
                    "f_share": [
                    "x",
                    "y",
                    "z"
                    ],
                    "portion": "U",
                    "share": {
                    "users": [
                        "CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                        "CN=test tester02,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                        "CN=test tester03,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                        "CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US"
                    ],
                    "projects": [
                        {
                        "ukpn": {
                            "disp_nm": "project name",
                            "groups": [
                            "group name",
                            "cats",
                            "dogs"
                            ]
                        },
                        "ukpn2": {
                            "disp_nm": "project name 2",
                            "groups": [
                            "group 1",
                            "group 2",
                            "group 3"
                            ]
                        }
                        }
                    ]
                    },
                    "version": "2.1.0"
                },
                "permission": {
                    "create": {
                    "allow": [
                        "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"
                    ],
                    "deny": [
                        "``"
                    ]
                    },
                    "read": {
                    "allow": [
                        "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10",
                        "group/dctc/dctc/odrive_g1"
                    ],
                    "deny": [
                        "``"
                    ]
                    },
                    "update": {
                    "allow": [
                        "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"
                    ],
                    "deny": [
                        "``"
                    ]
                    },
                    "delete": {
                    "allow": [
                        "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"
                    ],
                    "deny": [
                        "``"
                    ]
                    },
                    "share": {
                    "allow": [
                        "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"
                    ],
                    "deny": [
                        "``"
                    ]
                    }
                },
                "contentType": "text",
                "contentSize": 1511,
                "containsUSPersonsData": "No",
                "exemptFromFOIA": "No",
                "properties": [
                    {
                    "id": "11e5e4867a6e3d8389020242ac189124",
                    "createdDate": "2016-03-07T17:03:13.4876Z",
                    "createdBy": "CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                    "modifiedDate": "2016-03-07T17:03:13.4876Z",
                    "modifiedBy": "CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                    "changeCount": 1,
                    "changeToken": "65eea405306ed436d18b8b1c0b0b2cd3",
                    "name": "Some Property",
                    "value": "Some Property Value",
                    "classificationPM": "U"
                    }
                ],
                "callerPermissions": {
                    "allowCreate": false,
                    "allowRead": true,
                    "allowUpdate": false,
                    "allowDelete": false,
                    "allowShare": false
                },
                "permissions": [
                    {
                    "grantee": "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
                    "userDistinguishedName": "CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                    "displayName": "test tester10",
                    "allowCreate": true,
                    "allowRead": true,
                    "allowUpdate": true,
                    "allowDelete": true,
                    "allowShare": true
                    },
                    {
                    "grantee": "dctc_odrive_g1",
                    "projectName": "dctc",
                    "projectDisplayName": "dctc",
                    "groupName": "odrive_g1",
                    "displayName": "dctc odrive_g1",
                    "allowCreate": true,
                    "allowRead": true,
                    "allowUpdate": true,
                    "allowDelete": true,
                    "allowShare": true
                    }
                ],
                "breadcrumbs": [
                    {
                    "id": "11e0202427a6eac115e4863d83890002",
                    "parentId": "",
                    "name": "parentFolderA"
                    },
                    {
                    "id": "11e5e4867a6e3d8489020242ac110002",
                    "parentId": "11e0202427a6eac115e4863d83890002",
                    "name": "folderA"
                    }
                ]
                }
            ],
            objectErrors: [
                "objectId": "11e5e4867a6e3f8389020242ac110002",
                "code": 400,
                "error": "error in query",
                "msg": "not found",
            ]
            }

+ Response 400

        Unable to decode request
        
+ Response 403

        Forbidden

+ Response 500

        Error retrieving data

## Bulk Move Objects [/objects/move]

### Bulk Move Objects [POST]

Move a set of objects.  It requires the id and the change token for each one.

+ Request (application/json)

    + Body

            [
                {"id":"11e5e4867a6e3d8389020242ac110002", "changeToken":"e18919", "parentId":"11e5e4867aaa3d8389020242ac110002"},
                {"id":"11e5e4867a6f3d8389020242ac110002", "changeToken":"a38919", "parentId":"11e5e4867aaa3d8389020242ac110002"}
            ]

+ Response 200

    + Body

            [
                {"objectId":"11e5e4867a6e3d8389020242ac110002", "code":200},
                {"objectId":"11e5e4867a6f3d8389020242ac110002", "code":400, "error":"unable to find object", "msg":"cannot move object"}
            ]

## Bulk Change Owner [/objects/owner/{newOwner}]

+ Parameters
    + newOwner: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us` (string(maxlength=255), required) - A resource string compliant value representing the new owner. Resources take the following form:
       * {resourceType}/{serialized-representation}/{optional-display-name}
       * Examples for Users
         * user/{distinguishedName}/{displayName}
         * user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10
       * Examples for groups
         * group/{projectName}/{groupName}
         * group/{projectName}/{projectDisplayName}/{groupName}/{displayName}
         * group/dctc/odrive_g1
         * group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1

### Bulk Change Owner [POST]

This changes ownership of files in bulk.  It behaves like multiple changeOwner requests.

+ Request (application/json)

    We will supply a list of object ids and their change tokens.  We will get back an individual http code for each one.

    + Body

            {
                {"objectId":"22de09d2e0a09cbc0d90e", "changeToken":"ad90e90c9e245"}, 
                {"objectId":"42de09d2e0a09cbc0d90e", "changeToken":"cd90e90c9e245"} 
            }

+ Response 200

    The request itself is generally valid.  We will get a list of response codes and messages per ID from the original request.

    + Body

            {
                {"objectId":"22de09d2e0a09cbc0d90e","error":"","msg":"","code",200},
                {"objectId":"42de09d2e0a09cbc0d90e","error":"","msg":"","code",200}
            }  


## Zip of Objects [/zip]

+ objectIds (string array, required) - An array of object identifiers of files to be zipped.  
+ fileName (string, optional) - The name to give to the zip file.  Default to "drive.zip".
+ disposition (string, optional) - Either "inline" or "attachment", which is a hint to the browser for handling the result

### Zip of Objects [POST]

Create a zip of objects from a shopping cart
The UI will accumulate a list of file ID values to include in a zip file.

+ Request (application/json)

    The JSON object in the request body should contain a change token:

    + Attributes (CreateZipRequest)

+ Response 200

    + Headers

            Content-Type: application/zip
            Content-Disposition: {disposition}; filename={filename}

+ Response 400              

+ Response 500


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

+ users: `cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us`, `cn=test tester02,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us`, `cn=test tester03,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us`, `cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us` (array[string], optional) - Array of distinguished names for users that are targets of this share.
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

## ACMShareCreateGroupSample (object)

+ projects (array[ACMShareCreateSampleProjectsDCTC], optional) - Array of projects with nested groups that are targets of this share.

## ACMShareCreateUserSample (object)

+ users: `CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (array[string], optional) - Array of distinguished names for users that are targets of this share.

## Breadcrumb (object)

+ id: `11e5e4867a6e3d8489020242ac110002` (string) - The object ID of an object's breadcrumb. Should never be empty.
+ parentId: `11e0202427a6eac115e4863d83890002` (string) - The parent ID of an object's breadcrumb. Will be empty if a breadcrumb is a root object.
+ name: `folderA` (string) - The object name for an object's breadcrumb. Useful for displaying folder hierarchies.

## BreadcrumbParent (object)

+ id: `11e0202427a6eac115e4863d83890002` (string) - The object ID of an object's breadcrumb. Should never be empty.
+ parentId: ` ` (string) - The parent ID of an object's breadcrumb. Will be empty if a breadcrumb is a root object.
+ name: `parentFolderA` (string) - The object name for an object's breadcrumb. Useful for displaying folder hierarchies.

## CallerPermission (object)

+ allowCreate: false (boolean) -  Indicates whether the caller can create child objects under this object.
+ allowRead: true (boolean) -  Indicates whether the caller can view this object.
+ allowUpdate: false (boolean) -  Indicates whether the caller can modify this object.
+ allowDelete: false (boolean) -  Indicates whether the caller can delete this object.
+ allowShare: false (boolean) -  Indicates whether the caller can reshare this object.

## ChangeToken (object)

+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.

## ChangeOwnerRequest (object)

+ applyRecursively: `false` (boolean) - If true, the ownership change operation will apply to all of an object's descendants.
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.

## CreateObjectRequest (object)

+ typeName: `File` (string, optional) - The display name of the type assigned this object.
+ name: `gettysburgaddress.txt` (string, optional) - The name for this object. 
+ description: `Description here` (string, optional) - An abstract of the object's purpose.
+ parentId: ` ` (string, optional) - The unique identifier of the objects parent hex encoded to a string.  An empty value will result in this object being created in the user's root folder.
+ acm (ACM, required) - The acm value associated with this object in object form
+ permission (PermissionRequest, optional) - [1.1] The permissions associated with this object by capability and resource allowed.
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value should be empty.
+ contentSize: 1511 (number) - The length of the object's content stream, if present. For objects without a content stream, this value should be 0.
+ properties (array[PropertyCreate]) - Array of custom properties to be associated with the object.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.
+ permissions (array[PermissionUserCreate,PermissionGroupCreate]) - **Deprecated** - Array of permissions associated with this object.

## CreateObjectRequestNoStream (object)

+ typeName: `Folder` (string) - The display name of the type assigned this object.
+ name: `Famous Speeches` (string) - The name for this object. 
+ description: `Description here` (string) - An abstract of the object's purpose.
+ parentId: ` ` (string, optional) - The unique identifier of the objects parent hex encoded to a string.  An empty value will result in this object being created in the user's root folder.
+ acm (ACM, required) - The acm value associated with this object in object form
+ permission (PermissionRequest, optional) - [1.1] The permissions associated with this object by capability and resource allowed.
+ contentType: ` ` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value should be empty.
+ contentSize: 0 (number) - The length of the object's content stream, if present. For objects without a content stream, this value should be 0.
+ properties (array[PropertyCreate]) - Array of custom properties to be associated with the object.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.
+ permissions (array[PermissionUserCreate,PermissionGroupCreate]) - **Deprecated** - Array of permissions associated with this object.

## CreateZipRequest (object)

+ objectIds: `11e5e4867a6e3d8389020242ac110002`, `11e5e4867a6e11e5e48100026e3d8389` (array[string]) - The unique identifiers of objects to be bundled in the zip archive returned.
+ fileName: `drive.zip` (string) - The filename to be assigned the returned zip file by default.
+ disposition: `inline` (string) - The disposition setting for the response. Valid values are `inline` and `attachment` to direct browsers how to treat the file.

## DeleteObjectRequest (object)

+ id: `11e5e4867a6e3d8389020242ac110002`  (string, required) - The unique identifier of the object hex encoded to a string. 
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.

## GetObjectResponse (object)

+ id: `11e5e4867a6e3d8389020242ac110002`  (string, required) - The unique identifier of the object hex encoded to a string. This value can be used for alterations and listing on other RESTful methods.
+ createdDate: `2016-03-07T17:03:13.000001Z`  (string) - The date and time the object was created in the system in RFC3339 format. 
+ createdBy: `cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us` (string) - The user that created the object.
+ modifiedDate: `2016-03-07T17:03:13Z` (string) -  The date and time the object was last modified in the system in RFC3339 format. For unchanged objects, this will reflect the same value as the createdDate field.
+ modifiedBy: `cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us` (string) - The user that last modified this object. For unchanged objects, this will reflect the same value as the createdBy field.
+ deletedDate: `0001-01-01T00:00:00Z` (string, optional) -  The date and time the object was deleted in the system in RFC3339 format. This field is only populated if the object is in the trash.
+ deletedBy: `` (string) - The user that deleted the object. This field is only populated if the object is in the trash.
+ changeCount: 42 (number) - The total count of changes that have been made to this object over its lifespan. Synonymous with version number. For unchanged objects, this will always be 0.
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.
+ ownedBy: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us` (string) - The resource that owns this object. For a user, this is denoted with a prefix of `user/` followed by the user's distinguished name.  A group is prefixed by `group/` followed by project name, and group name.  Example group resource string: `group/dctc/odrive_g1` 
+ typeId: `11e5e48664f5d8c789020242ac110002` (string) - The unique identifier of the type assigned this object hex encoded to a string.
+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `gettysburgaddress.txt` (string) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
+ description: `Description here` (string) - An abstract of the object's purpose.
+ parentId: `11e5e4867a6e3d8489020242ac110002` (string, optional) - The unique identifier of the objects parent hex encoded to a string. This may be used to traverse up the tree. For objects stored at the root of a user, this value will be null.
+ acm (ACMResponse, required) - The acm value associated with this object in object form
+ permission (PermissionResponse, optional) - [1.1] The permissions associated with this object by capability and resource allowed.
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value will be null.
+ contentSize: 1511 (number) - The length of the object's content stream, if present. For objects without a content stream, this value will be 0.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.
+ properties (array[Property]) - Array of custom properties associated with the object.
+ callerPermissions (CallerPermission) - Permissions granted to the caller that resulted in this object being returned.
+ permissions (array[PermissionUser,PermissionGroup]) - **Deprecated** - Array of permissions associated with this object.
+ breadcrumbs (array[BreadcrumbParent,Breadcrumb]) - Array of IDs representing the parent chain for the object returned buy the API call. Will be empty for objects located at the root.

## GroupSpaceResp (object)

+ grantee: `dctc_odrive` (string) - The flattened group name identifier that matches f_share values in an ACM.
+ resourceString: `group/dctc/odrive` (string) - The resource string identifying the group suitable for use as the ownedBy value when creating new objects to be owned by the group.
+ displayName: `dctc odrive` (string) - A UI friendly representation of the resource string.
+ quantity: 3 (number) - The current number of objects owned by the group for which the caller is allowed to see at the root.

## GroupSpaceResultset (object)

+ totalRows: 100 (number) - Total number of groups the user is a member of that own objects at the root.
+ pageCount: 10 (number) - Always 1.
+ pageNumber: 1 (number) - Always 1.
+ pageSize: 10 (number) - Total number of groups the user is a member of that own objects at the root.
+ pageRows: 10 (number) - Total number of groups the user is a member of that own objects at the root.
+ groups (array[GroupSpaceResp]) - Array containing group information.

## MoveObjectRequest (object)

+ id: `11e5e4867a6e3d8389020242ac110002`  (string, required) - The unique identifier of the object hex encoded to a string. 
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.

## ObjectDeleted (object)

+ deletedDate: `2016-03-07T17:03:13Z` (string, optional) -  The date and time the object was deleted in the system in RFC3339 format. This field is only present if the object is in the trash.

## ObjectExpunged (object)

+ expungedDate: `2016-03-07T17:03:13Z` (string, optional) -  The date and time the object was expunged from the system in RFC3339 format. 

## ObjectResp (object)

+ id: `11e5e4867a6e3d8389020242ac110002`  (string, required) - The unique identifier of the object hex encoded to a string. This value can be used for alterations and listing on other RESTful methods.
+ createdDate: `2016-03-07T17:03:13Z`  (string) - The date and time the object was created in the system in RFC3339 format.
+ createdBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that created the object.
+ modifiedDate: `2016-03-07T17:03:13Z` (string) -  The date and time the object was last modified in the system in RFC3339 format. For unchanged objects, this will reflect the same value as the createdDate field.
+ modifiedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that last modified this object. For unchanged objects, this will reflect the same value as the createdBy field.
+ deletedDate: `0001-01-01T00:00:00Z` (string, optional) -  The date and time the object was deleted in the system in RFC3339 format. This field is only populated if the object is in the trash.
+ deletedBy: `` (string) - The user that deleted the object. This field is only populated if the object is in the trash.
+ changeCount: 42 (number) - The total count of changes that have been made to this object over its lifespan. Synonymous with version number. For unchanged objects, this will always be 0.
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.
+ ownedBy: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us` (string) - The resource that owns this object. For a user, this is denoted with a prefix of `user/` followed by the user's distinguished name.  A group is prefixed by `group/` followed by project name, and group name.  Example group resource string: `group/dctc/odrive_g1` 
+ typeId: `11e5e48664f5d8c789020242ac110002` (string) - The unique identifier of the type assigned this object hex encoded to a string.
+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `gettysburgaddress.txt` (string) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
+ description: `Description here` (string) - An abstract of the object's purpose.
+ parentId: ` ` (string, optional) - The unique identifier of the objects parent hex encoded to a string. This may be used to traverse up the tree. For objects stored at the root of a user, this value will be null.
+ acm (ACMResponse, required) - The acm value associated with this object in object form
+ permission (PermissionResponse, optional) - [1.1] The permissions associated with this object by capability and resource allowed.
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value will be null.
+ contentSize: 1511 (number) - The length of the object's content stream, if present. For objects without a content stream, this value will be 0.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.
+ properties (array[Property]) - Array of custom properties associated with the object.
+ callerPermissions (CallerPermission) - Permissions granted to the caller that resulted in this object being returned.
+ permissions (array[PermissionUser,PermissionGroup]) - **Deprecated** - Array of permissions associated with this object.

## ObjectRespChild1 (object)

+ id: `11e5e4867a6e3d8389020242ac110001`  (string, required) - The unique identifier of the object hex encoded to a string. This value can be used for alterations and listing on other RESTful methods.
+ createdDate: `2016-03-07T17:03:13Z`  (string) - The date and time the object was created in the system in RFC3339 format.
+ createdBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that created the object.
+ modifiedDate: `2016-03-07T17:03:13Z` (string) -  The date and time the object was last modified in the system in RFC3339 format. For unchanged objects, this will reflect the same value as the createdDate field.
+ modifiedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that last modified this object. For unchanged objects, this will reflect the same value as the createdBy field.
+ deletedDate: `0001-01-01T00:00:00Z` (string, optional) -  The date and time the object was deleted in the system in RFC3339 format. This field is only populated if the object is in the trash.
+ deletedBy: `` (string) - The user that deleted the object. This field is only populated if the object is in the trash.
+ changeCount: 42 (number) - The total count of changes that have been made to this object over its lifespan. Synonymous with version number. For unchanged objects, this will always be 0.
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.
+ ownedBy: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us` (string) - The resource that owns this object. For a user, this is denoted with a prefix of `user/` followed by the user's distinguished name.  A group is prefixed by `group/` followed by project name, project display name, and group name.  Example group resource string: `group/dctc/odrive_g1` 
+ typeId: `11e5e48664f5d8c789020242ac110002` (string) - The unique identifier of the type assigned this object hex encoded to a string.
+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `child 1` (string) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
+ description: `Description of object here` (string) - An abstract of the object's purpose.
+ parentId: `11e5e4867a6e3d8389020242ac11aaaa` (string, optional) - The unique identifier of the objects parent hex encoded to a string. This may be used to traverse up the tree. For objects stored at the root of a user, this value will be null.
+ acm (ACMResponse, required) - The acm value associated with this object in object form
+ permission (PermissionResponse, optional) - [1.1] The permissions associated with this object by capability and resource allowed.
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value will be null.
+ contentSize: 1511 (number) - The length of the object's content stream, if present. For objects without a content stream, this value will be 0.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.
+ properties (array[Property]) - Array of custom properties associated with the object.
+ callerPermissions (CallerPermission) - Permissions granted to the caller that resulted in this object being returned.
+ permissions (array[PermissionUser,PermissionGroup]) - **Deprecated** - Array of permissions associated with this object.

## ObjectRespChild2 (object)

+ id: `11e5e4867a6e3d8389020242ac110002`  (string, required) - The unique identifier of the object hex encoded to a string. This value can be used for alterations and listing on other RESTful methods.
+ createdDate: `2016-03-07T17:03:13Z`  (string) - The date and time the object was created in the system in RFC3339 format.
+ createdBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that created the object.
+ modifiedDate: `2016-03-07T17:03:13Z` (string) -  The date and time the object was last modified in the system in RFC3339 format. For unchanged objects, this will reflect the same value as the createdDate field.
+ modifiedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that last modified this object. For unchanged objects, this will reflect the same value as the createdBy field.
+ deletedDate: `0001-01-01T00:00:00Z` (string, optional) -  The date and time the object was deleted in the system in RFC3339 format. This field is only populated if the object is in the trash.
+ deletedBy: `` (string) - The user that deleted the object. This field is only populated if the object is in the trash.
+ changeCount: 42 (number) - The total count of changes that have been made to this object over its lifespan. Synonymous with version number. For unchanged objects, this will always be 0.
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.
+ ownedBy: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us` (string) - The resource that owns this object. For a user, this is denoted with a prefix of `user/` followed by the user's distinguished name.  A group is prefixed by `group/` followed by project name, and group name.  Example group resource string: `group/dctc/odrive_g1` 
+ typeId: `11e5e48664f5d8c789020242ac110002` (string) - The unique identifier of the type assigned this object hex encoded to a string.
+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `child 2` (string) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
+ description: `Description here` (string) - An abstract of the object's purpose.
+ parentId: `11e5e4867a6e3d8389020242ac11aaaa` (string, optional) - The unique identifier of the objects parent hex encoded to a string. This may be used to traverse up the tree. For objects stored at the root of a user, this value will be null.
+ acm (ACMResponse, required) - The acm value associated with this object in object form
+ permission (PermissionResponse, optional) - [1.1] The permissions associated with this object by capability and resource allowed.
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value will be null.
+ contentSize: 1511 (number) - The length of the object's content stream, if present. For objects without a content stream, this value will be 0.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.
+ properties (array[Property]) - Array of custom properties associated with the object.
+ callerPermissions (CallerPermission) - Permissions granted to the caller that resulted in this object being returned.
+ permissions (array[PermissionUser,PermissionGroup]) - **Deprecated** - Array of permissions associated with this object.


## ObjectRespDeleted (object)

+ id: `11e5e4867a6e3d8389020242ac110002`  (string, required) - string The unique identifier of the object hex encoded to a string. This value can be used for alterations and listing on other RESTful methods.
+ createdDate: `2016-03-07T17:03:13Z`  (string) - The date and time the object was created in the system in RFC3339 format.
+ createdBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that created the object.
+ modifiedDate: `2016-03-07T17:03:13Z` (string) -  The date and time the object was last modified in the system in RFC3339 format. For unchanged objects, this will reflect the same value as the createdDate field.
+ modifiedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that last modified this object. For unchanged objects, this will reflect the same value as the createdBy field.
+ deletedDate: `2016-03-07T17:03:13Z` (string, optional) -  The date and time the object was deleted in the system in RFC3339 format. This field is only present if the object is in the trash.
+ deletedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that deleted the object. This field is only present if the object is in the trash.
+ changeCount: 42 (number) - The total count of changes that have been made to this object over its lifespan. Synonymous with version number. For unchanged objects, this will always be 0.
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.
+ ownedBy: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us` (string) - The resource that owns this object. For a user, this is denoted with a prefix of `user/` followed by the user's distinguished name.  A group is prefixed by `group/` followed by project name, and group name.  Example group resource string: `group/dctc/odrive_g1` 
+ typeId: `11e5e48664f5d8c789020242ac110002` (string) - The unique identifier of the type assigned this object hex encoded to a string.
+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `gettysburgaddress.txt` (string) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
+ description: `Description here` (string) - An abstract of the object's purpose.
+ parentId: ` ` (string, optional) - The unique identifier of the objects parent hex encoded to a string. This may be used to traverse up the tree. For objects stored at the root of a user, this value will be null.
+ acm (ACMResponse, required) - The acm value associated with this object in object form
+ permission (PermissionResponse, optional) - [1.1] The permissions associated with this object by capability and resource allowed.
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value will be null.
+ contentSize: 1511 (number) - The length of the object's content stream, if present. For objects without a content stream, this value will be 0.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.
+ properties (array[Property]) - Array of custom properties associated with the object.
+ callerPermissions (CallerPermission) - Permissions granted to the caller that resulted in this object being returned.
+ permissions (array[PermissionUser,PermissionGroup]) - **Deprecated** - Array of permissions associated with this object.

## ObjectResultset (object)

+ totalRows: 100 (number) - Total number of items matching the query.
+ pageCount: 10 (number) - Total rows divided by page size.
+ pageNumber: 1 (number) - Requested page number for this resultset.
+ pageSize: 10 (number) - Requested page size for this resultset.
+ pageRows: 10 (number) - Number of items included in this page of the results, which may be less than pagesize, but never greater.
+ objects (array[ObjectResp]) - Array containing objects for this page of the resultset.

## ObjectResultsetChildren (object)

+ totalRows: 2 (number) - Total number of items matching the query.
+ pageCount: 1 (number) - Total rows divided by page size.
+ pageNumber: 1 (number) - Requested page number for this resultset.
+ pageSize: 10 (number) - Requested page size for this resultset.
+ pageRows: 2 (number) - Number of items included in this page of the results, which may be less than pagesize, but never greater.
+ objects (array[ObjectRespChild1,ObjectRespChild2]) - Array containing objects for this page of the resultset.

## ObjectResultsetDeleted (object)

+ totalRows: 2 (number) - Total number of items matching the query.
+ pageCount: 1 (number) - Total rows divided by page size.
+ pageNumber: 1 (number) - Requested page number for this resultset.
+ pageSize: 10 (number) - Requested page size for this resultset.
+ pageRows: 2 (number) - Number of items included in this page of the results, which may be less than pagesize, but never greater.
+ objects (array[ObjectRespDeleted]) - Array containing objects for this page of the resultset.

## ObjectShare (object)

+ share (ACMShare, optional) - **DEPRECATED** - The users and project/groups that will be granted read access to this object. If no share is specified, then the object is public.
+ allowCreate: `false` (boolean, optional) - **DEPRECATED** - Indicates whether the targets for the share can create child objects under the referenced object.
+ allowRead: `true` (boolean, optional) - **DEPRECATED** - Indicates whether the targets for the share can view the object referenced by this permission.
+ allowUpdate: `false` (boolean, optional) - **DEPRECATED** - Indicates whether the targets for the share can modify the object referenced by this permission.
+ allowDelete: `false` (boolean, optional) - **DEPRECATED** - Indicates whether the targets for the share can delete the object referenced by this permission.
+ allowShare: `false` (boolean, optional) - **DEPRECATED** - Indicates whether the targets for the share can reshare the object referenced by this permission.

## ObjectStorageMetric (object)

+ typeName: `File` (string) - The type of object, which is usually File or Folder.
+ objects: 24 (number) - The number of current objects that are stored.
+ objectsWithRevision: 432 (number) - The number of versioned objects that are stored.
+ objectsSize: 249234 (number) - The total size of objects in bytes, which could be a very large number.
+ objectsWithRevisionSize: 23478234 (number) - The total size of versioned objects in bytes, which may be very large.

## PermissionRequest (object)

+ create (PermissionCapabilityRequestCreate, optional) - The permission to create child objects beneath this object.
+ read (PermissionCapabilityRequestRead, optional) - The permission to read the metadata, properties, content stream, or list children of this object.
+ update (PermissionCapabilityRequestUpdate, optional) - The permission to modify the metadata, properties or content stream of this object.
+ delete (PermissionCapabilityRequestDelete, optional) - The permission to delete, undelete, or expunge forever this object.
+ share (PermissionCapabilityRequestShare, optional) - The permission to alter the share settings for this object, and delegate sharing capabilities to others.

## PermissionCapabilityRequestCreate (object)

+ allow: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10` (array[string], optional) - The list of resources allowed to perform this capability
+ deny: `` (array[string], optional) - The list of resources denied this capability

## PermissionCapabilityRequestRead (object)

+ allow: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10`, `group/dctc/odrive_g1` (array[string], optional) - The list of resources allowed to perform this capability
+ deny: `` (array[string], optional) - The list of resources denied this capability

## PermissionCapabilityRequestUpdate (object)

+ allow: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10` (array[string], optional) - The list of resources allowed to perform this capability
+ deny: `` (array[string], optional) - The list of resources denied this capability

## PermissionCapabilityRequestDelete (object)

+ allow: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10` (array[string], optional) - The list of resources allowed to perform this capability
+ deny: `` (array[string], optional) - The list of resources denied this capability

## PermissionCapabilityRequestShare (object)

+ allow: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10` (array[string], optional) - The list of resources allowed to perform this capability
+ deny: `` (array[string], optional) - The list of resources denied this capability

## PermissionResponse (object)

+ create (PermissionCapabilityResponseCreate, optional) - The permission to create child objects beneath this object.
+ read (PermissionCapabilityResponseRead, optional) - The permission to read the metadata, properties, content stream, or list children of this object.
+ update (PermissionCapabilityResponseUpdate, optional) - The permission to modify the metadata, properties or content stream of this object.
+ delete (PermissionCapabilityResponseDelete, optional) - The permission to delete, undelete, or expunge forever this object.
+ share (PermissionCapabilityResponseShare, optional) - The permission to alter the share settings for this object, and delegate sharing capabilities to others.

## PermissionCapabilityResponseCreate (object)

+ allow: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10` (array[string], optional) - The list of resources allowed to perform this capability
+ deny: `` (array[string], optional) - The list of resources denied this capability

## PermissionCapabilityResponseRead (object)

+ allow: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10`, `group/dctc/odrive_g1` (array[string], optional) - The list of resources allowed to perform this capability
+ deny: `` (array[string], optional) - The list of resources denied this capability

## PermissionCapabilityResponseUpdate (object)

+ allow: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10` (array[string], optional) - The list of resources allowed to perform this capability
+ deny: `` (array[string], optional) - The list of resources denied this capability

## PermissionCapabilityResponseDelete (object)

+ allow: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10` (array[string], optional) - The list of resources allowed to perform this capability
+ deny: `` (array[string], optional) - The list of resources denied this capability

## PermissionCapabilityResponseShare (object)

+ allow: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10` (array[string], optional) - The list of resources allowed to perform this capability
+ deny: `` (array[string], optional) - The list of resources denied this capability

## Permission (object)

+ create (PermissionCapability, optional) - The permission to create child objects beneath this object.
+ read (PermissionCapability, optional) - The permission to read the metadata, properties, content stream, or list children of this object.
+ update (PermissionCapability, optional) - The permission to modify the metadata, properties or content stream of this object.
+ delete (PermissionCapability, optional) - The permission to delete, undelete, or expunge forever this object.
+ share (PermissionCapability, optional) - The permission to alter the share settings for this object, and delegate sharing capabilities to others.

## PermissionCapability (object)

+ allow (array[string], optional) - The list of resources allowed to perform this capability
+ deny (array[string], optional) - The list of resources denied this capability

## PermissionCreate (object)

+ share (ACMShareCreateSample) - The share structure for this permission representing one or more targets to be granted the permissions
+ allowCreate: false (boolean) -  Indicates whether the grantee can create child objects under the referenced object of this permission.
+ allowRead: true (boolean) -  Indicates whether the grantee can view the object referenced by this permission.
+ allowUpdate: true (boolean) -  Indicates whether the grantee can modify the object referenced by this permission.
+ allowDelete: false (boolean) -  Indicates whether the grantee can delete the object referenced by this permission.
+ allowShare: false (boolean) -  Indicates whether the grantee can reshare the object referenced by this permission.

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

## PermissionGroupCreate (object)

+ share (ACMShareCreateGroupSample) - The share structure for this permission representing one or more targets to be granted the permissions
+ allowCreate: false (boolean) -  Indicates whether the grantee can create child objects under the referenced object of this permission.
+ allowRead: true (boolean) -  Indicates whether the grantee can view the object referenced by this permission.
+ allowUpdate: false (boolean) -  Indicates whether the grantee can modify the object referenced by this permission.
+ allowDelete: false (boolean) -  Indicates whether the grantee can delete the object referenced by this permission.
+ allowShare: false (boolean) -  Indicates whether the grantee can reshare the object referenced by this permission.

## PermissionUser (object)

+ grantee: `cntesttester10oupeopleoudaeouchimeraou_s_governmentcus` (string) -  The flattened form of the user or group this permission targets
+ userDistinguishedName: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user for whom this permission is granted to
+ displayName: `test tester10` (string) - A representation of the grantee suitable for display in user interfaces
+ allowCreate: true (boolean) -  Indicates whether the grantee can create child objects under the referenced object of this permission.
+ allowRead: true (boolean) -  Indicates whether the grantee can view the object referenced by this permission.
+ allowUpdate: true (boolean) -  Indicates whether the grantee can modify the object referenced by this permission.
+ allowDelete: true (boolean) -  Indicates whether the grantee can delete the object referenced by this permission.
+ allowShare: true (boolean) -  Indicates whether the grantee can reshare the object referenced by this permission.

## PermissionUserCreate (object)

+ share (ACMShareCreateUserSample) - The share structure for this permission representing one or more targets to be granted the permissions
+ allowCreate: false (boolean) -  Indicates whether the grantee can create child objects under the referenced object of this permission.
+ allowRead: true (boolean) -  Indicates whether the grantee can view the object referenced by this permission.
+ allowUpdate: true (boolean) -  Indicates whether the grantee can modify the object referenced by this permission.
+ allowDelete: false (boolean) -  Indicates whether the grantee can delete the object referenced by this permission.
+ allowShare: false (boolean) -  Indicates whether the grantee can reshare the object referenced by this permission.

## Property (object)

+ id: `11e5e4867a6e3d8389020242ac110002` (string) - The unique identifier of the property associated to the object hex encoded to a string.
+ createdDate: `2016-03-07T17:03:13Z` (string) - The date and time the property was created in the system in RFC3339 format.
+ createdBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) -  The user that created the property .
+ modifiedDate: `2016-03-07T17:03:13Z` (string) - The date and time the property was last modified in the system in RFC3339 format.
+ modifiedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) -  The user that last modified this property .
+ changeCount: 1 (number) - The total count of changes that have been made to this property over its lifespan. Synonymous with version number.
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) -  A hash of the property's unique identifier and last modification date and time.
+ name: `Some Property` (string) - The name, key, field or label given to a property for usability
+ value: `Some Property Value` (string) -  The value assigned for the property
+ classificationPM: `U` (string) -  The portion mark classification for the value of this property

## PropertyCreate (object)

+ name: `Some Property` (string) - The name, key, field or label given to a property for usability
+ value: `Some Property Value` (string) -  The value assigned for the property
+ classificationPM: `U//FOUO` (string) -  The portion mark classification for the value of this property

## UpdateObject (object)

+ id: `11e5e4867a6e3d8389020242ac110002` (string, required) - The unique identifier of the object hex encoded to a string. 
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - The current change token on the object
+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `gettysburgaddress.txt` (string) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
+ description: `Description here` (string) - An abstract of the object's purpose.
+ acm (ACM, optional) - The acm value associated with this object in object form. If not provided, the current ACM on the object will be retained.
+ permission (PermissionRequest, optional) - [1.1] The permissions to be associated with this object, replacing existing permissions.
+ properties (array[Property]) - Array of custom properties associated with the object. New properties will be added. Properties that have the same name as existing properties will be replaced. Those with an empty value will be deleted.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.

## UserStats (object)

+ totalObjects: 24 (number) - The number of current objects that are stored.
+ totalObjectsWithRevision: 432 (number) - The number of versioned objects that are stored.
+ totalObjectsSize: 249234 (number) - The total size of objects in bytes, which could be a very large number.
+ totalObjectsWithRevisionSize: 23478234 (number) - The total size of versioned objects in bytes, which may be very large.
+ objectStorageMetrics: ObjectStorageMetric (array[ObjectStorageMetric]) - An array of ObjectStorageMetrics denoting the type of object, quantity of base object and revisions, and size used by base object and revision.
