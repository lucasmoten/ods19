-- +migrate Up

-- drop triggers affecting data we are migrating
INSERT INTO migration_status SET description = '20170505_738_calcResourceString drop triggers on object table';
DROP TRIGGER IF EXISTS ti_object;
DROP TRIGGER IF EXISTS tu_object;

-- force resource string to be lowercase
INSERT INTO migration_status SET description = '20170505_738_calcResourceString lowercase values in acmgrantee table';
UPDATE acmgrantee SET 
    grantee = lower(grantee)
    ,resourcestring = lower(resourcestring)
    ,projectname = lower(projectname)
    ,projectdisplayname = lower(projectdisplayname)
    ,groupname = lower(groupname)
    ,userdistinguishedname = lower(userdistinguishedname)
    ,displayname = lower(displayname)
;

-- force ownedby to be lowercase
INSERT INTO migration_status SET description = '20170505_738_calcResourceString lowercase ownedby in object table';
UPDATE object SET ownedby = lower(ownedby);
UPDATE a_object SET ownedby = lower(ownedby);


-- a function to add resource string to acmgrantee if not yet present this function will break up the parts of a 
-- resource string and check if its in acmgrantee and get normalized resource string from acmgrantee if parsed 
-- grantee value is already present but resourcestring value differs.
INSERT INTO migration_status SET description = '20170505_738_calcResourceString recreating function calcResourceString';
DROP FUNCTION IF EXISTS calcResourceString;
-- +migrate StatementBegin
CREATE FUNCTION calcResourceString(vOriginalString varchar(300)) RETURNS varchar(300)
BEGIN
    DECLARE vParts int default 0;
    DECLARE vPart1 varchar(255) default ''; -- type
    DECLARE vPart2 varchar(255) default ''; -- user dn, short group name, project name
    DECLARE vPart3 varchar(255) default ''; -- user display name, short group display name, full project display name
    DECLARE vPart4 varchar(255) default ''; -- full group name
    DECLARE vPart5 varchar(255) default ''; -- full display name
    DECLARE vLowerString varchar(300) default '';
    DECLARE vResourceString varchar(300) default '';
    DECLARE vGrantee varchar(255) default '';
    DECLARE vDisplayName varchar(255) default '';

    -- As of 2017-05-05 Always forced to lowercase
    SET vLowerString := LOWER(vOriginalString);
    SELECT (length(vLowerString)-length(replace(vLowerString,'/',''))) + 1 INTO vParts;
    SELECT substring_index(vLowerString,'/',1) INTO vPart1;
    SELECT substring_index(substring_index(vLowerString,'/',2),'/',-1) INTO vPart2;
    SELECT substring_index(substring_index(vLowerString,'/',3),'/',-1) INTO vPart3;
    SELECT substring_index(substring_index(vLowerString,'/',4),'/',-1) INTO vPart4;
    SELECT substring_index(substring_index(vLowerString,'/',5),'/',-1) INTO vPart5;

    IF vParts > 1 AND (vPart1 = 'user' or vPart1 = 'group') THEN
        -- Calculate resource string and grantee, check if exists in acmgrantee, inserting as needed
        IF vPart1 = 'user' THEN
            SET vResourceString := CONCAT(vPart1, '/', vPart2);
            SET vGrantee := aacflatten(vPart2);
            IF (SELECT 1=1 FROM acmgrantee WHERE binary resourcestring = vResourceString) IS NULL THEN
                IF (select 1=1 FROM acmgrantee WHERE binary grantee = vGrantee) IS NULL THEN
                    IF vParts > 2 THEN
                        SET vDisplayName := vPart3;
                    ELSE
                        SET vDisplayName := replace(replace(substring_index(vPart2,',',1),'cn=',''),'CN=','');
                    END IF;
                    INSERT INTO acmgrantee SET 
                        grantee = vGrantee, 
                        resourcestring = vResourceString, 
                        projectName = null, 
                        projectDisplayName = null,
                        groupName = null,
                        userDistinguishedName = vPart2,
                        displayName = vDisplayName;
                ELSE
                    SELECT resourcestring INTO vResourceString FROM acmgrantee WHERE binary grantee = vGrantee LIMIT 1;
                END IF;
            END IF;
        END IF;
        IF vPart1 = 'group' THEN
            IF vParts <= 3 THEN
                -- Pseudo group (i.e., Everyone)
                SET vResourceString := CONCAT(vPart1, '/', vPart2);
                SET vGrantee := aacflatten(vPart2);
                IF (SELECT 1=1 FROM acmgrantee WHERE binary resourcestring = vResourceString) IS NULL THEN
                    IF (select 1=1 FROM acmgrantee WHERE binary grantee = vGrantee) IS NULL THEN
                        IF vParts > 2 THEN
                            SET vDisplayName := vPart3;
                        ELSE
                            SET vDisplayName := vPart2;
                        END IF;
                        INSERT INTO acmgrantee SET 
                            grantee = vGrantee, 
                            resourcestring = vResourceString, 
                            projectName = null, 
                            projectDisplayName = null,
                            groupName = vPart2,
                            userDistinguishedName = null,
                            displayName = vDisplayName;
                    ELSE
                        SELECT resourcestring INTO vResourceString FROM acmgrantee WHERE binary grantee = vGrantee LIMIT 1;
                    END IF;
                END IF;                                
            END IF;
            IF vParts > 3 THEN
                -- Typical groups
                SET vResourceString := CONCAT(vPart1, '/', vPart2, '/', vPart3, '/', vPart4);
                SET vGrantee := aacflatten(CONCAT(vPart2,'_',vPart4));
                IF (SELECT 1=1 FROM acmgrantee WHERE binary resourcestring = vResourceString) IS NULL THEN
                    IF (select 1=1 FROM acmgrantee WHERE binary grantee = vGrantee) IS NULL THEN
                        IF vParts > 4 THEN
                            SET vDisplayName := vPart5;
                        ELSE
                            SET vDisplayName := CONCAT(vPart3, ' ', vPart4);
                        END IF;
                        INSERT INTO acmgrantee SET 
                            grantee = vGrantee, 
                            resourcestring = vResourceString, 
                            projectName = vPart2, 
                            projectDisplayName = vPart3,
                            groupName = vPart4,
                            userDistinguishedName = null,
                            displayName = vDisplayName;
                    ELSE
                        SELECT resourcestring INTO vResourceString FROM acmgrantee WHERE binary grantee = vGrantee LIMIT 1;
                    END IF;
                END IF;   
            END IF;
        END IF;
        -- See if grantee exists in acmvalue2
        IF (SELECT 1=1 FROM acmvalue2 WHERE binary name = vGrantee) IS NULL THEN
            INSERT INTO acmvalue2 (name) VALUES (vGrantee);
        END IF;        
    ELSE
        SET vResourceString := '';
    END IF;
	RETURN vResourceString;
END;
-- +migrate StatementEnd

INSERT INTO migration_status SET description = '20170505_738_calcResourceString recreating triggers on object';
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
    SET NEW.ownedBy := calcResourceString(NEW.ownedBy);
    SET NEW.ownedByID := calcGranteeIDFromResourceString(NEW.ownedBy);
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
        ,acmId
        ,ownedById
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
        ,NEW.acmId
        ,NEW.ownedById
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
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmId', newValue = NEW.acmId;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedById', newValue = NEW.ownedById;

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
            SELECT grantee INTO vRootOwnedByGrantee FROM acmgrantee WHERE resourceString = NEW.ownedBy or LOCATE(concat(resourceString,vResourceDelimiter),NEW.ownedBy) > 0 LIMIT 1;
            SET vUniquePathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vUniquePathName);
            SET vPathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vPathName);
        ELSE
            SELECT uniquePathName, pathName INTO vParentUniquePathName, vParentPathName FROM object_pathing WHERE id = NEW.parentId LIMIT 1;
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
    SET NEW.ownedBy := calcResourceString(NEW.ownedBy);
    # ownedByID derived from ownedBy
    SET NEW.ownedByID := calcGranteeIDFromResourceString(NEW.ownedBy);
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
        ,acmId
        ,ownedById
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
        ,NEW.acmId
        ,NEW.ownedById
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
    IF NEW.acmId <> OLD.acmId THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmId', newValue = NEW.acmId;
    END IF;
    IF NEW.ownedById <> OLD.ownedById THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedById', newValue = NEW.ownedById;
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
                SELECT grantee INTO vRootOwnedByGrantee FROM acmgrantee WHERE resourceString = NEW.ownedBy or LOCATE(concat(resourceString,vResourceDelimiter),NEW.ownedBy) > 0 LIMIT 1;
                SET vUniquePathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vUniquePathName);
                SET vPathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vPathName);
            ELSE
                SELECT uniquePathName, pathName INTO vParentUniquePathName, vParentPathName FROM object_pathing WHERE id = NEW.parentId LIMIT 1;
                SET vPathName := CONCAT(vParentPathName, vPathName);
                SET vUniquePathName := CONCAT(vParentUniquePathName, vUniquePathName);
            END IF;        

            -- Get old pathing info
            SELECT uniquePathName, pathName INTO vOldUniquePathName, vOldPathName FROM object_pathing WHERE id = NEW.id LIMIT 1;

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


INSERT INTO migration_status SET description = '20170505_738_calcResourceString creating triggers on acmgrantee';
DROP TRIGGER IF EXISTS ti_acmgrantee;
-- +migrate StatementBegin
CREATE TRIGGER ti_acmgrantee
BEFORE INSERT ON acmgrantee FOR EACH ROW
BEGIN
    DECLARE error_msg varchar(128) default '';
    DECLARE thisTableName varchar(128) default 'acmgrantee';
    # All fields lowercase
    IF NEW.grantee IS NOT NULL THEN
        SET NEW.grantee := LOWER(NEW.grantee);
    END IF;
    IF NEW.resourceString IS NOT NULL THEN
        SET NEW.resourceString := LOWER(NEW.resourceString);
    END IF;
    IF NEW.projectName IS NOT NULL THEN
        SET NEW.projectName := LOWER(NEW.projectName);
    END IF;
    IF NEW.projectDisplayName IS NOT NULL THEN
        SET NEW.projectDisplayName := LOWER(NEW.projectDisplayName);
    END IF;
    IF NEW.groupName IS NOT NULL THEN
        SET NEW.groupName := LOWER(NEW.groupName);
    END IF;
    IF NEW.userDistinguishedName IS NOT NULL THEN
        SET NEW.userDistinguishedName := LOWER(NEW.userDistinguishedName);
    END IF;
    IF NEW.displayName IS NOT NULL THEN
        SET NEW.displayName := LOWER(NEW.displayName);
    END IF;    
    # Check required fields
    IF NEW.grantee IS NULL THEN
        SET error_msg := concat(error_msg, 'Field grantee must be set ');
    ELSE
        IF (SELECT 1=1 FROM acmgrantee WHERE binary grantee = NEW.grantee) IS NOT NULL THEN
            SET error_msg := concat(error_msg, 'Field grantee must be unique ');
        END IF;
    END IF;
    IF NEW.resourceString IS NULL THEN
        SET error_msg := concat(error_msg, 'Field resourceString must be set ');
    ELSE
        IF (SELECT 1=1 FROM acmgrantee WHERE binary resourceString = NEW.resourceString) IS NOT NULL THEN
            SET error_msg := concat(error_msg, 'Field resourceString must be unique ');
        END IF;
    END IF;
    IF length(error_msg) > 0 THEN
        SET error_msg := concat(error_msg, 'when inserting record');
        signal sqlstate '45000' set message_text = error_msg;
    END IF;
    # Add grantee to acmvalue2 is not yet present
    IF (SELECT 1=1 FROM acmvalue2 WHERE binary name = NEW.grantee) IS NULL THEN
        INSERT INTO acmvalue2 (name) VALUES (NEW.grantee);
    END IF;
END;
-- +migrate StatementEnd

-- dbstate
DROP TRIGGER IF EXISTS ti_dbstate;
INSERT INTO migration_status SET description = '20170505_738_calcResourceString.sql setting schema version';
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
	SET NEW.schemaversion := '20170505'; 
	# Identifier is randomized as a GUID
	SET NEW.identifier := concat(@@hostname, '-', left(uuid(),8));
END;
-- +migrate StatementEnd
update dbstate set schemaVersion = '20170505' where schemaVersion <> '20170505';

-- +migrate Down
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
	SET NEW.schemaversion := '20170421'; 
	# Identifier is randomized as a GUID
	SET NEW.identifier := concat(@@hostname, '-', left(uuid(),8));
END;
-- +migrate StatementEnd
update dbstate set schemaVersion = '20170421' where schemaVersion <> '20170421';