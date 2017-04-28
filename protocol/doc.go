/*
Package protocol provides structures which represent operations and returns from ObjectDrive.

Basics

The Object structure represents the core of an object in the Drive, and is typically the structure
returned from many operations on ObjectDrive.  This is a nestable structure, with many attributes
containing other objects from the protocol package.

Objects to initiate changes in the Drive are typically suffixed with *Request.  POSTing correctly
formatted objects to the correct route in a running ObjectDrive instance will cause the built
action to be performed; e.g. DeleteObjectRequest or CreateObjectRequest.

*/
package protocol
