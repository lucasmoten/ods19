delimiter //
SELECT 'Creating user_object_subscription table' as Action
//
CREATE TABLE IF NOT EXISTS user_object_subscription
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
  ,onCreate boolean null
  ,onUpdate boolean null
  ,onDelete boolean null
  ,recursive boolean null
  ,CONSTRAINT pk_user_object_subscription PRIMARY KEY (id)
  ,INDEX ix_createdBy (createdBy)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_objectId (objectId)
)
//
SELECT 'Creating a_user_object_subscription table' as Action
//
# Archive table takes the same format, but does not specify defaults
CREATE TABLE IF NOT EXISTS a_user_object_subscription
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
  ,onCreate boolean null
  ,onUpdate boolean null
  ,onDelete boolean null
  ,recursive boolean null
  ,CONSTRAINT pk_a_user_object_subscription PRIMARY KEY (a_id)
  ,INDEX ix_id (id)
  ,INDEX ix_createdBy (createdBy)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_objectId (objectId)
)
//
delimiter ;
