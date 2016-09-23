DROP TRIGGER IF EXISTS ti_object_permission;
DROP TRIGGER IF EXISTS tu_object_permission;
DROP TRIGGER IF EXISTS td_object_permission;
DROP TRIGGER IF EXISTS td_a_object_permission;
SOURCE triggers.object_permission.create.sql

delimiter //
DROP PROCEDURE IF EXISTS sp_PatchCRUDSForOwner
//
SELECT 'Creating procedure to patch objects so they have permissions granting CRUDS to owner' as Action
//
CREATE PROCEDURE sp_PatchCRUDSForOwner(
    IN masterKey varchar(255),
    IN randomString varchar(255)
)
BEGIN
    DECLARE vID binary(16);
    DECLARE vOwnedBy varchar(255);
    DECLARE vGrantee varchar(255);
    DECLARE vAcmShare varchar(255);
    DECLARE vAllowCreate tinyint(1);
    DECLARE vAllowRead tinyint(1);
    DECLARE vAllowUpdate tinyint(1);
    DECLARE vAllowDelete tinyint(1);
    DECLARE vAllowShare tinyint(1);
    DECLARE vEncryptKey binary(32);
    DECLARE vPermissionIV binary(32);
    DECLARE aGrantee varchar(255);
    DECLARE aAcmShare varchar(255);
    DECLARE aAllowCreate tinyint(1);
    DECLARE aAllowRead tinyint(1);
    DECLARE aAllowUpdate tinyint(1);
    DECLARE aAllowDelete tinyint(1);
    DECLARE aAllowShare tinyint(1);
    DECLARE eEncryptKey binary(32);
    DECLARE ePermissionIV binary(32);
    DECLARE ePermissionMAC binary(32);

    BLOCK1: begin
        DECLARE finished_objects integer default 0;
        DECLARE cursor_objects cursor for SELECT id, ownedby from object order by createddate asc;
        DECLARE continue handler for not found set finished_objects = 1;
        OPEN cursor_objects;
        get_object: LOOP
            FETCH cursor_objects INTO vID, vOwnedBy;
            IF finished_objects = 1 THEN
                CLOSE cursor_objects;
                LEAVE get_object;
            END IF;

            SET aGrantee := aacflatten(vOwnedBy);
            SET aAcmShare := CONCAT('{"users":["',vOwnedBy,'"]}');

            BLOCK2: begin
                DECLARE finished_permissions integer default 0;
                DECLARE cursor_permissions cursor for SELECT grantee, acmShare, allowCreate, allowRead, allowUpdate, allowDelete, allowShare, encryptKey, permissionIV FROM object_permission WHERE objectId = vID AND isDeleted = 0;
                DECLARE continue handler for not found set finished_permissions = 1;
                SET aAllowCreate := 1;
                SET aAllowRead := 1;
                SET aAllowUpdate := 1;
                SET aAllowDelete := 1;
                SET aAllowShare := 1;
                OPEN cursor_permissions;
                get_permissions: LOOP
                    FETCH cursor_permissions INTO vGrantee, vAcmShare, vAllowCreate, vAllowRead, vAllowUpdate, vAllowDelete, vAllowShare, vEncryptKey, vPermissionIV;
                    IF finished_permissions = 1 THEN
                        CLOSE cursor_permissions;
                        LEAVE get_permissions;
                    END IF;
                    IF vGrantee = aGrantee THEN
                        SET aAcmShare := vAcmShare;
                        IF vAllowCreate = 1 THEN
                            SET aAllowCreate := 0;
                        END IF;
                        IF vAllowRead = 1 THEN
                            SET aAllowRead := 0;
                        END IF;
                        IF vAllowUpdate = 1 THEN
                            SET aAllowUpdate := 0;
                        END IF;
                        IF vAllowDelete = 1 THEN
                            SET aAllowDelete := 0;
                        END IF;
                        IF vAllowShare = 1 THEN
                            SET aAllowShare := 0;
                        END IF;
                    END IF;
                END LOOP get_permissions;

                IF (aAllowCreate + aAllowRead + aAllowUpdate + aAllowDelete + aAllowShare) > 0 THEN
                    -- make missing permission for the owner
                    SET ePermissionIV := unhex(pseudorand256(concat(randomString,hex(vID))));
                    SET eEncryptKey := unhex(
                        bitwise256_xor(
                            new_keydecrypt(masterKey,hex(ePermissionIV)),
                            bitwise256_xor(
                                new_keydecrypt(masterKey,hex(vPermissionIV)),
                                hex(vEncryptKey)
                            )
                        )
                    );
                    SET ePermissionMAC := unhex(new_keymac(masterKey, aGrantee, aAllowCreate, aAllowRead, aAllowUpdate, aAllowDelete, aAllowShare, hex(eEncryptKey)));
                    SELECT CONCAT('Adding permission for ', aGrantee, ' to objectid = ', hex(vID), ' iv=', hex(ePermissionIV), ' key=', hex(eEncryptKey), ' mac=', hex(ePermissionMAC));
                    INSERT INTO object_permission SET createdBy = vOwnedBy, objectId = vID, grantee = aGrantee, acmShare = aAcmShare, 
                        allowCreate = aAllowCreate, allowRead = aAllowRead, allowUpdate = aAllowUpdate, allowDelete = aAllowDelete, allowShare = aAllowShare,
                        explicitShare = 0, encryptKey = eEncryptKey, permissionIV = ePermissionIV, permissionMAC = ePermissionMAC;
                END IF;
            END BLOCK2;
  
        END LOOP get_object;        
    END BLOCK1;
END
//
delimiter ;

