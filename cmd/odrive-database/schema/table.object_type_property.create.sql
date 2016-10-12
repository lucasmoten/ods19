CREATE TABLE IF NOT EXISTS object_type_property
(
  id binary(16) not null default 0
  ,createdDate timestamp(6) null
  ,createdBy varchar(255) not null
  ,modifiedDate timestamp(6) null
  ,modifiedBy varchar(255) null
  ,isDeleted boolean null
  ,deletedDate timestamp(6) null
  ,deletedBy varchar(255) null
  ,typeId binary(16) not null
  ,propertyId binary(16) not null
  ,CONSTRAINT pk_object_type_property PRIMARY KEY (id)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_typeId (typeId)
  ,INDEX ix_propertyId (propertyId)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;
