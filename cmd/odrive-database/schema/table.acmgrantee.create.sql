delimiter //
SELECT 'Creating acmgrantee table' as Action
//
CREATE TABLE IF NOT EXISTS acmgrantee
(
  grantee varchar(255) not null
  ,projectName varchar(255) null
  ,projectDisplayName varchar(255) null
  ,groupName varchar(255) null
  ,userDistinguishedName varchar(255) null
  ,displayName varchar(255) null
  ,CONSTRAINT pk_acmgrantee PRIMARY KEY (grantee)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
//
delimiter ;
