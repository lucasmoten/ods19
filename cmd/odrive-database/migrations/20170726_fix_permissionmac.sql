-- +migrate Up

-- Fixes permission macs if broken
INSERT INTO migration_status SET description = '20170726_checking_permissions';
DROP PROCEDURE IF EXISTS sp_Patch_20170726_checking_permissions;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170726_checking_permissions()
BEGIN
    DECLARE vActual varchar(255) default '';
    DECLARE vExpected varchar(255) default '';
    select lower(hex(permissionmac)), new_keymac('${OD_ENCRYPT_MASTERKEY}',p.grantee,p.allowcreate,p.allowread,p.allowupdate,p.allowdelete,p.allowshare,hex(p.encryptkey)) INTO vActual, vExpected from object_permission p limit 1;
    IF vActual <> vExpected THEN
        update object_permission p set permissionmac = unhex(new_keymac('${OD_ENCRYPT_MASTERKEY}',p.grantee,p.allowcreate,p.allowread,p.allowupdate,p.allowdelete,p.allowshare,hex(p.encryptkey)));
        update a_object_permission p set permissionmac = unhex(new_keymac('${OD_ENCRYPT_MASTERKEY}',p.grantee,p.allowcreate,p.allowread,p.allowupdate,p.allowdelete,p.allowshare,hex(p.encryptkey)));
    END IF;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170726_checking_permissions();
DROP PROCEDURE IF EXISTS sp_Patch_20170726_checking_permissions; 

update dbstate set schemaVersion = '20170726' where schemaVersion <> '20170726';

-- +migrate Down

-- A downgrade from this schema version will not be supported.

update dbstate set schemaVersion = '20170726' where schemaVersion <> '20170726';