-- +migrate Up

-- recreate triggers on object for create/update with support for determining object_pathing
DROP TRIGGER IF EXISTS ti_object;
DROP TRIGGER IF EXISTS tu_object;

-- +migrate StatementBegin
CREATE TRIGGER ti_object
BEFORE INSERT ON object FOR EACH ROW
BEGIN
    DECLARE error_msg varchar(128) default '';
    DECLARE thisTableName varchar(128) default 'object';
    DECLARE count_parent int default 0;
    DECLARE count_type int default 0;
    DECLARE type_contentConnector varchar(2000) default '';

    # Rules
    # Type must be specified
    SELECT COUNT(*) FROM object_type WHERE isDeleted = 0 AND id = NEW.typeId INTO count_type;
    IF count_type = 0 THEN
        SET error_msg := concat(error_msg, 'Field typeId required ');
    END IF;
    # US Persons Data is NULLed if empty (will change to Unknown)
    IF NEW.containsUSPersonsData IS NOT NULL AND NEW.containsUSPersonsData = '' THEN
        SET NEW.containsUSPersonsData := NULL;
    END IF;
    # FOIA Exempt is NULLed if empty (will change to Unknown)
    IF NEW.exemptFromFOIA IS NOT NULL AND NEW.exemptFromFOIA = '' THEN
        SET NEW.exemptFromFOIA := NULL;
    END IF;
    # ParentId must be valid if specified
    IF NEW.parentId IS NOT NULL AND NEW.parentId = '' THEN
        SET NEW.parentId := NULL;
    END IF;
    IF NEW.parentId IS NOT NULL THEN
        SELECT COUNT(*) FROM object WHERE isDeleted = 0 AND id = NEW.parentId INTO count_parent;
        IF count_parent = 0 THEN
            SET error_msg := concat(error_msg, 'Field parentId must be valid ');
        END IF;
    END IF;
    IF error_msg <> '' THEN
        SET error_msg := concat(error_msg, 'when inserting record into ', thisTableName);
        signal sqlstate '45000' set message_text = error_msg;
    END IF;

    # Force values on create
    SET NEW.id := ordered_uuid(UUID());
    SET NEW.createdDate := current_timestamp(6);
    SET NEW.modifiedDate := current_timestamp(6);
    SET NEW.modifiedBy := NEW.createdBy;
    SET NEW.isDeleted := 0;
    SET NEW.deletedDate := NULL;
    SET NEW.deletedBy := NULL;
    SET NEW.isAncestorDeleted := 0;
    SET NEW.isExpunged := 0;
    SET NEW.expungedDate := NULL;
    SET NEW.expungedBy := NULL;
    SET NEW.ownedByNew := NULL;
    # Assign Owner if not set
    IF NEW.ownedBy IS NULL OR NEW.ownedBy = '' THEN
        SET NEW.ownedBy := concat('user/', NEW.createdBy);
    END IF;
    SET NEW.changeCount := 0;
    # Standard change token formula
    SET NEW.changeToken := md5(CONCAT(CAST(NEW.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));
    # Assign contentConnector if not set
    IF NEW.contentConnector IS NULL OR NEW.contentConnector = '' THEN
        SELECT contentConnector FROM object_type WHERE isDeleted = 0 AND id = NEW.typeId INTO type_contentConnector;
        SET NEW.contentConnector := type_contentConnector;
    END IF;
    # Assign PDF availability if not set
    IF NEW.isPDFAvailable IS NULL THEN
        SET NEW.isPDFAvailable := 0;
    END IF;
    # Assign US Persons Data if not set
    IF NEW.containsUSPersonsData IS NULL THEN
        SET NEW.containsUSPersonsData := 'Unknown';
    END IF;
    # Assign FOIA Exempt status if not set
    IF NEW.exemptFromFOIA IS NULL THEN
        SET NEW.exemptFromFOIA := 'Unknown';
    END IF;
    # Assign stream storage state if not set
    IF NEW.isStreamStored IS NULL THEN
        SET NEW.isStreamStored := 0;
    END IF;

    # Archive table
    INSERT INTO
        a_object
    (
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,isAncestorDeleted
        ,isExpunged
        ,expungedDate
        ,expungedBy
        ,changeCount
        ,changeToken
        ,ownedBy
        ,typeId
        ,name
        ,description
        ,parentId
        ,contentConnector
        ,rawAcm
        ,contentType
        ,contentSize
        ,contentHash
        ,encryptIV
        ,ownedByNew
        ,isPDFAvailable
        ,isStreamStored
        ,containsUSPersonsData
        ,exemptFromFOIA
    ) values (
        NEW.id
        ,NEW.createdDate
        ,NEW.createdBy
        ,NEW.modifiedDate
        ,NEW.modifiedBy
        ,NEW.isDeleted
        ,NEW.deletedDate
        ,NEW.deletedBy
        ,NEW.isAncestorDeleted
        ,NEW.isExpunged
        ,NEW.expungedDate
        ,NEW.expungedBy
        ,NEW.changeCount
        ,NEW.changeToken
        ,NEW.ownedBy
        ,NEW.typeId
        ,NEW.name
        ,NEW.description
        ,NEW.parentId
        ,NEW.contentConnector
        ,NEW.rawAcm
        ,NEW.contentType
        ,NEW.contentSize
        ,NEW.contentHash
        ,NEW.encryptIV
        ,NEW.ownedByNew
        ,NEW.isPDFAvailable
        ,NEW.isStreamStored
        ,NEW.containsUSPersonsData
        ,NEW.exemptFromFOIA
    );

    # Specific field level changes
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedBy', newValue = NEW.ownedBy;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'typeId', newValue = hex(NEW.typeId);
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'name', newValue = NEW.name;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'description', newValue = NEW.description;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'parentId', newValue = hex(NEW.parentId);
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentConnector', newValue = NEW.contentConnector;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'rawAcm', newTextValue = NEW.rawAcm;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentType', newValue = NEW.contentType;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentSize', newValue = NEW.contentSize;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentHash', newValue = hex(NEW.contentHash);
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'encryptIV', newValue = hex(NEW.encryptIV);
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedByNew', newValue = NEW.ownedByNew;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isPDFAvailable', newValue = NEW.isPDFAvailable;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isStreamStored', newValue = NEW.isStreamStored;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'containsUSPersonsData', newValue = NEW.containsUSPersonsData;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'exemptFromFOIA', newValue = NEW.exemptFromFOIA;

    # Pathing
    PATHING: BEGIN
        DECLARE vUniqueName varchar(300) default '';
        DECLARE vUniquePathName varchar(10240) default '';
        DECLARE vPathName varchar(10240) default '';
        DECLARE vParentPathName varchar(10240) default '';
        DECLARE vParentUniquePathName varchar(10240) default '';
        DECLARE vRootOwnedByGrantee varchar(255) default '';
        DECLARE vPathDelimiter varchar(5) default char(30);
        DECLARE vResourceDelimiter varchar(5) default '/';

        -- Get the unique name and initialize path as rooted
        SET vUniqueName := calcUniqueName(NEW.id, NEW.name, NEW.parentId, NEW.ownedBy);
        SET vUniquePathName := CONCAT(vPathDelimiter, vUniqueName);
        SET vPathName := CONCAT(vPathDelimiter, NEW.name);

        -- Get pathing info from parents
        IF NEW.parentId IS NULL THEN
            SELECT grantee INTO vRootOwnedByGrantee FROM acmgrantee WHERE resourceString = NEW.ownedBy or LOCATE(concat(resourceString,vResourceDelimiter),NEW.ownedBy) > 0;
            SET vUniquePathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vUniquePathName);
            SET vPathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vPathName);
        ELSE
            SELECT uniquePathName, pathName INTO vParentUniquePathName, vParentPathName FROM object_pathing WHERE id = NEW.parentId;
            SET vPathName := CONCAT(vParentPathName, vPathName);
            SET vUniquePathName := CONCAT(vParentUniquePathName, vUniquePathName);
        END IF;        
        INSERT INTO object_pathing (id, uniqueName, uniquePathName, pathName) VALUES (NEW.id, vUniqueName, vUniquePathName, vPathName);
    END PATHING;

END;
-- +migrate StatementEnd
-- +migrate StatementBegin
CREATE TRIGGER tu_object
BEFORE UPDATE ON object FOR EACH ROW
BEGIN
    DECLARE error_msg varchar(128) default '';
    DECLARE thisTableName varchar(128) default 'object';
    DECLARE count_parent int default 0;
    DECLARE count_type int default 0;
    DECLARE type_contentConnector varchar(2000) default '';

    # Rules
    # id cannot be changed
    IF (NEW.id <> OLD.id) AND length(error_msg) < 83 THEN
        SET error_msg := concat(error_msg, 'Unable to set id ');
    END IF;
    # createdDate cannot be changed
    IF (NEW.createdDate <> OLD.createdDate) AND length(error_msg) < 74 THEN
        SET error_msg := concat(error_msg, 'Unable to set createdDate ');
    END IF;
    # createdBy cannot be changed
    IF (NEW.createdBy <> OLD.createdBy) AND length(error_msg) < 76 THEN
        SET error_msg := concat(error_msg, 'Unable to set createdBy ');
    END IF;
    # changeCount cannot be changed
    IF (NEW.changeCount <> OLD.changeCount) AND length(error_msg) < 74 THEN
        SET error_msg := concat(error_msg, 'Unable to set changeCount ');
    END IF;
    # changeToken must be given and match the record
    IF (NEW.changeToken IS NULL OR NEW.changeToken = '') and length(error_msg) < 73 THEN
        SET error_msg := concat(error_msg, 'Field changeToken required ');
    END IF;
    IF (NEW.changeToken <> OLD.changeToken) and length(error_msg) < 71 THEN
        SET error_msg := concat(error_msg, 'Field changeToken must match ');
    END IF;
    # TypeId must be specified
    SELECT COUNT(*) FROM object_type WHERE isDeleted = 0 AND id = NEW.typeId INTO count_type;
    IF (count_type = 0) and length(error_msg) < 78 THEN
        SET error_msg := concat(error_msg, 'Field typeId required ');
    END IF;
    # US Persons Data is set to old value if null/empty
    IF NEW.containsUSPersonsData IS NULL OR NEW.containsUSPersonsData = '' THEN
        SET NEW.containsUSPersonsData := OLD.containsUSPersonsData;
    END IF;
    # FOIA Exempt is set to old value if null/empty
    IF NEW.exemptFromFOIA IS NULL OR NEW.exemptFromFOIA = '' THEN
        SET NEW.exemptFromFOIA := OLD.exemptFromFOIA;
    END IF;    
    # ParentId must be valid if specified
    IF NEW.parentId IS NOT NULL AND LENGTH(NEW.parentId) = 0 THEN
        SET NEW.parentId := NULL;
    END IF;
    IF NEW.parentId IS NOT NULL THEN
        SELECT COUNT(*) FROM object WHERE (isDeleted = 0 or (NEW.IsDeleted <> OLD.IsDeleted)) AND id = NEW.parentId INTO count_parent;
        IF count_parent = 0 THEN
            SET error_msg := concat(error_msg, 'Field parentId must be valid ');
        END IF;
    END IF;    
    IF length(error_msg) > 0 THEN
        SET error_msg := concat(error_msg, 'when updating record');
        signal sqlstate '45000' set message_text = error_msg;
    END IF;

    # Force values on modify
    SET NEW.modifiedDate := current_timestamp(6);
    IF (NEW.isDeleted <> OLD.isDeleted) THEN
        IF  (NEW.IsDeleted = 1) THEN
            SET NEW.deletedDate := current_timestamp(6);
            SET NEW.deletedBy := NEW.modifiedBy;
        ELSE
            SET NEW.deletedDate := NULL;
            SET NEW.deletedBy := NULL;
        END IF;                
    END IF;
    SET NEW.changeCount := OLD.changeCount + 1;
    # Standard change token formula
    SET NEW.changeToken := md5(CONCAT(CAST(OLD.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));
    # Assign Owner if not set
    IF NEW.ownedBy IS NULL OR NEW.ownedBy = '' THEN
        SET NEW.ownedBy := concat('user/', NEW.createdBy);
        SET NEW.ownedByNew := NULL;
    END IF;
    # Assign PDF availability if not set
    IF NEW.isPDFAvailable IS NULL THEN
        SET NEW.isPDFAvailable = OLD.isPDFAvailable;
    END IF;
    # Assign US Persons Data if not set
    IF NEW.containsUSPersonsData IS NULL THEN
        SET NEW.containsUSPersonsData = 'Unknown';
    END IF;
    # Assign FOIA Exempt status if not set
    IF NEW.exemptFromFOIA IS NULL THEN
        SET NEW.exemptFromFOIA = 'Unknown';
    END IF;
    # Assign stream storage state if not set
    IF NEW.isStreamStored IS NULL THEN
        SET NEW.isStreamStored = OLD.isStreamStored;
    END IF;

    # Archive table
    INSERT INTO
        a_object
    (
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,isAncestorDeleted
        ,isExpunged
        ,expungedDate
        ,expungedBy
        ,changeCount
        ,changeToken
        ,ownedBy
        ,typeId
        ,name
        ,description
        ,parentId
        ,contentConnector
        ,rawAcm
        ,contentType
        ,contentSize
        ,contentHash
        ,encryptIV
        ,ownedByNew
        ,isPDFAvailable
        ,isStreamStored
        ,containsUSPersonsData
        ,exemptFromFOIA
    ) values (
        NEW.id
        ,NEW.createdDate
        ,NEW.createdBy
        ,NEW.modifiedDate
        ,NEW.modifiedBy
        ,NEW.isDeleted
        ,NEW.deletedDate
        ,NEW.deletedBy
        ,NEW.isAncestorDeleted
        ,NEW.isExpunged
        ,NEW.expungedDate
        ,NEW.expungedBy
        ,NEW.changeCount
        ,NEW.changeToken
        ,NEW.ownedBy
        ,NEW.typeId
        ,NEW.name
        ,NEW.description
        ,NEW.parentId
        ,NEW.contentConnector
        ,NEW.rawAcm
        ,NEW.contentType
        ,NEW.contentSize
        ,NEW.contentHash
        ,NEW.encryptIV
        ,NEW.ownedByNew
        ,NEW.isPDFAvailable
        ,NEW.isStreamStored
        ,NEW.containsUSPersonsData
        ,NEW.exemptFromFOIA
    );

    # Specific field level changes
    IF NEW.ownedBy <> OLD.ownedBy THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedBy', newValue = NEW.ownedBy;
    END IF;
    IF NEW.typeId <> OLD.typeId THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'typeId', newValue = hex(NEW.typeId);
    END IF;
    IF NEW.name <> OLD.name THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'name', newValue = NEW.name;
    END IF;
    IF NEW.description <> OLD.description THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'description', newValue = NEW.description;
    END IF;
    IF NEW.parentId <> OLD.parentId THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'parentId', newValue = hex(NEW.parentId);
    END IF;
    IF NEW.contentConnector <> OLD.contentConnector THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentConnector', newValue = NEW.contentConnector;
    END IF;
    IF NEW.rawAcm <> OLD.rawAcm THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'rawAcm', newTextValue = NEW.rawAcm;
    END IF;
    IF NEW.contentType <> OLD.contentType THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentType', newValue = NEW.contentType;
    END IF;
    IF NEW.contentSize <> OLD.contentSize THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentSize', newValue = NEW.contentSize;
    END IF;
    IF NEW.contentHash <> OLD.contentHash THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentHash', newValue = hex(NEW.contentHash);
    END IF;
    IF NEW.encryptIV <> OLD.encryptIV THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'encryptIV', newValue = hex(NEW.encryptIV);
    END IF;
    IF NEW.ownedByNew <> OLD.ownedByNew THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedByNew', newValue = NEW.ownedByNew;
    END IF;
    IF NEW.isPDFAvailable <> OLD.isPDFAvailable THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isPDFAvailable', newValue = NEW.isPDFAvailable;
    END IF;
    IF NEW.isStreamStored <> OLD.isStreamStored THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isStreamStored', newValue = NEW.isStreamStored;
    END IF;
    IF NEW.containsUSPersonsData <> OLD.containsUSPersonsData THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'containsUSPersonsData', newValue = NEW.containsUSPersonsData;
    END IF;
    IF NEW.exemptFromFOIA <> OLD.exemptFromFOIA THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'exemptFromFOIA', newValue = NEW.exemptFromFOIA;
    END IF;

    # Pathing
    IF (NEW.name <> OLD.name) or (NEW.parentId <> OLD.parentId) or (NEW.ownedBy <> OLD.ownedBy) THEN
        PATHING: BEGIN
            DECLARE vUniqueName varchar(300) default '';
            DECLARE vUniquePathName varchar(10240) default '';
            DECLARE vPathName varchar(10240) default '';
            DECLARE vParentPathName varchar(10240) default '';
            DECLARE vParentUniquePathName varchar(10240) default '';
            DECLARE vRootOwnedByGrantee varchar(255) default '';
            DECLARE vOldPathName varchar(10240) default '';
            DECLARE vOldUniquePathName varchar(10240) default '';
            DECLARE vPathDelimiter varchar(5) default char(30);
            DECLARE vResourceDelimiter varchar(5) default '/';

            -- Get the unique name and initialize path as rooted
            SET vUniqueName := calcUniqueName(NEW.id, NEW.name, NEW.parentID, NEW.ownedBy);
            SET vUniquePathName := CONCAT(vPathDelimiter, vUniqueName);
            SET vPathName := CONCAT(vPathDelimiter, NEW.name);

            -- Get new pathing info from parents
            IF NEW.parentId IS NULL THEN
                SELECT grantee INTO vRootOwnedByGrantee FROM acmgrantee WHERE resourceString = NEW.ownedBy or LOCATE(concat(resourceString,vResourceDelimiter),NEW.ownedBy) > 0;
                SET vUniquePathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vUniquePathName);
                SET vPathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vPathName);
            ELSE
                SELECT uniquePathName, pathName INTO vParentUniquePathName, vParentPathName FROM object_pathing WHERE id = NEW.parentId;
                SET vPathName := CONCAT(vParentPathName, vPathName);
                SET vUniquePathName := CONCAT(vParentUniquePathName, vUniquePathName);
            END IF;        

            -- Get old pathing info
            SELECT uniquePathName, pathName INTO vOldUniquePathName, vOldPathName FROM object_pathing WHERE id = NEW.id;

            -- Update this record
            UPDATE object_pathing SET 
                uniqueName = vUniqueName, 
                uniquePathName = vUniquePathName, 
                pathName = vPathName 
            WHERE id = NEW.id;

            -- Update descendents
            UPDATE object_pathing SET 
                uniquePathName = REPLACE(uniquePathName, vOldUniquePathName, vUniquePathName),
                pathName = REPLACE(pathName, vOldPathName, vPathName)
            WHERE
                uniquePathName LIKE CONCAT(vOldUniquePathName,vPathDelimiter,'%')
                AND
                pathName LIKE CONCAT(vOldPathName,vPathDelimiter,'%');
        END PATHING;    
    END IF;

END;
-- +migrate StatementEnd

-- Recreate object_pathing
DROP TABLE IF EXISTS object_pathing;

-- create the table for storing path info
CREATE TABLE IF NOT EXISTS object_pathing
(
  id binary(16) not null default 0
  ,uniqueName varchar(300) not null
  ,uniquePathName varchar(10240) not null
  ,pathName varchar(10240) not null
  ,CONSTRAINT pk_object_pathing PRIMARY KEY (id)
  ,INDEX ix_uniqueName (uniqueName)
  ,INDEX ix_uniquePathName (uniquePathName)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;

-- a temporary procedure that will handle populating the table from existing data
DROP PROCEDURE IF EXISTS sp_Patch_PopulatePathing;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_PopulatePathing()
BEGIN
    DECLARE vID binary(16) default 0;
    DECLARE vName varchar(255) default '';
    DECLARE vParentID binary(16) default 0;
    DECLARE vOwnedBy varchar(255) default '';

    DECLARE vUniqueName varchar(300) default '';
    DECLARE vUniquePathName varchar(10240) default '';
    DECLARE vPathName varchar(10240) default '';
    DECLARE vParentPathName varchar(10240) default '';
    DECLARE vParentUniquePathName varchar(10240) default '';
    DECLARE vRootOwnedByGrantee varchar(255) default '';
    DECLARE vParentPathCalculated int default 0;
    DECLARE vMatchingGranteeCount int default 0;
    DECLARE vPathDelimiter varchar(5) default char(30);
    DECLARE vResourceDelimiter varchar(5) default '/';

    -- vGoAgain is a flag that tracks if there are still parents that havent been mapped that a child depends on for pathing
    DECLARE vGoAgain int default 1;
    WHILE vGoAgain > 0 DO
        OVERALL: BEGIN
            DECLARE error_msg varchar(128) default '';
            DECLARE finished_objects int default 0;
            DECLARE cursor_objects cursor for SELECT o.id, o.name, o.parentid, o.ownedBy from object o left outer join object_pathing op on o.id = op.id where op.id is null;
            DECLARE continue handler for not found set finished_objects = 1;
            SET vGoAgain := 0;
            OPEN cursor_objects;
            get_object: LOOP
                FETCH cursor_objects INTO vID, vName, vParentID, vOwnedBy;
                IF finished_objects = 1 THEN
                    CLOSE cursor_objects;
                    LEAVE get_object;
                END IF;

                -- Get the unique name and initialize path as rooted
                SET vUniqueName := calcUniqueName(vID, vName, vParentID, vOwnedBy);
                SET vUniquePathName := CONCAT(vPathDelimiter, vUniqueName);

                -- Initialize this objects path name. All paths start with a record separator
                SET vPathName := CONCAT(vPathDelimiter, vName);

                -- Determine if processing a rooted object or one where we need parent info
                IF vParentID IS NULL THEN
                    -- Rooted. we're responsible for building on this pass'
                    select count(0) into vMatchingGranteeCount FROM acmgrantee WHERE
                        resourceString = vOwnedBy 
                        or  LOCATE(concat(resourceString,vResourceDelimiter), vOwnedBy) > 0
                        or  LOCATE(grantee,vOwnedBy) > 0
                        ;
                    IF vMatchingGranteeCount = 0 THEN
                        -- This should be an error...
                        --   SET error_msg := concat('No match for owner [', vOwnedBy, ']');
                        --   SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = error_msg;
                        -- But for accomodating junk test data and an open bug for ownedby vs acmgrantee.resourcestring permit it
                        SELECT vOwnedBy INTO vRootOwnedByGrantee;
                    ELSE
                        SELECT grantee INTO vRootOwnedByGrantee FROM acmgrantee WHERE 
                                resourceString = vOwnedBy 
                            or  LOCATE(concat(resourceString,vResourceDelimiter), vOwnedBy) > 0
                            or  LOCATE(grantee,vOwnedBy) > 0
                            ;
                    END IF;
                    SET vUniquePathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vUniquePathName);
                    SET vPathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vPathName);
                    INSERT INTO object_pathing (id, uniqueName, uniquePathName, pathName) VALUES (vID, vUniqueName, vUniquePathName, vPathName);
                ELSE
                    -- Determine if parents pathing has been calculated yet
                    SELECT count(0) INTO vParentPathCalculated FROM object_pathing WHERE id = vParentID;
                    IF vParentPathCalculated > 0 THEN
                        -- Get pathing info from parents as the base
                        SELECT uniquePathName, pathName INTO vParentUniquePathName, vParentPathName FROM object_pathing WHERE id = vParentID;
                        -- Assemble our paths and insert us
                        SET vPathName := CONCAT(vParentPathName, vPathName);
                        SET vUniquePathName := CONCAT(vParentUniquePathName, vUniquePathName);
                        INSERT INTO object_pathing (id, uniqueName, uniquePathName, pathName) VALUES (vID, vUniqueName, vUniquePathName, vPathName);
                    ELSE
                        -- Parent not yet calculated. Force another full iteration of unprocessed objects
                        SET vGoAgain := 1;
                    END IF;
                END IF;
            END LOOP get_object;
        END OVERALL;
    END WHILE;
END
-- +migrate StatementEnd
CALL sp_Patch_PopulatePathing();
DROP PROCEDURE IF EXISTS sp_Patch_PopulatePathing;

DROP TRIGGER IF EXISTS ti_dbstate;
-- +migrate StatementBegin
CREATE TRIGGER ti_dbstate
BEFORE INSERT ON dbstate FOR EACH ROW
BEGIN
	DECLARE count_rows int default 0;

	# Rules
	# Can only be one record
	SELECT count(0) FROM dbstate INTO count_rows;
	IF count_rows > 0 THEN
		signal sqlstate '45000' set message_text = 'Only one record is allowed in dbstate table.';
	END IF;

	# Force values on create
	# Created Date
	SET NEW.createdDate := current_timestamp(6);
	# Modified Date
	SET NEW.modifiedDate := current_timestamp(6);
	# Version should be changed if the schema changes
	SET NEW.schemaversion := '20170301'; 
	# Identifier is randomized as a GUID
	SET NEW.identifier := concat(@@hostname, '-', left(uuid(),8));
END;
-- +migrate StatementEnd
update dbstate set schemaVersion = '20170301';


-- +migrate Down

-- remove triggers on object for create/update
DROP TRIGGER IF EXISTS ti_object;
DROP TRIGGER IF EXISTS tu_object;

-- recreate triggers for create/update as in 20161230
-- +migrate StatementBegin
CREATE TRIGGER ti_object
BEFORE INSERT ON object FOR EACH ROW
BEGIN
    DECLARE error_msg varchar(128) default '';
    DECLARE thisTableName varchar(128) default 'object';
    DECLARE count_parent int default 0;
    DECLARE count_type int default 0;
    DECLARE type_contentConnector varchar(2000) default '';

    # Rules
    # Type must be specified
    SELECT COUNT(*) FROM object_type WHERE isDeleted = 0 AND id = NEW.typeId INTO count_type;
    IF count_type = 0 THEN
        SET error_msg := concat(error_msg, 'Field typeId required ');
    END IF;
    # US Persons Data is NULLed if empty (will change to Unknown)
    IF NEW.containsUSPersonsData IS NOT NULL AND NEW.containsUSPersonsData = '' THEN
        SET NEW.containsUSPersonsData := NULL;
    END IF;
    # FOIA Exempt is NULLed if empty (will change to Unknown)
    IF NEW.exemptFromFOIA IS NOT NULL AND NEW.exemptFromFOIA = '' THEN
        SET NEW.exemptFromFOIA := NULL;
    END IF;
    # ParentId must be valid if specified
    IF NEW.parentId IS NOT NULL AND NEW.parentId = '' THEN
        SET NEW.parentId := NULL;
    END IF;
    IF NEW.parentId IS NOT NULL THEN
        SELECT COUNT(*) FROM object WHERE isDeleted = 0 AND id = NEW.parentId INTO count_parent;
        IF count_parent = 0 THEN
            SET error_msg := concat(error_msg, 'Field parentId must be valid ');
        END IF;
    END IF;
    IF error_msg <> '' THEN
        SET error_msg := concat(error_msg, 'when inserting record into ', thisTableName);
        signal sqlstate '45000' set message_text = error_msg;
    END IF;

    # Force values on create
    SET NEW.id := ordered_uuid(UUID());
    SET NEW.createdDate := current_timestamp(6);
    SET NEW.modifiedDate := current_timestamp(6);
    SET NEW.modifiedBy := NEW.createdBy;
    SET NEW.isDeleted := 0;
    SET NEW.deletedDate := NULL;
    SET NEW.deletedBy := NULL;
    SET NEW.isAncestorDeleted := 0;
    SET NEW.isExpunged := 0;
    SET NEW.expungedDate := NULL;
    SET NEW.expungedBy := NULL;
    SET NEW.ownedByNew := NULL;
    # Assign Owner if not set
    IF NEW.ownedBy IS NULL OR NEW.ownedBy = '' THEN
        SET NEW.ownedBy := concat('user/', NEW.createdBy);
    END IF;
    SET NEW.changeCount := 0;
    # Standard change token formula
    SET NEW.changeToken := md5(CONCAT(CAST(NEW.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));
    # Assign contentConnector if not set
    IF NEW.contentConnector IS NULL OR NEW.contentConnector = '' THEN
        SELECT contentConnector FROM object_type WHERE isDeleted = 0 AND id = NEW.typeId INTO type_contentConnector;
        SET NEW.contentConnector := type_contentConnector;
    END IF;
    # Assign PDF availability if not set
    IF NEW.isPDFAvailable IS NULL THEN
        SET NEW.isPDFAvailable := 0;
    END IF;
    # Assign US Persons Data if not set
    IF NEW.containsUSPersonsData IS NULL THEN
        SET NEW.containsUSPersonsData := 'Unknown';
    END IF;
    # Assign FOIA Exempt status if not set
    IF NEW.exemptFromFOIA IS NULL THEN
        SET NEW.exemptFromFOIA := 'Unknown';
    END IF;
    # Assign stream storage state if not set
    IF NEW.isStreamStored IS NULL THEN
        SET NEW.isStreamStored := 0;
    END IF;

    # Archive table
    INSERT INTO
        a_object
    (
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,isAncestorDeleted
        ,isExpunged
        ,expungedDate
        ,expungedBy
        ,changeCount
        ,changeToken
        ,ownedBy
        ,typeId
        ,name
        ,description
        ,parentId
        ,contentConnector
        ,rawAcm
        ,contentType
        ,contentSize
        ,contentHash
        ,encryptIV
        ,ownedByNew
        ,isPDFAvailable
        ,isStreamStored
        ,containsUSPersonsData
        ,exemptFromFOIA
    ) values (
        NEW.id
        ,NEW.createdDate
        ,NEW.createdBy
        ,NEW.modifiedDate
        ,NEW.modifiedBy
        ,NEW.isDeleted
        ,NEW.deletedDate
        ,NEW.deletedBy
        ,NEW.isAncestorDeleted
        ,NEW.isExpunged
        ,NEW.expungedDate
        ,NEW.expungedBy
        ,NEW.changeCount
        ,NEW.changeToken
        ,NEW.ownedBy
        ,NEW.typeId
        ,NEW.name
        ,NEW.description
        ,NEW.parentId
        ,NEW.contentConnector
        ,NEW.rawAcm
        ,NEW.contentType
        ,NEW.contentSize
        ,NEW.contentHash
        ,NEW.encryptIV
        ,NEW.ownedByNew
        ,NEW.isPDFAvailable
        ,NEW.isStreamStored
        ,NEW.containsUSPersonsData
        ,NEW.exemptFromFOIA
    );

    # Specific field level changes
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedBy', newValue = NEW.ownedBy;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'typeId', newValue = hex(NEW.typeId);
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'name', newValue = NEW.name;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'description', newValue = NEW.description;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'parentId', newValue = hex(NEW.parentId);
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentConnector', newValue = NEW.contentConnector;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'rawAcm', newTextValue = NEW.rawAcm;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentType', newValue = NEW.contentType;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentSize', newValue = NEW.contentSize;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentHash', newValue = hex(NEW.contentHash);
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'encryptIV', newValue = hex(NEW.encryptIV);
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedByNew', newValue = NEW.ownedByNew;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isPDFAvailable', newValue = NEW.isPDFAvailable;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isStreamStored', newValue = NEW.isStreamStored;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'containsUSPersonsData', newValue = NEW.containsUSPersonsData;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'exemptFromFOIA', newValue = NEW.exemptFromFOIA;

    # Pathing
    PATHING: BEGIN
        DECLARE vUniqueName varchar(300) default '';
        DECLARE vUniquePathName varchar(10240) default '';
        DECLARE vPathName varchar(10240) default '';
        DECLARE vParentPathName varchar(10240) default '';
        DECLARE vParentUniquePathName varchar(10240) default '';
        DECLARE vRootOwnedByGrantee varchar(255) default '';
        DECLARE vPathDelimiter varchar(5) default char(30);
        DECLARE vResourceDelimiter varchar(5) default '/';

        -- Get the unique name and initialize path as rooted
        SET vUniqueName := calcUniqueName(NEW.id, NEW.name, NEW.parentId, NEW.ownedBy);
        SET vUniquePathName := CONCAT(vPathDelimiter, vUniqueName);
        SET vPathName := CONCAT(vPathDelimiter, NEW.name);

        -- Get pathing info from parents
        IF NEW.parentId IS NULL THEN
            SELECT grantee INTO vRootOwnedByGrantee FROM acmgrantee WHERE resourceString = NEW.ownedBy or LOCATE(concat(resourceString,vResourceDelimiter),NEW.ownedBy) > 0;
            SET vUniquePathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vUniquePathName);
            SET vPathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vPathName);
        ELSE
            SELECT uniquePathName, pathName INTO vParentUniquePathName, vParentPathName FROM object_pathing WHERE id = NEW.parentId;
            SET vPathName := CONCAT(vParentPathName, vPathName);
            SET vUniquePathName := CONCAT(vParentUniquePathName, vUniquePathName);
        END IF;        
        INSERT INTO object_pathing (id, uniqueName, uniquePathName, pathName) VALUES (NEW.id, vUniqueName, vUniquePathName, vPathName);
    END PATHING;

END;
-- +migrate StatementEnd
-- +migrate StatementBegin
CREATE TRIGGER tu_object
BEFORE UPDATE ON object FOR EACH ROW
BEGIN
    DECLARE error_msg varchar(128) default '';
    DECLARE thisTableName varchar(128) default 'object';
    DECLARE count_parent int default 0;
    DECLARE count_type int default 0;
    DECLARE type_contentConnector varchar(2000) default '';

    # Rules
    # id cannot be changed
    IF (NEW.id <> OLD.id) AND length(error_msg) < 83 THEN
        SET error_msg := concat(error_msg, 'Unable to set id ');
    END IF;
    # createdDate cannot be changed
    IF (NEW.createdDate <> OLD.createdDate) AND length(error_msg) < 74 THEN
        SET error_msg := concat(error_msg, 'Unable to set createdDate ');
    END IF;
    # createdBy cannot be changed
    IF (NEW.createdBy <> OLD.createdBy) AND length(error_msg) < 76 THEN
        SET error_msg := concat(error_msg, 'Unable to set createdBy ');
    END IF;
    # changeCount cannot be changed
    IF (NEW.changeCount <> OLD.changeCount) AND length(error_msg) < 74 THEN
        SET error_msg := concat(error_msg, 'Unable to set changeCount ');
    END IF;
    # changeToken must be given and match the record
    IF (NEW.changeToken IS NULL OR NEW.changeToken = '') and length(error_msg) < 73 THEN
        SET error_msg := concat(error_msg, 'Field changeToken required ');
    END IF;
    IF (NEW.changeToken <> OLD.changeToken) and length(error_msg) < 71 THEN
        SET error_msg := concat(error_msg, 'Field changeToken must match ');
    END IF;
    # TypeId must be specified
    SELECT COUNT(*) FROM object_type WHERE isDeleted = 0 AND id = NEW.typeId INTO count_type;
    IF (count_type = 0) and length(error_msg) < 78 THEN
        SET error_msg := concat(error_msg, 'Field typeId required ');
    END IF;
    # US Persons Data is set to old value if null/empty
    IF NEW.containsUSPersonsData IS NULL OR NEW.containsUSPersonsData = '' THEN
        SET NEW.containsUSPersonsData := OLD.containsUSPersonsData;
    END IF;
    # FOIA Exempt is set to old value if null/empty
    IF NEW.exemptFromFOIA IS NULL OR NEW.exemptFromFOIA = '' THEN
        SET NEW.exemptFromFOIA := OLD.exemptFromFOIA;
    END IF;    
    # ParentId must be valid if specified
    IF NEW.parentId IS NOT NULL AND LENGTH(NEW.parentId) = 0 THEN
        SET NEW.parentId := NULL;
    END IF;
    IF NEW.parentId IS NOT NULL THEN
        SELECT COUNT(*) FROM object WHERE (isDeleted = 0 or (NEW.IsDeleted <> OLD.IsDeleted)) AND id = NEW.parentId INTO count_parent;
        IF count_parent = 0 THEN
            SET error_msg := concat(error_msg, 'Field parentId must be valid ');
        END IF;
    END IF;    
    IF length(error_msg) > 0 THEN
        SET error_msg := concat(error_msg, 'when updating record');
        signal sqlstate '45000' set message_text = error_msg;
    END IF;

    # Force values on modify
    SET NEW.modifiedDate := current_timestamp(6);
    IF (NEW.isDeleted <> OLD.isDeleted) THEN
        IF  (NEW.IsDeleted = 1) THEN
            SET NEW.deletedDate := current_timestamp(6);
            SET NEW.deletedBy := NEW.modifiedBy;
        ELSE
            SET NEW.deletedDate := NULL;
            SET NEW.deletedBy := NULL;
        END IF;                
    END IF;
    SET NEW.changeCount := OLD.changeCount + 1;
    # Standard change token formula
    SET NEW.changeToken := md5(CONCAT(CAST(OLD.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));
    # Assign Owner if not set
    IF NEW.ownedBy IS NULL OR NEW.ownedBy = '' THEN
        SET NEW.ownedBy := concat('user/', NEW.createdBy);
        SET NEW.ownedByNew := NULL;
    END IF;
    # Assign PDF availability if not set
    IF NEW.isPDFAvailable IS NULL THEN
        SET NEW.isPDFAvailable = OLD.isPDFAvailable;
    END IF;
    # Assign US Persons Data if not set
    IF NEW.containsUSPersonsData IS NULL THEN
        SET NEW.containsUSPersonsData = 'Unknown';
    END IF;
    # Assign FOIA Exempt status if not set
    IF NEW.exemptFromFOIA IS NULL THEN
        SET NEW.exemptFromFOIA = 'Unknown';
    END IF;
    # Assign stream storage state if not set
    IF NEW.isStreamStored IS NULL THEN
        SET NEW.isStreamStored = OLD.isStreamStored;
    END IF;

    # Archive table
    INSERT INTO
        a_object
    (
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,isAncestorDeleted
        ,isExpunged
        ,expungedDate
        ,expungedBy
        ,changeCount
        ,changeToken
        ,ownedBy
        ,typeId
        ,name
        ,description
        ,parentId
        ,contentConnector
        ,rawAcm
        ,contentType
        ,contentSize
        ,contentHash
        ,encryptIV
        ,ownedByNew
        ,isPDFAvailable
        ,isStreamStored
        ,containsUSPersonsData
        ,exemptFromFOIA
    ) values (
        NEW.id
        ,NEW.createdDate
        ,NEW.createdBy
        ,NEW.modifiedDate
        ,NEW.modifiedBy
        ,NEW.isDeleted
        ,NEW.deletedDate
        ,NEW.deletedBy
        ,NEW.isAncestorDeleted
        ,NEW.isExpunged
        ,NEW.expungedDate
        ,NEW.expungedBy
        ,NEW.changeCount
        ,NEW.changeToken
        ,NEW.ownedBy
        ,NEW.typeId
        ,NEW.name
        ,NEW.description
        ,NEW.parentId
        ,NEW.contentConnector
        ,NEW.rawAcm
        ,NEW.contentType
        ,NEW.contentSize
        ,NEW.contentHash
        ,NEW.encryptIV
        ,NEW.ownedByNew
        ,NEW.isPDFAvailable
        ,NEW.isStreamStored
        ,NEW.containsUSPersonsData
        ,NEW.exemptFromFOIA
    );

    # Specific field level changes
    IF NEW.ownedBy <> OLD.ownedBy THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedBy', newValue = NEW.ownedBy;
    END IF;
    IF NEW.typeId <> OLD.typeId THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'typeId', newValue = hex(NEW.typeId);
    END IF;
    IF NEW.name <> OLD.name THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'name', newValue = NEW.name;
    END IF;
    IF NEW.description <> OLD.description THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'description', newValue = NEW.description;
    END IF;
    IF NEW.parentId <> OLD.parentId THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'parentId', newValue = hex(NEW.parentId);
    END IF;
    IF NEW.contentConnector <> OLD.contentConnector THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentConnector', newValue = NEW.contentConnector;
    END IF;
    IF NEW.rawAcm <> OLD.rawAcm THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'rawAcm', newTextValue = NEW.rawAcm;
    END IF;
    IF NEW.contentType <> OLD.contentType THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentType', newValue = NEW.contentType;
    END IF;
    IF NEW.contentSize <> OLD.contentSize THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentSize', newValue = NEW.contentSize;
    END IF;
    IF NEW.contentHash <> OLD.contentHash THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentHash', newValue = hex(NEW.contentHash);
    END IF;
    IF NEW.encryptIV <> OLD.encryptIV THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'encryptIV', newValue = hex(NEW.encryptIV);
    END IF;
    IF NEW.ownedByNew <> OLD.ownedByNew THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedByNew', newValue = NEW.ownedByNew;
    END IF;
    IF NEW.isPDFAvailable <> OLD.isPDFAvailable THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isPDFAvailable', newValue = NEW.isPDFAvailable;
    END IF;
    IF NEW.isStreamStored <> OLD.isStreamStored THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isStreamStored', newValue = NEW.isStreamStored;
    END IF;
    IF NEW.containsUSPersonsData <> OLD.containsUSPersonsData THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'containsUSPersonsData', newValue = NEW.containsUSPersonsData;
    END IF;
    IF NEW.exemptFromFOIA <> OLD.exemptFromFOIA THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'exemptFromFOIA', newValue = NEW.exemptFromFOIA;
    END IF;

    # Pathing
    IF (NEW.name <> OLD.name) or (NEW.parentId <> OLD.parentId) or (NEW.ownedBy <> OLD.ownedBy) THEN
        PATHING: BEGIN
            DECLARE vUniqueName varchar(300) default '';
            DECLARE vUniquePathName varchar(10240) default '';
            DECLARE vPathName varchar(10240) default '';
            DECLARE vParentPathName varchar(10240) default '';
            DECLARE vParentUniquePathName varchar(10240) default '';
            DECLARE vRootOwnedByGrantee varchar(255) default '';
            DECLARE vOldPathName varchar(10240) default '';
            DECLARE vOldUniquePathName varchar(10240) default '';
            DECLARE vPathDelimiter varchar(5) default char(30);
            DECLARE vResourceDelimiter varchar(5) default '/';

            -- Get the unique name and initialize path as rooted
            SET vUniqueName := calcUniqueName(NEW.id, NEW.name, NEW.parentID, NEW.ownedBy);
            SET vUniquePathName := CONCAT(vPathDelimiter, vUniqueName);
            SET vPathName := CONCAT(vPathDelimiter, NEW.name);

            -- Get new pathing info from parents
            IF NEW.parentId IS NULL THEN
                SELECT grantee INTO vRootOwnedByGrantee FROM acmgrantee WHERE resourceString = NEW.ownedBy or LOCATE(concat(resourceString,vResourceDelimiter),NEW.ownedBy) > 0;
                SET vUniquePathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vUniquePathName);
                SET vPathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vPathName);
            ELSE
                SELECT uniquePathName, pathName INTO vParentUniquePathName, vParentPathName FROM object_pathing WHERE id = NEW.parentId;
                SET vPathName := CONCAT(vParentPathName, vPathName);
                SET vUniquePathName := CONCAT(vParentUniquePathName, vUniquePathName);
            END IF;        

            -- Get old pathing info
            SELECT uniquePathName, pathName INTO vOldUniquePathName, vOldPathName FROM object_pathing WHERE id = NEW.id;

            -- Update this record
            UPDATE object_pathing SET 
                uniqueName = vUniqueName, 
                uniquePathName = vUniquePathName, 
                pathName = vPathName 
            WHERE id = NEW.id;

            -- Update descendents
            UPDATE object_pathing SET 
                uniquePathName = REPLACE(uniquePathName, vOldUniquePathName, vUniquePathName),
                pathName = REPLACE(pathName, vOldPathName, vPathName)
            WHERE
                uniquePathName LIKE CONCAT(vOldUniquePathName,vPathDelimiter,'%')
                AND
                pathName LIKE CONCAT(vOldPathName,vPathDelimiter,'%');
        END PATHING;    
    END IF;

END;
-- +migrate StatementEnd

DROP TRIGGER IF EXISTS ti_dbstate;

-- +migrate StatementBegin
CREATE TRIGGER ti_dbstate
BEFORE INSERT ON dbstate FOR EACH ROW
BEGIN
	DECLARE count_rows int default 0;

	# Rules
	# Can only be one record
	SELECT count(0) FROM dbstate INTO count_rows;
	IF count_rows > 0 THEN
		signal sqlstate '45000' set message_text = 'Only one record is allowed in dbstate table.';
	END IF;

	# Force values on create
	# Created Date
	SET NEW.createdDate := current_timestamp(6);
	# Modified Date
	SET NEW.modifiedDate := current_timestamp(6);
	# Version should be changed if the schema changes
	SET NEW.schemaversion := '20161230'; 
	# Identifier is randomized as a GUID
	SET NEW.identifier := concat(@@hostname, '-', left(uuid(),8));
END;
-- +migrate StatementEnd
update dbstate set schemaVersion = '20161230';