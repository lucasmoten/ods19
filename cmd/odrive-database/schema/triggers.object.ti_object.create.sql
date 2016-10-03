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
