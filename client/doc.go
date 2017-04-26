/*
Package client implements common operations to build basic client-side
applications.  These focus on the basic create, read, update, and destroy type operations.

Basics

This is kinda an overview of things.

Operations

Once I figure out what it does, i'll put the available operations and examples
here.  Kinda like the following.

  me, err := NewClient(conf)

  newObj, err := me.CreateObject(upObj, nil)
  reader, err := me.GetObjectStream(newObj.ID)
  WriteObject(newObj.Name, reader)

*/
package client
