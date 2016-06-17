delimiter //
SELECT 'Creating object_type table' as Action
//
CREATE TABLE IF NOT EXISTS object_type
(
  id binary(16) not null default 0
  ,createdDate timestamp(6) null
  ,createdBy varchar(255) not null
  ,modifiedDate timestamp(6) null
  ,modifiedBy varchar(255) null
  ,isDeleted boolean null
  ,deletedDate timestamp(6) null
  ,deletedBy varchar(255) null
  ,ownedBy varchar(255) null
  ,changeCount int null
  ,changeToken varchar(60) null
  ,name varchar(255) not null
  ,description varchar(10240) null
  ,contentConnector varchar(2000) null
  ,CONSTRAINT pk_object_type PRIMARY KEY (id)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_name (name)
)
//
SELECT 'Creating a_object_type table' as Action
//
# Archive table takes the same format, but does not specify defaults
CREATE TABLE IF NOT EXISTS a_object_type
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
  ,ownedBy varchar(255) null
  ,changeCount int null
  ,changeToken varchar(60) null
  ,name varchar(255) not null
  ,description varchar(10240) null
  ,contentConnector varchar(2000) null
  ,CONSTRAINT pk_a_object_type PRIMARY KEY (a_id)
  ,INDEX ix_id (id)
  ,INDEX ix_modifiedDate (modifiedDate)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_changeCount (changeCount)
)
//
delimiter ;
