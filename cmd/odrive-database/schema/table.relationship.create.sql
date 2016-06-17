delimiter //
SELECT 'Creating relationship table' as Action
//
CREATE TABLE IF NOT EXISTS relationship
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
  ,sourceId binary(16) not null
  ,targetId binary(16) not null
  ,description varchar(10240) null
  ,classificationPM varchar(200) null
  ,CONSTRAINT pk_relationship PRIMARY KEY (id)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_sourceId (sourceId)
  ,INDEX ix_targetId (targetId)
)
//
SELECT 'Creating a_relationship table' as Action
//
# Archive table takes the same format, but does not specify defaults
CREATE TABLE IF NOT EXISTS a_relationship
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
  ,sourceId binary(16) not null
  ,targetId binary(16) not null
  ,description varchar(10240) null
  ,classificationPM varchar(200) null
  ,CONSTRAINT pk_a_relationship PRIMARY KEY (a_id)
  ,INDEX ix_id (id)
  ,INDEX ix_modifiedDate (modifiedDate)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_changeCount (changeCount)
)
//
delimiter ;
