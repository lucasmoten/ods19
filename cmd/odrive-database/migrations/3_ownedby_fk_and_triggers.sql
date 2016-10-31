-- +migrate Up

-- Treat ownedBy field as serialized resource string
-- Drop associated foreign keys, indexes, and triggers, migrate the 
-- data, and apply revised logic in the triggers 

-- Drop foreign keys and indexes referencing user distinguishedName from ownedBy columns
ALTER TABLE object DROP FOREIGN KEY fk_object_ownedBy;
ALTER TABLE object DROP FOREIGN KEY fk_object_ownedByNew;
DROP INDEX ix_ownedBy ON object;
DROP INDEX ix_ownedBy ON a_object;
ALTER TABLE object_type DROP FOREIGN KEY fk_object_type_ownedBy;
-- Drop triggers to avoid collisions on updates
DROP TRIGGER IF EXISTS ti_object;
DROP TRIGGER IF EXISTS ti_object_type;
DROP TRIGGER IF EXISTS tu_object;
DROP TRIGGER IF EXISTS tu_object_type;
-- Update current and archive records of data, prefixing user distinguished name with the resource type
UPDATE a_object SET ownedBy = concat('user/', createdBy);
UPDATE a_object_type SET ownedBy = concat('user/', createdBy);
UPDATE object SET ownedBy = concat('user/', createdBy);
UPDATE object_type SET ownedBy = concat('user/', createdBy);
-- Recreate the triggers with revised logic for the ownedBy field.
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
    SET NEW.createdDate := current_timestamp();
    SET NEW.modifiedDate := current_timestamp();
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
CREATE TRIGGER ti_object_type
BEFORE INSERT ON object_type FOR EACH ROW
BEGIN
    DECLARE error_msg varchar(128) default '';
    DECLARE thisTableName varchar(128) default 'object_type';
    DECLARE count_name int default 0;

    # Rules
    # Name must be unique for non-deletedBy
    SELECT COUNT(*) FROM object_type WHERE isDeleted = 0 AND name = NEW.name INTO count_name;
    IF count_name > 0 THEN
        SET error_msg := concat(error_msg, 'Field name must be unique ');
    END IF;
    IF error_msg <> '' THEN
        SET error_msg := concat(error_msg, 'when inserting record into ', thisTableName);
        signal sqlstate '45000' set message_text = error_msg;
    END IF;

    # Force values on create
    SET NEW.id := ordered_uuid(UUID());
    SET NEW.createdDate := current_timestamp();
    SET NEW.modifiedDate := current_timestamp();
    SET NEW.modifiedBy := NEW.createdBy;
    SET NEW.isDeleted := 0;
    SET NEW.deletedDate := NULL;
    SET NEW.deletedBy := NULL;
    # Assign Owner if not set
    IF NEW.ownedBy IS NULL OR NEW.ownedBy = '' THEN
        SET NEW.ownedBy := concat('user/', NEW.createdBy);
    END IF;
    SET NEW.changeCount := 0;
    # Standard change token formula
    SET NEW.changeToken := md5(CONCAT(CAST(NEW.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));

    # Archive table
    INSERT INTO
        a_object_type
    (
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,ownedBy
        ,changeCount
        ,changeToken
        ,name
        ,description
        ,contentConnector
    ) values (
        NEW.id
        ,NEW.createdDate
        ,NEW.createdBy
        ,NEW.modifiedDate
        ,NEW.modifiedBy
        ,NEW.isDeleted
        ,NEW.deletedDate
        ,NEW.deletedBy
        ,NEW.ownedBy
        ,NEW.changeCount
        ,NEW.changeToken
        ,NEW.name
        ,NEW.description
        ,NEW.contentConnector
    );

    # Specific field level changes
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'name', newValue = NEW.name;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'description', newValue = NEW.description;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentConnector', newValue = NEW.contentConnector;

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
    SET NEW.modifiedDate := current_timestamp();
    IF (NEW.isDeleted <> OLD.isDeleted) THEN
        IF  (NEW.IsDeleted = 1) THEN
            SET NEW.deletedDate := current_timestamp();
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
-- +migrate StatementBegin
CREATE TRIGGER tu_object_type
BEFORE UPDATE ON object_type FOR EACH ROW
BEGIN
    DECLARE error_msg varchar(128) default '';
    DECLARE thisTableName varchar(128) default 'object_type';
    DECLARE count_name int default 0;

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
    # Name must be unique for non-deletedBy
    SELECT COUNT(*) FROM object_type WHERE isDeleted = 0 AND name = NEW.name AND id <> OLD.id INTO count_name;
    IF (count_name > 0) AND length(error_msg) < 74 THEN
        SET error_msg := concat(error_msg, 'Field name must be unique ');
    END IF;
    # changeToken must be given and match the record
    IF (NEW.changeToken IS NULL OR NEW.changeToken = '') and length(error_msg) < 73 THEN
        SET error_msg := concat(error_msg, 'Field changeToken required ');
    END IF;
    IF (NEW.changeToken <> OLD.changeToken) and length(error_msg) < 71 THEN
        SET error_msg := concat(error_msg, 'Field changeToken must match ');
    END IF;
    IF length(error_msg) > 0 THEN
        SET error_msg := concat(error_msg, 'when updating record');
        signal sqlstate '45000' set message_text = error_msg;
    END IF;

    # Force values on modify
    SET NEW.modifiedDate := current_timestamp();
    IF (NEW.isDeleted <> OLD.isDeleted) THEN
        IF  (NEW.IsDeleted = 1) THEN
            SET NEW.deletedDate := current_timestamp();
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
    END IF;

    # Archive table
    INSERT INTO
        a_object_type
    (
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,ownedBy
        ,changeCount
        ,changeToken
        ,name
        ,description
        ,contentConnector
    ) values (
        NEW.id
        ,NEW.createdDate
        ,NEW.createdBy
        ,NEW.modifiedDate
        ,NEW.modifiedBy
        ,NEW.isDeleted
        ,NEW.deletedDate
        ,NEW.deletedBy
        ,NEW.ownedBy
        ,NEW.changeCount
        ,NEW.changeToken
        ,NEW.name
        ,NEW.description
        ,NEW.contentConnector
    );

    # Specific field level changes
    IF NEW.name <> OLD.name THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'name', newValue = NEW.name;
    END IF;
    IF NEW.description <> OLD.description THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'description', newValue = NEW.description;
    END IF;
    IF NEW.contentConnector <> OLD.contentConnector THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentConnector', newValue = NEW.contentConnector;
    END IF;

END;
-- +migrate StatementEnd

-- +migrate Down

-- Rollback ownedBy field to consist of the creators distingusihed name
-- Drop the associated triggers, revert the data, recreate the foreign 
-- key constraints and indexes and finally rebuild the triggers that 
-- set the ownedby field equal to createdby if not set

-- Drop triggers to avoid collisions on updates
DROP TRIGGER IF EXISTS ti_object;
DROP TRIGGER IF EXISTS ti_object_type;
DROP TRIGGER IF EXISTS tu_object;
DROP TRIGGER IF EXISTS tu_object_type;

-- Rollback the data, assigning the ownedBy to equal the createdBy
UPDATE a_object SET ownedBy = createdBy, ownedByNew = null;
UPDATE a_object_type SET ownedBy = createdBy;
UPDATE object SET ownedBy = createdBy, ownedByNew = null;
UPDATE object_type SET ownedBy = createdBy;

-- Recreate the foreign key constraints and indexes
ALTER TABLE object_type	
      ADD CONSTRAINT fk_object_type_ownedBy FOREIGN KEY (ownedBy) REFERENCES user(distinguishedName);
ALTER TABLE object 
      ADD CONSTRAINT fk_object_ownedBy FOREIGN KEY (ownedBy) REFERENCES user(distinguishedName)
    , ADD INDEX ix_ownedBy (ownedBy)
    , ADD CONSTRAINT fk_object_ownedByNew FOREIGN KEY (ownedByNew) REFERENCES user(distinguishedName);
ALTER TABLE a_object
      ADD INDEX ix_ownedBy (ownedBy);

-- Recreate the original triggers where the ownedBy field is assigned
-- the value of the createdBy field if not set.
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
	SET NEW.createdDate := current_timestamp();
	SET NEW.modifiedDate := current_timestamp();
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
		SET NEW.ownedBy := NEW.createdBy;
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
CREATE TRIGGER ti_object_type
BEFORE INSERT ON object_type FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'object_type';
	DECLARE count_name int default 0;

	# Rules
	# Name must be unique for non-deletedBy
	SELECT COUNT(*) FROM object_type WHERE isDeleted = 0 AND name = NEW.name INTO count_name;
	IF count_name > 0 THEN
		SET error_msg := concat(error_msg, 'Field name must be unique ');
	END IF;
	IF error_msg <> '' THEN
		SET error_msg := concat(error_msg, 'when inserting record into ', thisTableName);
		signal sqlstate '45000' set message_text = error_msg;
	END IF;

	# Force values on create
	SET NEW.id := ordered_uuid(UUID());
	SET NEW.createdDate := current_timestamp();
	SET NEW.modifiedDate := current_timestamp();
	SET NEW.modifiedBy := NEW.createdBy;
	SET NEW.isDeleted := 0;
	SET NEW.deletedDate := NULL;
	SET NEW.deletedBy := NULL;
	# Assign Owner if not set
	IF NEW.ownedBy IS NULL OR NEW.ownedBy = '' THEN
		SET NEW.ownedBy := NEW.createdBy;
	END IF;
	SET NEW.changeCount := 0;
	# Standard change token formula
	SET NEW.changeToken := md5(CONCAT(CAST(NEW.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));

	# Archive table
	INSERT INTO
		a_object_type
	(
		id
		,createdDate
		,createdBy
		,modifiedDate
		,modifiedBy
		,isDeleted
		,deletedDate
		,deletedBy
		,ownedBy
		,changeCount
		,changeToken
		,name
		,description
		,contentConnector
	) values (
		NEW.id
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.isDeleted
		,NEW.deletedDate
		,NEW.deletedBy
		,NEW.ownedBy
		,NEW.changeCount
		,NEW.changeToken
		,NEW.name
		,NEW.description
		,NEW.contentConnector
	);

	# Specific field level changes
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'name', newValue = NEW.name;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'description', newValue = NEW.description;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentConnector', newValue = NEW.contentConnector;

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
	SET NEW.modifiedDate := current_timestamp();
    IF (NEW.isDeleted <> OLD.isDeleted) THEN
        IF  (NEW.IsDeleted = 1) THEN
            SET NEW.deletedDate := current_timestamp();
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
		SET NEW.ownedBy := NEW.createdBy;
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
-- +migrate StatementBegin
CREATE TRIGGER tu_object_type
BEFORE UPDATE ON object_type FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'object_type';
	DECLARE count_name int default 0;

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
	# Name must be unique for non-deletedBy
	SELECT COUNT(*) FROM object_type WHERE isDeleted = 0 AND name = NEW.name AND id <> OLD.id INTO count_name;
	IF (count_name > 0) AND length(error_msg) < 74 THEN
		SET error_msg := concat(error_msg, 'Field name must be unique ');
	END IF;
	# changeToken must be given and match the record
	IF (NEW.changeToken IS NULL OR NEW.changeToken = '') and length(error_msg) < 73 THEN
		SET error_msg := concat(error_msg, 'Field changeToken required ');
	END IF;
	IF (NEW.changeToken <> OLD.changeToken) and length(error_msg) < 71 THEN
		SET error_msg := concat(error_msg, 'Field changeToken must match ');
	END IF;
	IF length(error_msg) > 0 THEN
		SET error_msg := concat(error_msg, 'when updating record');
		signal sqlstate '45000' set message_text = error_msg;
	END IF;

	# Force values on modify
	SET NEW.modifiedDate := current_timestamp();
    IF (NEW.isDeleted <> OLD.isDeleted) THEN
        IF  (NEW.IsDeleted = 1) THEN
            SET NEW.deletedDate := current_timestamp();
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
		SET NEW.ownedBy := NEW.createdBy;
	END IF;

	# Archive table
	INSERT INTO
		a_object_type
	(
		id
		,createdDate
		,createdBy
		,modifiedDate
		,modifiedBy
		,isDeleted
		,deletedDate
		,deletedBy
		,ownedBy
		,changeCount
		,changeToken
		,name
		,description
		,contentConnector
	) values (
		NEW.id
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.isDeleted
		,NEW.deletedDate
		,NEW.deletedBy
		,NEW.ownedBy
		,NEW.changeCount
		,NEW.changeToken
		,NEW.name
		,NEW.description
		,NEW.contentConnector
	);

	# Specific field level changes
	IF NEW.name <> OLD.name THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'name', newValue = NEW.name;
	END IF;
	IF NEW.description <> OLD.description THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'description', newValue = NEW.description;
	END IF;
	IF NEW.contentConnector <> OLD.contentConnector THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentConnector', newValue = NEW.contentConnector;
	END IF;

END;
-- +migrate StatementEnd

