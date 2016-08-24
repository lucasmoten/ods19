Changelog
=========

Release vNEXT
-------------

* TODO collect here

Release v0.1.0
--------------

* !192 – Refactor: Remove broken STANDALONE flag
* !197 – FIX: Return 404 instead of 500 when retrieving an object properties and given ID is invalid.
* !200 – NEW: Allow caller to specify returned content-disposition format when requesting streams and zipped content
* !201 – NEW: Response to create object will now populate callerPermisison
* !203 - NEW: Publish Events to Kafka
* !205 – Refactor: Docstrings on Index event fields
* !208 – Enhancement: RPMs generated will now create odrive user when installed
* !209 – Enhancement: All API responses returning object now populate callerPermissions
* !210 – Enhancement: US Persons Data and FOIA Exemption state fields now track Yes/No/Unknown instead of True/False
* !216 - Enhancement: Connection to Zookeeper recovery improvements when timed out
* !218 - Enhancement: Capture full ACM share information for individual permission grants
