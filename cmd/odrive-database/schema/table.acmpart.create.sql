CREATE TABLE IF NOT EXISTS acmpart
(
  id binary(16) not null default 0
  ,createdDate timestamp(6) null
  ,createdBy varchar(255) not null
  ,modifiedDate timestamp(6) null
  ,modifiedBy varchar(255) null
  ,isDeleted boolean null
  ,deletedDate timestamp(6) null
  ,deletedBy varchar(255) null
  ,acmId binary(16) not null
  ,acmKeyId binary(16) not null
  ,acmValueId binary(16) not null
  ,CONSTRAINT pk_acmpart PRIMARY KEY (id)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_acmId (acmId)
  ,INDEX ix_acmKeyId (acmKeyId)
  ,INDEX ix_acmValueId (acmValueId)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;

CREATE TABLE IF NOT EXISTS a_acmpart
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
  ,acmId binary(16) not null
  ,acmKeyId binary(16) not null
  ,acmValueId binary(16) not null
  ,CONSTRAINT pk_a_acmpart PRIMARY KEY (a_id)
  ,INDEX ix_id (id)
  ,INDEX ix_acmId (acmId)
  ,INDEX ix_acmKeyId (acmKeyId)
  ,INDEX ix_acmValueId (acmValueId)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;
