FORMAT: 1A

# Object Drive Microservice API
A series of microservice operations are exposed on the API gateway for use of Object Drive. 

[Change Log](static/temmplates/changelog.html)


## Summary of Operations Available

A listing of microservice operations is summarized in the table below.

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
| User Stats | Retrieve information for user's storage consumtpion. |
| Zip Files | Get a zip of some files. |


##  Reference Examples

Detailed code examples that use the API:

[Java Caller (create an object)](static/templates/ObjectDriveSDK.java)

[Javascript Caller (our simple test user interface)](static/templates/listObjects.js)

The http level result of calling APIs that happens inside of SSL:

[Actual Traffic - Basic Operations](static/templates/APISample.html)

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

    + typeName (string, required) -  The type to be assigned to this object.  Custom types may be referenecd here for the purposes of rules or processing constraints and definition of properties.
       * File - This type may be assigned if no type is given, and you are creating an object with a stream
       * Folder - This type may be assigned if no type is given, and you are creating an object without a stream
    + name: `New File` (string, optional) - The name to be given this object.  If no name is given, then objects are created with the default name pattern of `New <typeName>`.
    + description (string, optional) - An optional abstract of the object's contents.
    + parentId (string, optional) - Hex encoded identifier of an object, typically a folder, into which this new object is being created as a child object. If no value is specified, then the object will be created in the root location of the user who is creating it.
    + acm (object, required) - Access Control Model is the security model leveraged by the system when enforcing access control. It is based on the ISM, NTK, ACCM and Share standards, requirements and policies.  This value may be provided in either serialized string format, or nested object format.
    + permission (PermissionRequest, optional) - [1.1] The permissions associated with this object by capability and resource allowed.  Resources take the following form:
       * {resourceType}/{serialized-representation}/{optional-display-name}
       * Examples for Users
         * user/{distinguishedName}/{displayName}
         * user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10
       * Examples for groups
         * group/{projectName}/{projectDisplayName}/{groupName}/{displayName}
         * group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1
         * group/-Everyone
    + contentType: `text/plain` (string, optional) - The suggested mime type for the content stream if given for this object.
    + contentSize: 0 (int, optional) - The length of the content stream, in bytes. If there is no content stream, this value should be 0.
    + containsUSPersonsData: `Yes` (string, optional) - Indicates if this object contains US Persons data.
        + Default: `Unknown`  
        + Members
            + `Yes`
            + `No`
            + `Unknown`
    + exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  
        + Default: `Unknown`  
        + Members
            + `Yes`
            + `No`
            + `Unknown`
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
              "contentSize": "1511",
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

    + typeName (string, required) -  The type to be assigned to this object.  Custom types may be referenecd here for the purposes of rules or processing constraints and definition of properties.
       * File - This type may be assigned if no type is given, and you are creating an object with a stream
       * Folder - This type may be assigned if no type is given, and you are creating an object without a stream
    + name: `New Folder` (string, optional) - The name to be given this object.  If no name is given, then objects are created with the default name pattern of `New <typeName>`.
    + description (string, optional) - An optional abstract of the object's contents.
    + parentId (string, optional) - Hex encoded identifier of an object, typically a folder, into which this new object is being created as a child object. If no value is specified, then the object will be created in the root location of the user who is creating it.
    + acm (object, required) - Access Control Model is the security model leveraged by the system when enforcing access control. It is based on the ISM, NTK, ACCM and Share standards, requirements and policies.  This value may be provided in either serialized string format, or nested object format.
    + permission (PermissionRequest, optional) - [1.1] The permissions associated with this object by capability and resource allowed.  Resources take the following form:
       * {resourceType}/{serialized-representation}/{optional-display-name}
       * Examples for Users
         * user/{distinguishedName}/{displayName}
         * user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10
       * Examples for groups
         * group/{projectName}/{projectDisplayName}/{groupName}/{displayName}
         * group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1
         * group/-Everyone
    + contentType: `` (string, optional) - The suggested mime type for the content stream if given for this object.
    + contentSize: 0 (int, optional) - The length of the content stream, in bytes. If there is no content stream, this value should be 0.
    + containsUSPersonsData: `Yes` (string, optional) - Indicates if this object contains US Persons data.
        + Default: `Unknown`  
        + Members
            + `Yes`
            + `No`
            + `Unknown`
    + exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  
        + Default: `Unknown`  
        + Members
            + `Yes`
            + `No`
            + `Unknown`
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

### Delete Objects [DELETE]

Delete a set of objects.  It requires the id and the change token for each one.

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


## Bulk object properties [/objects/properties]

### Get bulk object properties [GET]
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
                "parentId": "",
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
                        "group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1"
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
                "contentSize": "1511",
                "isPDFAvailable": false,
                "containsUSPersonsData": "No",
                "exemptFromFOIA": "No",
                "properties": [
                    {
                    "id": "11e5e4867a6e3d8389020242ac110002",
                    "createdDate": "2016-03-07T17:03:13Z",
                    "createdBy": "CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                    "modifiedDate": "2016-03-07T17:03:13Z",
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
                    "projectDisplayName": "DCTC",
                    "groupName": "ODrive_G1",
                    "displayName": "DCTC ODrive_G1",
                    "allowCreate": true,
                    "allowRead": true,
                    "allowUpdate": true,
                    "allowDelete": true,
                    "allowShare": true
                    }
                ],
                "breadcrumbs": [
                    {
                    "id": "11e5e4867a6e3d8389020242ac110002",
                    "parentId": "",
                    "name": "parentFolderA"
                    },
                    {
                    "id": "11e5e4867a6e3d8389020242ac110002",
                    "parentId": "11e5e4867a6e3d8389020242ac110002",
                    "name": "folderA"
                    }
                ]
                },{
                "id": "11e5e4867a6e3d8389020242ac189124",
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
                "parentId": "",
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
                        "group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1"
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
                "contentSize": "1511",
                "isPDFAvailable": false,
                "containsUSPersonsData": "No",
                "exemptFromFOIA": "No",
                "properties": [
                    {
                    "id": "11e5e4867a6e3d8389020242ac189124",
                    "createdDate": "2016-03-07T17:03:13Z",
                    "createdBy": "CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US",
                    "modifiedDate": "2016-03-07T17:03:13Z",
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
                    "projectDisplayName": "DCTC",
                    "groupName": "ODrive_G1",
                    "displayName": "DCTC ODrive_G1",
                    "allowCreate": true,
                    "allowRead": true,
                    "allowUpdate": true,
                    "allowDelete": true,
                    "allowShare": true
                    }
                ],
                "breadcrumbs": [
                    {
                    "id": "11e5e4867a6e3d8389020242ac189124",
                    "parentId": "",
                    "name": "parentFolderA"
                    },
                    {
                    "id": "11e5e4867a6e3d8389020242ac110002",
                    "parentId": "11e5e4867a6e3d8389020242ac110002",
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

## Object Metadata [/objects/{objectId}/properties]

Metadata for an object may be retrieved or updated at the URI designated.  

+ Parameters
    + objectId: `11e5e48664f5d8c789020242ac110002` (string, required) - string Hex encoded identifier of the object to be retrieved.

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

    + changeToken (string, required) - The current change token on the object
    + typeName: `Folder` (string, optional) -  The type to be assigned to this object.  During update if no typeName is given, then the existing type will be retained
    + name (string, optional) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
    + description (string, optional) - The new description to be given as an abstract of the objects content stream. If no value is provided, or this field is ommitted, then the description will not be changed.
    + acm (object, optional) -  Access Control Model (ACM) is the security model leveraged by the system when enforcing access control. It is based on the ISM, NTK, ACCM and Share standards, requirements and policies. https://confluence.363-283.io/pages/viewpage.action?pageId=557850. If no value is provided, or this field is ommitted, then the acm will not be changed. This value may be provided in either serialized string format, or nested object format.
    + permission (PermissionRequest, optional) - [1.1] The permissions associated with this object by capability and resource allowed.  Resources take the following form:
       * {resourceType}/{serialized-representation}/{optional-display-name}
       * Examples for Users
         * user/{distinguishedName}/{displayName}
         * user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10
       * Examples for groups
         * group/{projectName}/{projectDisplayName}/{groupName}/{displayName}
         * group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1
         * group/-Everyone
    + containsUSPersonsData: `Yes` (string, optional) - Indicates if this object contains US Persons data.
        + Default: `Unknown`  
        + Members
            + `Yes`
            + `No`
            + `Unknown`
    + exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  
        + Default: `Unknown`  
        + Members
            + `Yes`
            + `No`
            + `Unknown`
    + properties (properties array, optional) -  An array of custom properties to be associated with this object for property changes. For the properties specified, those who do not match existing properties on the object by name will be added. For the properties that do match existing properties by name, if the value specified is blank or empty, then the existing property will be deleted, otherwise, the property will be updated to the new value. If properties are specified in the array, then existing properties on the object are retained. Properties are only removed from an object if they are provided, with their value set to an empty string.

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

The content stream for an object may be retrieved or updated at the URI designated.

+ Parameters
    + objectId: `11e5e48664f5d8c789020242ac110002` (string, required) - Hex encoded identifier of the object to be retrieved.
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
    + objectId: `11e5e48664f5d8c789020242ac110002` (string, required) - Hex encoded identifier of the object to be retrieved.

### Update an Object Stream [POST]

This creates a new revision of the object.
    
+ Request (multipart/form-data; boundary=b428e6cd1933)

    The JSON object provided in the body can contain the following fields:

    + id: `11e5e48664f5d8c789020242ac110002` (string, required) - The unique identifier of the object hex encoded to a string. This value must match the objectId provided in the URI.
    + changeToken (string, required) - A hash value expected to match the targeted object’s current changeToken value. This value is retrieved from get or list operations.
    + typeName (string, optional) -  The new type to be assigned to this object. Common types include 'File', 'Folder'. If no value is provided or this field is omitted, then the type will not be changed.
    + name (string, optional) - The new name to be given this object. It does not have to be unique. It may refer to a conventional filename and extension. If no value is provided, or this field is ommitted, then the name will not be changed.
    + description (string, optional) - The new description to be given as an abstract of the objects content stream. If no value is provided, or this field is ommitted, then the description will not be changed.
    + acm (string OR object, optional) -  Access Control Model (ACM) is the security model leveraged by the system when enforcing access control. It is based on the ISM, NTK, ACCM and Share standards, requirements and policies. https://confluence.363-283.io/pages/viewpage.action?pageId=557850. If no value is provided, or this field is ommitted, then the acm will not be changed.  This value may be provided in either serialized string format, or nested object format.
    + permission (PermissionRequest, optional) - [1.1] The permissions associated with this object by capability and resource allowed.  Resources take the following form:
       * {resourceType}/{serialized-representation}/{optional-display-name}
       * Examples for Users
         * user/{distinguishedName}/{displayName}
         * user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10
       * Examples for groups
         * group/{projectName}/{projectDisplayName}/{groupName}/{displayName}
         * group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1
         * group/-Everyone
    + contentType: `text/html` (string, optional) - The suggested mime type for the content stream if given for this object.
    + contentSize: 0 (int, optional) - The length of the content stream, in bytes. If there is no content stream, this value should be 0.
    + containsUSPersonsData: `Yes` (string, optional) - Indicates if this object contains US Persons data.
        + Default: `Unknown`  
        + Members
            + `Yes`
            + `No`
            + `Unknown`
    + exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  
        + Default: `Unknown`  
        + Members
            + `Yes`
            + `No`
            + `Unknown`
    + properties (properties array, optional) -  An array of custom properties to be associated with this object for property changes. For the properties specified, those who do not match existing properties on the object by name will be added. For the properties that do match existing properties by name, if the value specified is blank or empty, then the existing property will be deleted, otherwise, the property will be updated to the new value. If properties are specified in the array, then existing properties on the object are retained. Properties are only removed from an object if they are provided, with their value set to an empty string.
           
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

## Move Objects [/objects/move]

### Move Objects [POST]

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

## Delete Object [/objects/{objectId}/trash]

+ Parameters
    + objectId: `11e5e4867a6e3d8389020242ac110002` (string, required) - Hex encoded identifier of the object to be deleted.

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

        Forbidden
        
+ Response 405

        Deleted

+ Response 410

        Does Not Exist
        
+ Response 500

        Error storing metadata or stream

## Delete Object Forever [/objects/{objectId}]  

+ Parameters
     + objectId: `11e5e4867a6e3d8389020242ac110002` (string, required) - Hex encoded identifier of the object to be deleted.

### Delete Object Forever [DELETE]
This microservice operation will remove an object from the trash and delete it forever.  

+ Request (application/json)

    + Attributes (ChangeToken)

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

## List Object Revisions [/revisions/{objectId}{?pageNumber,pageSize,sortField,sortAscending}]

+ Parameters
    + objectId: `11e5e4867a6e3d8389020242ac110002` (string, required) - Hex encoded identifier of the object for which revisions are being requested.
    + pageNumber: 1 (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number, optional) - The number of results to return per page.
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
    + sortAscending: false (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: true



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


## Get Object Stream Revision [/revisions/{objectId}/{revisionId}/stream{?disposition}]

+ Parameters
    + objectId: `11e5e4867a6e3d8389020242ac110002` (string, required) - Hex encoded identifier of the object to be retrieved.
    + revisionId: 2 (number, required) - The revision number to be retrieved. 
    + disposition: `attachment` (number, optional) - The value to assign the Content-Disposition in the response.
        + Default: `inline`
        + Members
            + `inline` - The default disposition
            + `attachment` - Supports browser prompting the user to save the response as a file.

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

        If the user is forbidden to perform the request because they lack permissions to view the object.
        
+ Response 405

        If the object or an ancestor is deleted.
        
+ Response 410

        If the object referenced no longer exists
        
+ Response 500

        * Malformed JSON
        * Error retrieving object
        * Error determining user.`

# Group Search & List Operations

---

## Search [/search/{searchPhrase}{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

**EXPERIMENTAL** - Search operations are an experimental feature

+ Parameters
    + searchPhrase: `image/gif` (string, required) - The phrase to look for inclusion within the name or description of objects. This will be overridden if parameters for filterField are set.
    + pageNumber: 1 (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number, optional) - The number of results to return per page.
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
    + sortAscending: false (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: true
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition)
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
            + `equals`
            + `contains`
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

## List Objects At Root For Group [/groupobjects/{groupName}{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

+ Parameters

    + groupName: dctc_odrive_g1 (string, required) - The flattened name of a group for which the user is a member and objects owned by the group should be returned.
        * The flattened values for user identity are also acceptable
        * Psuedogroups, such as `_everyone` are not acceptable for this request, but are forbidden from owning objects anyway.
    + pageNumber: 1 (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number, optional) - The number of results to return per page.
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
    + sortAscending: false (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: true
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition)
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
            + `equals`
            + `contains`
    + expression: `0` (string, optional) - **experimental** - A phrase that should be used for the match against the field value

### List Objects At Root For Group [GET]

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

## List Objects Under Parent [/objects/{objectId}{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

+ Parameters
    + objectId: `11e5e4867a6e3d8389020242ac110002` (string, required) - Hex encoded unique identifier of the folder or other object for which to return a list of child objects. 
    + pageNumber: 1 (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number, optional) - The number of results to return per page.
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
    + sortAscending: false (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: true
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition)
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
            + `equals`
            + `contains`
    + expression: `0` (string, optional) - **experimental** - A phrase that should be used for the match against the field value

### List Object Under Parent [GET]
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

## List Objects Shared to Everyone [/sharedpublic{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

+ Parameters
    + pageNumber: 1 (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number, optional) - The number of results to return per page.
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
    + sortAscending: false (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: true
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition)
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
            + `equals`
            + `contains`
    + expression: `0` (string, optional) - **experimental** - A phrase that should be used for the match against the field value
    
### List Objects Shared to Everyone [GET]

This microservice operation retrieves a list of objects that are shared to everyone, but excludes those whose immediate parent is also shared to everyone, thus providing contextual root shares.

+ Request

    + Header
    
            Content-Type: application/json

+ Response 200
    + Attributes (ObjectResultset)

+ Response 400

        Unable to decode request
        
+ Response 500

        Error storing metadata or stream


# Group Filing Operations

---

## Move Object [/objects/{objectId}/move/{folderId}]

+ Parameters
    + objectId: `11e5e4867a6e3d8389020242ac110002` (string, required) - Hex encoded identifier of the object to be moved.
    + folderId: `30211e5e48ac110067a6e3d802420289` (string, optional) - Hex encoded identifier of the folder into which this object should be moved.  If no identifier is provided, then the object will be moved to the owner's root folder.

### Move Object [POST]
This microservice operation supports moving an object such as a file or folder from one location to another. By default, all objects are created in the ‘root’ as they have no parent folder given.

This creates a new revision of the object.

Only the owner of an object is allowed to move it.

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

## Change Owner [/objects/{objectId}/owner/{newOwner}]

+ Parameters
    + objectId: `11e5e4867a6e3d8389020242ac110002` (string, required) - Hex encoded identifier of the object to be moved.
    + newOwner: `group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1` (string, required) - A resource string compliant value representing the new owner. Resources take the following form:
       * {resourceType}/{serialized-representation}/{optional-display-name}
       * Examples for Users
         * user/{distinguishedName}/{displayName}
         * user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10
       * Examples for groups
         * group/{projectName}/{projectDisplayName}/{groupName}/{displayName}
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

    The JSON object in the request body should contain a change token:

    + Attributes (ChangeToken)

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

## Change Owner [/objects/owner/{newOwner}]

+ Parameters
    + newOwner: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us`

### Change Owner [POST]

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

# Group User Centric Operations

---

## List Objects At Root For User [/objects{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

+ Parameters

    + pageNumber: 1 (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number, optional) - The number of results to return per page.
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
    + sortAscending: false (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: true
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition)
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
            + `equals`
            + `contains`
    + expression: `0` (string, optional) - **experimental** - A phrase that should be used for the match against the field value

### List Objects At Root For User [GET]

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

## List User Object Shares [/shares{?pageNumber,pageSize,sortField,sortAscending,filterMatchType,filterField,condition,expression}]

+ Parameters
    + pageNumber: 1 (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number, optional) - The number of results to return per page.
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
    + sortAscending: false (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: true
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition)
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
            + `equals`
            + `contains`
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
    + pageNumber: 1 (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number, optional) - The number of results to return per page.
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
    + sortAscending: false (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: true
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition)
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
            + `equals`
            + `contains`
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
    + pageNumber: 1 (number, optional) - The page number of results to be returned to support chunked output.
    + pageSize: 20 (number, optional) - The number of results to return per page.
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
    + sortAscending: false (boolean, optional) - Indicates whether to sort in ascending or descending order. If not provided, the default is false.
        + Default: true
    + filterMatchType: `and` (string, optional) - **experimental** - Allows for overriding default filter to require either all or any filters match.
        + Default: `or`
        + Members
            + `all`
            + `and`
            + `any`
            + `or`
    + filterField: `changecount` (string, optional) - **experimental** - Denotes a field that the results should be filtered on. Can be specified multiple times. If filterField is set, condition and expression must also be set to complete the tupled filter query.  Multiple filters act as a union, joining combined sets (OR condition) as opposed to requiring all filters be met as exclusionary (AND condition)
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
            + `equals`
            + `contains`
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
    + pageSize: 10000 (number, optional) - The batch size to expunge objects in

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


# Group Auxillary Operations

---

## Create Zip of objects [/zip]

+ objectIds (string array, required) - An array of object identifiers of files to be zipped.  
+ fileName (string, optional) - The name to give to the zip file.  Default to "drive.zip".
+ disposition (string, optional) - Either "inline" or "attachment", which is a hint to the browser for handling the result

### Create Zip of objects [POST]

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

## ACMShareCreateGroupSample (object)

+ projects (array[ACMShareCreateSampleProjectsDCTC], optional) - Array of projects with nested groups that are targets of this share.

## ACMShareCreateUserSample (object)

+ users: `CN=test tester01,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (array[string], optional) - Array of distinguished names for users that are targets of this share.

## Breadcrumb (object)

+ id: `11e5e4867a6e3d8389020242ac110002` (string) - The object ID of an object's breadcrumb. Should never be empty.
+ parentId: `11e5e4867a6e3d8389020242ac110002` (string) - The parent ID of an object's breadcrumb. Will be empty if a breadcrumb is a root object.
+ name: `folderA` (string) - The object name for an object's breadcrumb. Useful for displaying folder hierarchies.

## BreadcrumbParent (object)

+ id: `11e5e4867a6e3d8389020242ac110002` (string) - The object ID of an object's breadcrumb. Should never be empty.
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

## CreateObjectRequest (object)

+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `gettysburgaddress.txt` (string) - The name for this object. 
+ description: `Description here` (string) - An abstract of the object's purpose.
+ parentId: ` ` (string, optional) - The unique identifier of the objects parent hex encoded to a string.  An empty value will result in this object being created in the user's root folder.
+ acm (ACM, required) - The acm value associated with this object in object form
+ permission (PermissionRequest, optional) - [1.1] The permissions associated with this object by capability and resource allowed.
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value should be empty.
+ contentSize: 1511 (string) - The length of the object's content stream, if present. For objects without a content stream, this value should be 0.
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
+ contentSize: 0 (string) - The length of the object's content stream, if present. For objects without a content stream, this value should be 0.
+ properties (array[PropertyCreate]) - Array of custom properties to be associated with the object.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.
+ permissions (array[PermissionUserCreate,PermissionGroupCreate]) - **Deprecated** - Array of permissions associated with this object.

## CreateZipRequest (object)

+ objectIds: `11e5e4867a6e3d8389020242ac110002`, `11e5e4867a6e11e5e48100026e3d8389` (array[string]) - The unique identifiers of objects to be bundled in the zip archive returned.
+ fileName: `drive.zip` (string) - The filename to be assigned the returned zip file by default.
+ disposition: `inline` (string) - The disposition setting for the response. Valid values are `inline` and `attachment` to direct browsers how to treat the file.

## GetObjectResponse (object)

+ id: `11e5e4867a6e3d8389020242ac110002`  (string, required) - The unique identifier of the object hex encoded to a string. This value can be used for alterations and listing on other RESTful methods.
+ createdDate: `2016-03-07T17:03:13Z`  (string) - The date and time the object was created in the system in UTC ISO 8601 format.
+ createdBy: `cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us` (string) - The user that created the object.
+ modifiedDate: `2016-03-07T17:03:13Z` (string) -  The date and time the object was last modified in the system in UTC ISO 8601 format. For unchanged objects, this will reflect the same value as the createdDate field.
+ modifiedBy: `cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us` (string) - The user that last modified this object. For unchanged objects, this will reflect the same value as the createdBy field.
+ deletedDate: `0001-01-01T00:00:00Z` (string, optional) -  The date and time the object was deleted in the system in UTC ISO 8601 format. This field is only populated if the object is in the trash.
+ deletedBy: `` (string) - The user that deleted the object. This field is only populated if the object is in the trash.
+ changeCount: 42 (number) - The total count of changes that have been made to this object over its lifespan. Synonymous with version number. For unchanged objects, this will always be 0.
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.
+ ownedBy: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us` (string) - The resource that owns this object. For a user, this is denoted with a prefix of `user/` followed by the user's distinguished name.  A group is prefixed by `group/` followed by project name, project display name, and group name.  Example group resource string: `group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1` 
+ typeId: `11e5e48664f5d8c789020242ac110002` (string) - The unique identifier of the type assigned this object hex encoded to a string.
+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `gettysburgaddress.txt` (string) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
+ description: `Description here` (string) - An abstract of the object's purpose.
+ parentId: ` ` (string, optional) - The unique identifier of the objects parent hex encoded to a string. This may be used to traverse up the tree. For objects stored at the root of a user, this value will be null.
+ acm (ACMResponse, required) - The acm value associated with this object in object form
+ permission (PermissionResponse, optional) - [1.1] The permissions associated with this object by capability and resource allowed.
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value will be null.
+ contentSize: 1511 (string) - The length of the object's content stream, if present. For objects without a content stream, this value will be 0.
+ isPDFAvailable: `false` (boolean) - Indicates if a PDF rendition is available for this object.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.
+ properties (array[Property]) - Array of custom properties associated with the object.
+ callerPermissions (CallerPermission) - Permissions granted to the caller that resulted in this object being returned.
+ permissions (array[PermissionUser,PermissionGroup]) - **Deprecated** - Array of permissions associated with this object.
+ breadcrumbs (array[BreadcrumbParent,Breadcrumb]) - Array of IDs representing the parent chain for the object returned buy the API call. Will be empty for objects located at the root.

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
+ ownedBy: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us` (string) - The resource that owns this object. For a user, this is denoted with a prefix of `user/` followed by the user's distinguished name.  A group is prefixed by `group/` followed by project name, project display name, and group name.  Example group resource string: `group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1` 
+ typeId: `11e5e48664f5d8c789020242ac110002` (string) - The unique identifier of the type assigned this object hex encoded to a string.
+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `gettysburgaddress.txt` (string) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
+ description: `Description here` (string) - An abstract of the object's purpose.
+ parentId: ` ` (string, optional) - The unique identifier of the objects parent hex encoded to a string. This may be used to traverse up the tree. For objects stored at the root of a user, this value will be null.
+ acm (ACMResponse, required) - The acm value associated with this object in object form
+ permission (PermissionResponse, optional) - [1.1] The permissions associated with this object by capability and resource allowed.
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value will be null.
+ contentSize: 1511 (string) - The length of the object's content stream, if present. For objects without a content stream, this value will be 0.
+ isPDFAvailable: `false` (boolean) - Indicates if a PDF rendition is available for this object.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.
+ properties (array[Property]) - Array of custom properties associated with the object.
+ callerPermissions (CallerPermission) - Permissions granted to the caller that resulted in this object being returned.
+ permissions (array[PermissionUser,PermissionGroup]) - **Deprecated** - Array of permissions associated with this object.

## ObjectRespChild1 (object)

+ id: `11e5e4867a6e3d8389020242ac110001`  (string, required) - The unique identifier of the object hex encoded to a string. This value can be used for alterations and listing on other RESTful methods.
+ createdDate: `2016-03-07T17:03:13Z`  (string) - The date and time the object was created in the system in UTC ISO 8601 format.
+ createdBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that created the object.
+ modifiedDate: `2016-03-07T17:03:13Z` (string) -  The date and time the object was last modified in the system in UTC ISO 8601 format. For unchanged objects, this will reflect the same value as the createdDate field.
+ modifiedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that last modified this object. For unchanged objects, this will reflect the same value as the createdBy field.
+ deletedDate: `0001-01-01T00:00:00Z` (string, optional) -  The date and time the object was deleted in the system in UTC ISO 8601 format. This field is only populated if the object is in the trash.
+ deletedBy: `` (string) - The user that deleted the object. This field is only populated if the object is in the trash.
+ changeCount: 42 (number) - The total count of changes that have been made to this object over its lifespan. Synonymous with version number. For unchanged objects, this will always be 0.
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.
+ ownedBy: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us` (string) - The resource that owns this object. For a user, this is denoted with a prefix of `user/` followed by the user's distinguished name.  A group is prefixed by `group/` followed by project name, project display name, and group name.  Example group resource string: `group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1` 
+ typeId: `11e5e48664f5d8c789020242ac110002` (string) - The unique identifier of the type assigned this object hex encoded to a string.
+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `child 1` (string) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
+ description: `Description of object here` (string) - An abstract of the object's purpose.
+ parentId: `11e5e4867a6e3d8389020242ac11aaaa` (string, optional) - The unique identifier of the objects parent hex encoded to a string. This may be used to traverse up the tree. For objects stored at the root of a user, this value will be null.
+ acm (ACMResponse, required) - The acm value associated with this object in object form
+ permission (PermissionResponse, optional) - [1.1] The permissions associated with this object by capability and resource allowed.
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value will be null.
+ contentSize: 1511 (string) - The length of the object's content stream, if present. For objects without a content stream, this value will be 0.
+ isPDFAvailable: `false` (boolean) - Indicates if a PDF rendition is available for this object.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.
+ properties (array[Property]) - Array of custom properties associated with the object.
+ callerPermissions (CallerPermission) - Permissions granted to the caller that resulted in this object being returned.
+ permissions (array[PermissionUser,PermissionGroup]) - **Deprecated** - Array of permissions associated with this object.

## ObjectRespChild2 (object)

+ id: `11e5e4867a6e3d8389020242ac110002`  (string, required) - The unique identifier of the object hex encoded to a string. This value can be used for alterations and listing on other RESTful methods.
+ createdDate: `2016-03-07T17:03:13Z`  (string) - The date and time the object was created in the system in UTC ISO 8601 format.
+ createdBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that created the object.
+ modifiedDate: `2016-03-07T17:03:13Z` (string) -  The date and time the object was last modified in the system in UTC ISO 8601 format. For unchanged objects, this will reflect the same value as the createdDate field.
+ modifiedBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) - The user that last modified this object. For unchanged objects, this will reflect the same value as the createdBy field.
+ deletedDate: `0001-01-01T00:00:00Z` (string, optional) -  The date and time the object was deleted in the system in UTC ISO 8601 format. This field is only populated if the object is in the trash.
+ deletedBy: `` (string) - The user that deleted the object. This field is only populated if the object is in the trash.
+ changeCount: 42 (number) - The total count of changes that have been made to this object over its lifespan. Synonymous with version number. For unchanged objects, this will always be 0.
+ changeToken: `65eea405306ed436d18b8b1c0b0b2cd3` (string) - A hash of the object's unique identifier and last modification date and time.
+ ownedBy: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us` (string) - The resource that owns this object. For a user, this is denoted with a prefix of `user/` followed by the user's distinguished name.  A group is prefixed by `group/` followed by project name, project display name, and group name.  Example group resource string: `group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1` 
+ typeId: `11e5e48664f5d8c789020242ac110002` (string) - The unique identifier of the type assigned this object hex encoded to a string.
+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `child 2` (string) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
+ description: `Description here` (string) - An abstract of the object's purpose.
+ parentId: `11e5e4867a6e3d8389020242ac11aaaa` (string, optional) - The unique identifier of the objects parent hex encoded to a string. This may be used to traverse up the tree. For objects stored at the root of a user, this value will be null.
+ acm (ACMResponse, required) - The acm value associated with this object in object form
+ permission (PermissionResponse, optional) - [1.1] The permissions associated with this object by capability and resource allowed.
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value will be null.
+ contentSize: 1511 (string) - The length of the object's content stream, if present. For objects without a content stream, this value will be 0.
+ isPDFAvailable: `false` (boolean) - Indicates if a PDF rendition is available for this object.
+ containsUSPersonsData: `No` (string, optional) - Indicates if this object contains US Persons data.  Allowed values are `Yes`, `No`, and `Unknown`.
+ exemptFromFOIA: `No` (string, optional) - Indicates if this object is exempt from Freedom of Information Act requests.  Allowed values are `Yes`, `No`, and `Unknown`.
+ properties (array[Property]) - Array of custom properties associated with the object.
+ callerPermissions (CallerPermission) - Permissions granted to the caller that resulted in this object being returned.
+ permissions (array[PermissionUser,PermissionGroup]) - **Deprecated** - Array of permissions associated with this object.


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
+ ownedBy: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us` (string) - The resource that owns this object. For a user, this is denoted with a prefix of `user/` followed by the user's distinguished name.  A group is prefixed by `group/` followed by project name, project display name, and group name.  Example group resource string: `group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1` 
+ typeId: `11e5e48664f5d8c789020242ac110002` (string) - The unique identifier of the type assigned this object hex encoded to a string.
+ typeName: `File` (string) - The display name of the type assigned this object.
+ name: `gettysburgaddress.txt` (string) - The name given this object. It need not be unique as it is not used as the identifier of the object internally.
+ description: `Description here` (string) - An abstract of the object's purpose.
+ parentId: ` ` (string, optional) - The unique identifier of the objects parent hex encoded to a string. This may be used to traverse up the tree. For objects stored at the root of a user, this value will be null.
+ acm (ACMResponse, required) - The acm value associated with this object in object form
+ permission (PermissionResponse, optional) - [1.1] The permissions associated with this object by capability and resource allowed.
+ contentType: `text` (string) - The mime-type, and potentially character set encoding for the object's content stream, if present. For objects without a content stream, this value will be null.
+ contentSize: 1511 (string) - The length of the object's content stream, if present. For objects without a content stream, this value will be 0.
+ isPDFAvailable: `false` (boolean) - Indicates if a PDF rendition is available for this object.
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

+ allow: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10`, `group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1` (array[string], optional) - The list of resources allowed to perform this capability
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

+ allow: `user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10`, `group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1` (array[string], optional) - The list of resources allowed to perform this capability
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
+ createdDate: `2016-03-07T17:03:13Z` (string) - The date and time the property was created in the system in UTC ISO 8601 format.
+ createdBy: `CN=test tester10,OU=People,OU=DAE,OU=chimera,O=U.S. Government,C=US` (string) -  The user that created the property .
+ modifiedDate: `2016-03-07T17:03:13Z` (string) - The date and time the property was last modified in the system in UTC ISO 8601 format.
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


