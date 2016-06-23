delimiter //
SELECT 'Creating property table' as Action
//
CREATE TABLE IF NOT EXISTS property
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
  ,name varchar(255) null
  ,propertyValue varchar(20000) null
  ,classificationPM varchar(200) null
  ,CONSTRAINT pk_property PRIMARY KEY (id)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_name (name)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
//
SELECT 'Creating a_property table' as Action
//
# Archive table takes the same format, but does not specify defaults
CREATE TABLE IF NOT EXISTS a_property
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
  ,name varchar(255) null
  ,propertyValue varchar(20000) null
  ,classificationPM varchar(200) null
  ,CONSTRAINT pk_a_property PRIMARY KEY (a_id)
  ,INDEX ix_id (id)
  ,INDEX ix_modifiedDate (modifiedDate)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_changeCount (changeCount)
  ,INDEX ix_name (name)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
//
delimiter ;
