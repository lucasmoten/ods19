delimiter //
SELECT 'Creating acmvalue table' as Action
//
CREATE TABLE IF NOT EXISTS acmvalue
(
  id binary(16) not null default 0
  ,createdDate timestamp(6) null
  ,createdBy varchar(255) not null
  ,modifiedDate timestamp(6) null
  ,modifiedBy varchar(255) null
  ,isDeleted boolean null
  ,deletedDate timestamp(6) null
  ,deletedBy varchar(255) null
  ,name varchar(255) null
  ,CONSTRAINT pk_acmvalue PRIMARY KEY (id)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_name (name)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
//
SELECT 'Creating a_acmvalue table' as Action
//
CREATE TABLE IF NOT EXISTS a_acmvalue
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
  ,name varchar(255) null
  ,CONSTRAINT pk_a_acmvalue PRIMARY KEY (a_id)
  ,INDEX ix_id (id)
  ,INDEX ix_name (name)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
//
delimiter ;
