-- +migrate Up

-- 1. For any objects containing reserved characters (forward and backward slash) in their name, they will be
--    transformed to represent 1 or more objects with a hierarchy using those characters as delimiters for
--    folder pathing.  For any intermediate folders created, the permissions and ACM on those folders will be
--    set to the same as that for the object from which it was derived with same ownership.
--    NOTE: This migration requires the master key to be set in environment variable
--    NOTE: One way conversion. There is no downward migration for this change
-- 2. Resource strings will be computed for the acmgrantee table to facilitate common lookups
--    Downward migration will remove the added column and index
-- 3. Unique names, full path, and full unique path will be calculated for each object based their current
--    hierarchy and unique naming relative to their peers.
--    Downard migration will remove the added table for pathing

-- remove triggers on object for create/update
DROP TRIGGER IF EXISTS ti_object;
DROP TRIGGER IF EXISTS tu_object;

-- a temporary procedure that will convert existing objects using slashes to be represented in folder hierarchy
DROP PROCEDURE IF EXISTS sp_Patch_FolderHierarchy;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_FolderHierarchy(IN MASTERKEY varchar(255))
BEGIN
    DECLARE vID binary(16) default 0;
    DECLARE vParentID binary(16) default 0;
    DECLARE vCreatedBy varchar(255) default '';
    DECLARE vOwnedBy varchar(255) default '';
    DECLARE vName varchar(255) default '';
    DECLARE vRawAcm text;
    DECLARE vFolderTypeID binary(16) default 0;
    DECLARE vAcmId binary(16) default 0;

    -- validate inputs
    DECLARE validMasterKey int default 0;
    IF MASTERKEY IS NOT NULL THEN
        IF LENGTH(MASTERKEY) > 0 THEN
            IF LOCATE('MASTERKEY', MASTERKEY) < 1 THEN
                SET validMasterKey := 1;
            END IF;
        END IF;
    END IF;
    IF validMasterKey = 0 THEN
        SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'Must provide encryption master key when calling sp_Patch_FolderHierarchy';        
    END IF;

    OVERALL: BEGIN
        -- this main loop identifies all objects that have reserved characters used for deliniating hierarchy
        DECLARE posPart int default 0;
        DECLARE finished_objects int default 0;
        DECLARE cursor_objects cursor for SELECT id, parentid, createdBy, ownedby, name, rawacm FROM object WHERE name like '%/%' or name like '%\\\\\\%';
        DECLARE continue handler for not found set finished_objects = 1;
        SELECT id INTO vFolderTypeID FROM object_type WHERE name in ('Folder','File','') order by name desc limit 1;
        OPEN cursor_objects;
        get_object: LOOP
            FETCH cursor_objects INTO vID, vParentID, vCreatedBy, vOwnedBy, vName, vRawAcm;
            IF finished_objects = 1 THEN
                CLOSE cursor_objects;
                LEAVE get_object;
            END IF;
            -- get the acm identifier for this object
            SELECT acmId INTO vAcmId FROM objectacm WHERE objectId = vID;
            -- find next position of a delimiter
            SELECT least(ifnull(nullif(locate('/',vName),0),nullif(locate('\\',vName),0)),ifnull(nullif(locate('\\',vName),0),nullif(locate('/',vName),0))) INTO posPart;
            WHILE posPart > 0 DO
                OBJ_DELIM: BEGIN
                    DECLARE vPartName varchar(255) default '';
                    DECLARE vExistingFolderCount int default 0;
                    DECLARE vExistingFolderID binary(16) default 0;
                    DECLARE vContentConnector varchar(2000) default '';
                    -- get part name
                    SELECT substr(vName, 1, posPart) INTO vPartName;
                    -- remove part name from beginning of name
                    SET vName := substr(vName, posPart + 1);
                    -- process the part
                    IF LENGTH(vPartName) > 1 THEN
                        -- remove delimiter from the end of part name
                        SET vPartName := substr(vPartName, 1, LENGTH(vPartName) - 1);
                        -- generate a unique content connector value that can be used for this part if creating
                        SELECT pseudorand256(md5(rand())) INTO vContentConnector;
                        -- look for object having same name under parent.
                        IF vParentID IS NULL THEN
                            -- needs to be owned by the user
                            SELECT count(0) INTO vExistingFolderCount FROM object where parentid IS NULL and name = vPartName and ownedBy = vOwnedBy;
                            IF vExistingFolderCount > 0 THEN
                                SELECT id INTO vExistingFolderID FROM object WHERE parentID is NULL and name = vPartName AND ownedBy = vOwnedBy LIMIT 1;
                            ELSE
                                -- create if not found                                
                                INSERT INTO object (createdBy, ownedBy, parentID, name, typeid, rawacm, contentconnector) values (vCreatedBy, vOwnedBy, null, vPartName, vFolderTypeID, vRawAcm, vContentConnector);
                                SELECT id INTO vExistingFolderID FROM object WHERE parentID is NULL and name = vPartName AND ownedBy = vOwnedBy and contentConnector = vContentConnector LIMIT 1;
                            END IF;                            
                        ELSE
                            -- doesnt matter who the owner is
                            SELECT count(0) INTO vExistingFolderCount FROM object where parentid = vParentID and name = vPartName;
                            IF vExistingFolderCount > 0 THEN
                                SELECT id INTO vExistingFolderID FROM object WHERE parentID = vParentID and name = vPartName LIMIT 1;
                            ELSE
                                -- create if not found
                                INSERT INTO object (createdBy, ownedBy, parentID, name, typeid, rawacm, contentconnector) values (vCreatedBy, vOwnedBy, vParentID, vPartName, vFolderTypeID, vRawAcm, vContentConnector);
                                SELECT id INTO vExistingFolderID FROM object WHERE parentID = vParentID and name = vPartName AND ownedBy = vOwnedBy and contentConnector = vContentConnector LIMIT 1;
                            END IF;
                        END IF;
                        -- common things needing done if this part was created
                        IF vExistingFolderCount = 0 THEN
                            -- associate acm
                            INSERT INTO objectacm (createdBy, objectId, acmId) values (vCreatedBy, vExistingFolderID, vAcmId);
                            -- copy permissions
                            PERMISSIONS: BEGIN
                                DECLARE vGrantee varchar(255) default '';
                                DECLARE vAcmShare text default '';
                                DECLARE vAllowCreate tinyint(1) default 0;
                                DECLARE vAllowRead tinyint(1) default 0;
                                DECLARE vAllowUpdate tinyint(1) default 0;
                                DECLARE vAllowDelete tinyint(1) default 0;
                                DECLARE vAllowShare tinyint(1) default 0;
                                DECLARE vPermissionIV binary(32) default 0;
                                DECLARE vPermissionMAC binary(32) default 0;
                                DECLARE vEncryptKey binary(32) default 0;
                                DECLARE finished_permissions int default 0;
                                DECLARE cursor_permissions cursor for SELECT grantee, acmshare, allowcreate, allowread, allowupdate, allowdelete, allowshare FROM object_permission WHERE objectId = vID;
                                DECLARE continue handler for not found set finished_permissions = 1;
                                OPEN cursor_permissions;
                                get_permission: LOOP
                                    FETCH cursor_permissions INTO vGrantee, vAcmShare, vAllowCreate, vAllowRead, vAllowUpdate, vAllowDelete, vAllowShare;
                                    IF finished_permissions = 1 THEN
                                        CLOSE cursor_permissions;
                                        LEAVE get_permission;
                                    END IF;
                                    -- new iv
                                    SELECT unhex(pseudorand256(md5(rand()))) INTO vPermissionIV;
                                    -- new encryptkey
                                    SELECT unhex(new_keydecrypt(MASTERKEY, hex(vPermissionIV))) INTO vEncryptKey;
                                    -- new mac
                                    SELECT unhex(new_keymac(MASTERKEY, vGrantee, vAllowCreate, vAllowRead, vAllowUpdate, vAllowDelete, vAllowShare, hex(vEncryptKey))) INTO vPermissionMAC;
                                    -- insert it
                                    INSERT into object_permission (createdBy, objectId, grantee, acmShare, 
                                        allowCreate, allowRead, allowUpdate, allowDelete, allowShare, 
                                        explicitShare, encryptKey, permissionIV, permissionMAC) values (
                                            vCreatedBy, vExistingFolderID, vGrantee, vAcmShare,
                                            vAllowCreate, vAllowRead, vAllowUpdate, vAllowDelete, vAllowShare,
                                            0, vEncryptKey, vPermissionIV, vPermissionMAC
                                        );
                                END LOOP get_permission;
                            END PERMISSIONS;
                        END IF;
                        -- new parent
                        SET vParentID := vExistingFolderID;
                    END IF;
                END OBJ_DELIM;
                -- get updated position for next part
                SELECT least(ifnull(nullif(locate('/',vName),0),nullif(locate('\\',vName),0)),ifnull(nullif(locate('\\',vName),0),nullif(locate('/',vName),0))) INTO posPart;
            END WHILE;

            -- Whatever is left of vName is the new name
            UPDATE object SET name = vName, parentId = vParentID WHERE id = vID;

        END LOOP get_object;
    END OVERALL;
END
-- +migrate StatementEnd
CALL sp_Patch_FolderHierarchy('${OD_ENCRYPT_MASTERKEY}');
DROP PROCEDURE IF EXISTS sp_Patch_FolderHierarchy;

-- add field to acmgrantee table to store resourceString representation
ALTER TABLE acmgrantee 
    ADD COLUMN resourceString varchar(300) after grantee,
    ADD INDEX ix_resourceString (resourceString)
;

-- a temporary procedure that will populate the resourceString from the other fields in the acmgrantee table
DROP PROCEDURE IF EXISTS sp_Patch_PopulateResourceStrings;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_PopulateResourceStrings()
BEGIN
    DECLARE vGrantee varchar(255) default '';
    DECLARE vResourceString varchar(300) default '';
    DECLARE vProjectName varchar(255) default '';
    DECLARE vProjectDisplayName varchar(255) default '';
    DECLARE vGroupName varchar(255) default '';
    DECLARE vUserDistinguishedName varchar(255) default '';
    DECLARE finished_grantees int default 0;
    DECLARE cursor_grantees cursor for SELECT grantee, projectName, projectDisplayName, groupName, userDistinguishedName FROM acmgrantee;
    DECLARE continue handler for not found set finished_grantees = 1;
    OPEN cursor_grantees;
    get_grantee: LOOP
        FETCH cursor_grantees INTO vGrantee, vProjectName, vProjectDisplayName, vGroupName, vUserDistinguishedName;
        IF finished_grantees = 1 THEN
            CLOSE cursor_grantees;
            LEAVE get_grantee;
        END IF; 
        IF vUserDistinguishedName IS NULL THEN
            SET vResourceString := 'group';
            IF vProjectName IS NOT NULL THEN
                SET vResourceString := CONCAT(vResourceString, '/', vProjectName);
            END IF;
            IF vProjectDisplayName IS NOT NULL THEN
                SET vResourceString := CONCAT(vResourceString, '/', vProjectDisplayName);
            END IF;
            IF vGroupName IS NOT NULL THEN
                SET vResourceString := CONCAT(vResourceString, '/', vGroupName);
            END IF;
        ELSE
            SET vResourceString := CONCAT('user/', vUserDistinguishedName);
        END IF;
        UPDATE acmgrantee SET resourceString = vResourceString WHERE grantee = vGrantee AND resourceString IS NULL;
    END LOOP get_grantee;   
END
-- +migrate StatementEnd
CALL sp_Patch_PopulateResourceStrings();
DROP PROCEDURE IF EXISTS sp_Patch_PopulateResourceStrings;

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

-- a function to determine the unique name
DROP FUNCTION IF EXISTS calcUniqueName;
-- +migrate StatementBegin
CREATE FUNCTION calcUniqueName(vID binary(16), vName varchar(255), vParentID binary(16), vOwnedBy varchar(255)) RETURNS varchar(300)
BEGIN
    DECLARE vNamePrefix varchar(255) default '';
    DECLARE vNameSuffix varchar(255) default '';
    DECLARE vUniqueSuffixCounter int default 1;
    DECLARE vUniqueName varchar(300) default '';
    DECLARE vSiblingSameName int default 1;

    -- Determine parts of name splitting on file extension, if applicable
    SET vNameSuffix := SUBSTRING_INDEX(vName, '.', -1);
    SET vNamePrefix := LEFT(vName, LENGTH(vName) - LENGTH(vNameSuffix));
    IF LENGTH(vNamePrefix) > 0 THEN
        -- There is an extension. Move the period from the tail of the prefix to the head of the suffix
        SET vNamePrefix := LEFT(vNamePrefix, LENGTH(vNamePrefix) - 1);
        SET vNameSuffix := CONCAT('.', vNameSuffix);
    ELSE
        SET vNamePrefix := vNameSuffix;
        SET vNameSuffix := '';
    END IF;

    -- This counter initialized to 1 so if its used it will result in increments as 'filename (2).ext'
    SET vUniqueSuffixCounter := 1;
    -- Anticipated unique name, first start with same name
    SET vUniqueName := CONCAT(vNamePrefix, vNameSuffix);

    -- Always enter the subsequent loop to check siblings
    SET vSiblingSameName := 1;
    WHILE vSiblingSameName > 0 DO
        IF vParentID IS NULL THEN
            -- Look at siblings, to determine if any have the unique name already
            -- when at root, also discriminate on owner
            SELECT count(0) INTO vSiblingSameName
            FROM object_pathing sop INNER JOIN object so ON sop.id = so.id 
            WHERE so.id <> vID and sop.UniqueName = vUniqueName and so.parentId IS NULL and so.ownedBy = vOwnedBy;
        ELSE
            -- Look at siblings, to determine if any have the unique name already
            SELECT count(0) INTO vSiblingSameName
            FROM object_pathing sop INNER JOIN object so ON sop.id = so.id 
            WHERE so.id <> vID and sop.UniqueName = vUniqueName and so.parentId = vParentID;
        END IF;
        IF vSiblingSameName > 0 THEN
            -- Calculate a new unique name by incrementing counter
            SET vUniqueSuffixCounter := vUniqueSuffixCounter + 1;
            SET vUniqueName := CONCAT(vNamePrefix, '(', vUniqueSuffixCounter, ')', vNameSuffix);
        END IF;
    END WHILE;

	RETURN vUniqueName;
END;
-- +migrate StatementEnd

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

    -- vGoAgain is a flag that tracks if there are still parents that havent been mapped that a child depends on for pathing
    DECLARE vGoAgain int default 1;
    WHILE vGoAgain > 0 DO
        OVERALL: BEGIN
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
                SET vUniquePathName := CONCAT('/', vUniqueName);

                -- Initialize this objects path name. All paths start with a forward slash
                SET vPathName := CONCAT('/', vName);

                -- Get pathing info from parents
                IF vParentID IS NULL THEN
                    SELECT grantee INTO vRootOwnedByGrantee FROM acmgrantee WHERE resourceString = vOwnedBy or LOCATE(concat(resourceString,'/'),vOwnedBy) > 0;
                    SET vUniquePathName := CONCAT('/', vRootOwnedByGrantee, vUniquePathName);
                    SET vPathName := CONCAT('/', vRootOwnedByGrantee, vPathName);
                    INSERT INTO object_pathing (id, uniqueName, uniquePathName, pathName) VALUES (vID, vUniqueName, vUniquePathName, vPathName);
                ELSE
                    SELECT count(0) INTO vParentPathCalculated FROM object_pathing WHERE id = vParentID;
                    IF vParentPathCalculated > 0 THEN
                        SELECT uniquePathName, pathName INTO vParentUniquePathName, vParentPathName FROM object_pathing WHERE id = vParentID;
                        SET vPathName := CONCAT(vParentPathName, vPathName);
                        SET vUniquePathName := CONCAT(vParentUniquePathName, vUniquePathName);
                        INSERT INTO object_pathing (id, uniqueName, uniquePathName, pathName) VALUES (vID, vUniqueName, vUniquePathName, vPathName);
                    ELSE
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

-- recreate triggers on object for create/update with support for determining object_pathing
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

        -- Get the unique name and initialize path as rooted
        SET vUniqueName := calcUniqueName(NEW.id, NEW.name, NEW.parentId, NEW.ownedBy);
        SET vUniquePathName := CONCAT('/', vUniqueName);
        SET vPathName := CONCAT('/', NEW.name);

        -- Get pathing info from parents
        IF NEW.parentId IS NULL THEN
            SELECT grantee INTO vRootOwnedByGrantee FROM acmgrantee WHERE resourceString = NEW.ownedBy or LOCATE(concat(resourceString,'/'),NEW.ownedBy) > 0;
            SET vUniquePathName := CONCAT('/', vRootOwnedByGrantee, vUniquePathName);
            SET vPathName := CONCAT('/', vRootOwnedByGrantee, vPathName);
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

            -- Get the unique name and initialize path as rooted
            SET vUniqueName := calcUniqueName(NEW.id, NEW.name, NEW.parentID, NEW.ownedBy);
            SET vUniquePathName := CONCAT('/', vUniqueName);
            SET vPathName := CONCAT('/', NEW.name);

            -- Get new pathing info from parents
            IF NEW.parentId IS NULL THEN
                SELECT grantee INTO vRootOwnedByGrantee FROM acmgrantee WHERE resourceString = NEW.ownedBy or LOCATE(concat(resourceString,'/'),NEW.ownedBy) > 0;
                SET vUniquePathName := CONCAT('/', vRootOwnedByGrantee, vUniquePathName);
                SET vPathName := CONCAT('/', vRootOwnedByGrantee, vPathName);
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
                uniquePathName LIKE CONCAT(vOldUniquePathName,'/%')
                AND
                pathName LIKE CONCAT(vOldPathName, '/%');
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

-- +migrate Down
DROP INDEX ix_resourceString ON acmgrantee;
ALTER TABLE acmgrantee DROP COLUMN resourceString;
DROP TABLE IF EXISTS object_pathing;

-- remove triggers on object for create/update
DROP TRIGGER IF EXISTS ti_object;
DROP TRIGGER IF EXISTS tu_object;

-- remove functions created
DROP FUNCTION IF EXISTS calcUniqueName;

-- recreate triggers for create/update
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
	SET NEW.schemaversion := '20161223'; 
	# Identifier is randomized as a GUID
	SET NEW.identifier := concat(@@hostname, '-', left(uuid(),8));
END;
-- +migrate StatementEnd
update dbstate set schemaVersion = '20161223';