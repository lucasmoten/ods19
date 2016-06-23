delimiter //
SELECT 'Creating objectacm table' as Action
//
CREATE TABLE IF NOT EXISTS objectacm
(
  id binary(16) not null default 0
  ,createdDate timestamp(6) null
  ,createdBy varchar(255) not null
  ,modifiedDate timestamp(6) null
  ,modifiedBy varchar(255) null
  ,isDeleted boolean null
  ,deletedDate timestamp(6) null
  ,deletedBy varchar(255) null
  ,objectId binary(16) not null
  ,acmId binary(16) not null
  ,CONSTRAINT pk_objectacm PRIMARY KEY (id)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_objectId (objectId)
  ,INDEX ix_acmId (acmId)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
//
SELECT 'Creating a_objectacm table' as Action
//
CREATE TABLE IF NOT EXISTS a_objectacm
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
  ,objectId binary(16) not null
  ,acmId binary(16) not null
  ,CONSTRAINT pk_a_objectacm PRIMARY KEY (a_id)
  ,INDEX ix_id (id)
  ,INDEX ix_objectId (objectId)
  ,INDEX ix_acmId (acmId)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
//
delimiter ;
