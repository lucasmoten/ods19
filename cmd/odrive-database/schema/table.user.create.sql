CREATE TABLE IF NOT EXISTS user
(
  id binary(16) not null default 0
  ,createdDate timestamp(6) null
  ,createdBy varchar(255) not null
  ,modifiedDate timestamp(6) null
  ,modifiedBy varchar(255) null
  ,changeCount int null
  ,changeToken varchar(60) null
  ,distinguishedName varchar(255) null
  ,displayName varchar(100) null
  ,email varchar(255) null
  ,CONSTRAINT pk_user PRIMARY KEY (id)
  ,UNIQUE (distinguishedName)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;
# Archive table takes the same format, but does not specify defaults
CREATE TABLE IF NOT EXISTS a_user
(
  a_id int not null auto_increment
  ,id binary(16) not null
  ,createdDate timestamp(6) null
  ,createdBy varchar(255) not null
  ,modifiedDate timestamp(6) null
  ,modifiedBy varchar(255) null
  ,changeCount int null
  ,changeToken varchar(60) null
  ,distinguishedName varchar(255) null
  ,displayName varchar(100) null
  ,email varchar(255) null
  ,CONSTRAINT pk_a_user PRIMARY KEY (a_id)
  ,INDEX ix_distinguishedName (distinguishedName)
  ,INDEX ix_modifiedDate (modifiedDate)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;
