delimiter //

SELECT 'Temporarily dropping triggers for data migration' as Action
//
source triggers.drop.sql

DROP PROCEDURE IF EXISTS sp_PatchRevisionStreamInfo
//
SELECT 'Creating procedure to patch revision stream info' as Action
//
CREATE PROCEDURE sp_PatchRevisionStreamInfo(
)
BEGIN
    DECLARE vID binary(16);
    DECLARE vChangeCount int;
    DECLARE vContentConnector varchar(2000) default '';
    DECLARE vContentType varchar(255) default '';
    DECLARE vContentSize bigint default null;
    DECLARE vContentHash binary(32) default null;
    DECLARE vEncryptIV binary(16) default null;
    DECLARE aContentConnector varchar(2000);
    DECLARE aContentType varchar(255);
    DECLARE aContentSize bigint;
    DECLARE aContentHash binary(32);
    DECLARE aEncryptIV binary(16);

    BLOCK1: begin
        DECLARE finished_objects integer default 0;
        DECLARE cursor_objects cursor for SELECT id FROM object;
        DECLARE continue handler for not found set finished_objects = 1;
        OPEN cursor_objects;
        get_object: LOOP
            FETCH cursor_objects INTO vID;
            IF finished_objects = 1 THEN
                CLOSE cursor_objects;
                LEAVE get_object;
            END IF;
            BLOCK2: begin
                DECLARE finished_archive integer default 0;
                DECLARE cursor_archive cursor for SELECT changeCount, contentConnector, contentType, contentSize, contentHash, encryptIV FROM a_object WHERE id = vID ORDER BY changeCount ASC; 
                DECLARE continue handler for not found set finished_archive = 1;
                SET vContentConnector := '';
                SET vContentType := '';
                SET vContentSize := null;
                SET vContentHash := null;
                SET vEncryptIV := null;
                OPEN cursor_archive;
                get_archive: LOOP
                    FETCH cursor_archive INTO vChangeCount, aContentConnector, aContentType, aContentSize, aContentHash, aEncryptIV;
                    IF finished_archive = 1 THEN
                        CLOSE cursor_archive;
                        LEAVE get_archive;
                    END IF;
                    IF length(aContentConnector) = 0 THEN
                        IF length(vContentConnector) > 0 THEN
                            UPDATE a_object SET contentConnector = vContentConnector WHERE id = vID and changeCount = vChangeCount;
                        END IF;
                    ELSE
                        SET vContentConnector := aContentConnector;
                    END IF;
                    IF length(aContentType) = 0 THEN
                        IF length(vContentType) > 0 THEN
                            UPDATE a_object SET contentType = vContentType WHERE id = vID and changeCount = vChangeCount;
                        END IF;
                    ELSE
                        SET vContentType := aContentType;
                    END IF;
                    IF aContentSize IS NULL THEN
                        IF vContentSize IS NOT NULL THEN
                            UPDATE a_object SET contentSize = vContentSize WHERE id = vID and changeCount = vChangeCount;
                        END IF;
                    ELSE
                        SET vContentSize := aContentSize;
                    END IF;
                    IF hex(aContentHash) IS NULL THEN
                        IF hex(vContentHash) IS NOT NULL THEN
                            UPDATE a_object SET contentHash = vContentHash WHERE id = vID and changeCount = vChangeCount;
                        END IF;
                    ELSE
                        IF length(aContentHash) = 0 THEN
                            IF hex(vContentHash) IS NOT NULL THEN
                                UPDATE a_object SET contentHash = vContentHash WHERE id = vID and changeCount = vChangeCount;
                            END IF;
                        ELSE
                            SET vContentHash := aContentHash;
                        END IF;
                    END IF;
                    IF hex(aEncryptIV) IS NULL THEN
                        IF hex(vEncryptIV) IS NOT NULL THEN
                            UPDATE a_object SET encryptIV = vEncryptIV WHERE id = vID and changeCount = vChangeCount;
                        END IF;
                    ELSE
                        IF length(aEncryptIV) = 0 THEN
                            IF hex(vEncryptIV) IS NOT NULL THEN
                                UPDATE a_object SET encryptIV = vEncryptIV WHERE id = vID and changeCount = vChangeCount;
                            END IF;
                        ELSE
                            SET vEncryptIV := aEncryptIV;
                        END IF;
                    END IF;
                END LOOP get_archive;                
            END BLOCK2;
            UPDATE object SET contentConnector = vContentConnector, contentType = vContentType, contentSize = vContentSize, contentHash = vContentHash, encryptIV = vEncryptIV WHERE id = vID AND changeCount = vChangeCount;  
        END LOOP get_object;        
    END BLOCK1;
END
//
delimiter ;

SELECT 'Executing procedure to patch revision stream info' as Action;
call sp_PatchRevisionStreamInfo();

SELECT 'Removing procedure for patching revision stream info' as Action;
drop procedure if exists sp_PatchRevisionStreamInfo;

SELECT 'Recreating triggers' as Action;
source triggers.create.sql
