delimiter //
SELECT 'Creating object_tag table' as Action
//
CREATE TABLE IF NOT EXISTS object_tag
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
  ,name varchar(255) not null
  ,CONSTRAINT pk_object_tag PRIMARY KEY (id)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_objectId (objectId)
  ,INDEX ix_name (name)
)
//
SELECT 'Creating a_object_tag table' as Action
//
# Archive table takes the same format, but does not specify defaults
CREATE TABLE IF NOT EXISTS a_object_tag
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
  ,name varchar(255) not null
  ,CONSTRAINT pk_a_object_tag PRIMARY KEY (a_id)
  ,INDEX ix_id (id)
  ,INDEX ix_modifiedDate (modifiedDate)
  ,INDEX ix_changeCount (changeCount)
  ,INDEX ix_objectId (objectId)
)
//
delimiter ;
