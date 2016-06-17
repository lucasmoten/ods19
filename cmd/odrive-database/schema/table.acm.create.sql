delimiter //
SELECT 'Creating acm table' as Action
//
CREATE TABLE IF NOT EXISTS acm
(
  id binary(16) not null default 0
  ,createdDate timestamp(6) null
  ,createdBy varchar(255) not null
  ,modifiedDate timestamp(6) null
  ,modifiedBy varchar(255) null
  ,isDeleted boolean null
  ,deletedDate timestamp(6) null
  ,deletedBy varchar(255) null
  ,name text null
  ,CONSTRAINT pk_acm PRIMARY KEY (id)
  ,INDEX ix_isDeleted (isDeleted)
)
//
SELECT 'Creating a_acm table' as Action
//
CREATE TABLE IF NOT EXISTS a_acm
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
  ,name text null
  ,CONSTRAINT pk_a_acm PRIMARY KEY (a_id)
  ,INDEX ix_id (id)
)
//
delimiter ;
