CREATE TABLE IF NOT EXISTS user_object_favorite
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
  ,CONSTRAINT pk_user_object_favorite PRIMARY KEY (id)
  ,INDEX ix_createdBy (createdBy)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_objectId (objectId)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;

# Archive table takes the same format, but does not specify defaults
CREATE TABLE IF NOT EXISTS a_user_object_favorite
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
  ,CONSTRAINT pk_a_user_object_favorite PRIMARY KEY (a_id)
  ,INDEX ix_id (id)
  ,INDEX ix_createdBy (createdBy)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_objectId (objectId)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;
