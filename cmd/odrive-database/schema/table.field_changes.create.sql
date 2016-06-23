delimiter //
SELECT 'Creating field_changes table' as Action
//
CREATE TABLE IF NOT EXISTS field_changes
(
  id int not null auto_increment
  ,modifiedDate timestamp(6) null
  ,modifiedBy varchar(255) null
  ,tableName varchar(255) not null
  ,recordId binary(16) not null
  ,columnName varchar(255) not null
  ,newValue varchar(10240) null
  ,newTextValue text null
  ,CONSTRAINT pk_field_changes PRIMARY KEY (id)
  ,INDEX ix_tableName (tableName)
  ,INDEX ix_recordId (recordId)
  ,INDEX ix_columnName (columnName)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
//
delimiter ;
