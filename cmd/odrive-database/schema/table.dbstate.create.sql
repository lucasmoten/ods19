delimiter //
SELECT 'Creating dbstate table' as Action
//
CREATE TABLE IF NOT EXISTS dbstate
(
  createdDate timestamp(6) null,
  modifiedDate timestamp(6) null,
  schemaversion varchar(255) null,
  identifier varchar(200) null
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
//
delimiter ;
