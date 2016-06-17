delimiter //
SELECT 'Creating acmkey table' as Action
//
CREATE TABLE IF NOT EXISTS acmkey
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
  ,CONSTRAINT pk_acmkey PRIMARY KEY (id)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_name (name)
)
//
SELECT 'Creating a_acmkey table' as Action
//
CREATE TABLE IF NOT EXISTS a_acmkey
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
  ,CONSTRAINT pk_a_acmkey PRIMARY KEY (a_id)
  ,INDEX ix_id (id)
  ,INDEX ix_name (name)
)
//
delimiter ;
