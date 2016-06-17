delimiter //
SELECT 'Creating object table' as Action
//
CREATE TABLE IF NOT EXISTS object
(
  id binary(16) not null default 0
  ,createdDate timestamp(6) null
  ,createdBy varchar(255) not null
  ,modifiedDate timestamp(6) null
  ,modifiedBy varchar(255) null
  ,isDeleted boolean null
  ,deletedDate timestamp(6) null
  ,deletedBy varchar(255) null
  ,isAncestorDeleted boolean null
  ,isExpunged boolean null
  ,expungedDate timestamp(6) null
  ,expungedBy varchar(255) null
  ,changeCount int null
  ,changeToken varchar(60) null
  ,ownedBy varchar(255) null
  ,typeId binary(16) null
  ,name varchar(255) not null
  ,description varchar(10240) null
  ,parentId binary(16) null
  ,contentConnector varchar(2000) null
  ,rawAcm text null
  ,contentType varchar(255) null
  ,contentSize bigint null
  ,contentHash binary(32) null
  ,encryptIV binary(16) null
  ,ownedByNew varchar(255) null
  ,isPDFAvailable boolean null
  ,isStreamStored boolean null
  ,isUSPersonsData boolean null
  ,isFOIAExempt boolean null
  ,CONSTRAINT pk_object PRIMARY KEY (id)
  ,INDEX ix_isDeleted (isDeleted)
  ,INDEX ix_name (name)
  ,INDEX ix_ownedBy (ownedBy)
  ,INDEX ix_parentId (parentId)
  ,INDEX ix_typeId (typeId)
)
//
SELECT 'Creating a_object table' as Action
//
# Archive table takes the same format, but does not specify defaults
CREATE TABLE IF NOT EXISTS a_object
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
  ,isAncestorDeleted boolean null
  ,isExpunged boolean null
  ,expungedDate timestamp(6) null
  ,expungedBy varchar(255) null
  ,changeCount int null
  ,changeToken varchar(60) null
  ,ownedBy varchar(255) null
  ,typeId binary(16) null
  ,name varchar(255) not null
  ,description varchar(10240) null
  ,parentId binary(16) null
  ,contentConnector varchar(2000) null
  ,rawAcm text null
  ,contentType varchar(255) null
  ,contentSize bigint null
  ,contentHash binary(32) null
  ,encryptIV binary(16) null
  ,ownedByNew varchar(255) null
  ,isPDFAvailable boolean null
  ,isStreamStored boolean null
  ,isUSPersonsData boolean null
  ,isFOIAExempt boolean null
  ,CONSTRAINT pk_object PRIMARY KEY (a_id)
  ,INDEX ix_id (id)
  ,INDEX ix_modifiedDate (modifiedDate)
  ,INDEX ix_ownedBy (ownedBy)
  ,INDEX ix_changeCount (changeCount)
)
//
delimiter ;
