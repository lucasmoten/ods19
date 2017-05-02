/*
Package client implements common operations to build client-side applications.
These focus on the basic create, read, update, and destroy type operations.

Below briefly illustrates a simple cycle of creating a client and using it to perform
a few operations.  The first step is to create a new client.

  var conf = Config{
    // Setup certs, odrive URL, etc
  }

  client, err := NewClient(conf)
  // err handling

This client can then be used to perform operations in ObjectDrive.

 var uploadObject = protocol.CreateObjectRequest{
   TypeName: "File",
   Name:     "SampleFile",
   // etc..
 }

 reader, err := os.Open("SampleFile")

 retObj, err := client.CreateObject(uploadObject, reader)

The return from CreateObject will have the metadata of the uploaded file, which can be
used to facilitate further operations, such as retrieval or deletion. E.g.

  // Move the object to the trash.
  deleteResp, err := client.DeleteObject(retObj.ID, retObj.ChangeToken)

*/
package client
