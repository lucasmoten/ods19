delimiter //
SELECT 'Creating trigger ti_object' as Action
//
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
	# ParentId must be valid if specified
	IF NEW.parentId IS NOT NULL AND NEW.parentId = '' THEN
		SET NEW.parentId = NULL;
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
		SET NEW.contentConnector = type_contentConnector;
	END IF;
	# Assign PDF availability if not set
	IF NEW.isPDFAvailable IS NULL THEN
		SET NEW.isPDFAvailable = 0;
	END IF;
	# Assign US Persons Data if not set
	IF NEW.isUSPersonsData IS NULL THEN
		SET NEW.isUSPersonsData = 0;
	END IF;
	# Assign FOIA Exempt status if not set
	IF NEW.isFOIAExempt IS NULL THEN
		SET NEW.isFOIAExempt = 0;
	END IF;
	# Assign stream storage state if not set
	IF NEW.isStreamStored IS NULL THEN
		SET NEW.isStreamStored = 0;
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
		,isUSPersonsData
		,isFOIAExempt
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
		,NEW.isUSPersonsData
		,NEW.isFOIAExempt
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
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isUSPersonsData', newValue = NEW.isUSPersonsData;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isFOIAExempt', newValue = NEW.isFOIAExempt;

END
//
SELECT 'Creating trigger tu_object' as Action
//
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
	# ParentId must be valid if specified
	IF NEW.parentId IS NOT NULL AND NEW.parentId <> '' THEN
		SELECT COUNT(*) FROM object WHERE ((isDeleted = 0) or (NEW.IsDeleted = 1)) AND id = NEW.parentId INTO count_parent;
		IF (count_parent = 0) and length(error_msg) < 71 THEN
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
	IF NEW.isUSPersonsData IS NULL THEN
		SET NEW.isUSPersonsData = OLD.isUSPersonsData;
	END IF;
	# Assign FOIA Exempt status if not set
	IF NEW.isFOIAExempt IS NULL THEN
		SET NEW.isFOIAExempt = OLD.isFOIAExempt;
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
		,isUSPersonsData
		,isFOIAExempt
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
		,NEW.isUSPersonsData
		,NEW.isFOIAExempt
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
	IF NEW.isUSPersonsData <> OLD.isUSPersonsData THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isUSPersonsData', newValue = NEW.isUSPersonsData;
	END IF;
	IF NEW.isFOIAExempt <> OLD.isFOIAExempt THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isFOIAExempt', newValue = NEW.isFOIAExempt;
	END IF;

END
//
SELECT 'Creating trigger td_object' as Action
//
CREATE TRIGGER td_object
BEFORE DELETE ON object FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed. Use isDeleted, deletedDate, and deletedBy';
	signal sqlstate '45000' set message_text = error_msg;
END
//
SELECT 'Creating trigger td_a_object' as Action
//
CREATE TRIGGER td_a_object
BEFORE DELETE ON a_object FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed on archive tables.';
	signal sqlstate '45000' set message_text = error_msg;
END
//
delimiter ;
