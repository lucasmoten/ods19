delimiter //
SELECT 'Creating object_permission table' as Action
//
CREATE TABLE IF NOT EXISTS object_permission
(
  id binary(16) not null default 0
  ,createdDate timestamp(6) null
  ,createdBy varchar(255) not null
  ,modifiedDate timestamp(6) null
  ,modifiedBy varchar(255) null
  ,isDeleted boolean null
  ,deletedDate timestamp(6) null
  ,deletedBy varchar(255) null
  ,changeCount int null
  ,changeToken varchar(60) null
  ,objectId binary(16) null
  ,grantee varchar(255) not null
  ,acmShare text not null
  ,allowCreate boolean not null
  ,allowRead boolean not null
  ,allowUpdate boolean not null
  ,allowDelete boolean not null
  ,allowShare boolean not null
  ,explicitShare boolean not null
  ,encryptKey binary(32) null
  ,permissionIV binary(32) null
  ,permissionMAC binary(32) null
  ,CONSTRAINT pk_object_permission PRIMARY KEY (id)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_objectId (objectId)
  ,INDEX ix_grantee (grantee)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
//
SELECT 'Creating a_object_permission table' as Action
//
# Archive table takes the same format, but does not specify defaults
CREATE TABLE IF NOT EXISTS a_object_permission
(
  a_id int not null auto_increment
  ,id binary(16) not null
  ,createdDate timestamp(6) null
  ,createdBy varchar(255) not null
  ,modifiedDate timestamp(6) null
  ,modifiedBy varchar(255) null
  ,isDeleted boolean null
  ,deletedDate timestamp(6) null
  ,deletedBy varchar(255) null
  ,changeCount int null
  ,changeToken varchar(60) null
  ,objectId binary(16) null
  ,grantee varchar(255) not null
  ,acmShare text not null
  ,allowCreate boolean not null
  ,allowRead boolean not null
  ,allowUpdate boolean not null
  ,allowDelete boolean not null
  ,allowShare boolean not null
  ,explicitShare boolean not null
  ,encryptKey binary(32) null
  ,permissionIV binary(32) null
  ,permissionMAC binary(32) null
  ,CONSTRAINT pk_a_object_permission PRIMARY KEY (a_id)
  ,INDEX ix_id (id)
  ,INDEX ix_modifiedDate (modifiedDate)
  ,INDEX ix_changeCount (changeCount)
  ,INDEX ix_objectId (objectId)
  ,INDEX ix_grantee (grantee)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
//
delimiter ;
