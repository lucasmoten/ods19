CREATE TRIGGER ti_relationship
BEFORE INSERT ON relationship FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'relationship';
	DECLARE count_object int default 0;

	# Rules
	# modifiedBy should be set to createdBy when createdBy
	SET NEW.modifiedBy = NEW.createdBy;
	# sourceId must be specified and exist
	SELECT COUNT(*) FROM object WHERE isDeleted = 0 AND id = NEW.sourceId INTO count_object;
	IF count_type = 0 THEN
		SET error_msg := concat(error_msg, 'Field sourceId required ');
	END IF;
	# targetId must be specified and exist
	SELECT COUNT(*) FROM object WHERE isDeleted = 0 AND id = NEW.targetId INTO count_object;
	IF count_type = 0 THEN
		SET error_msg := concat(error_msg, 'Field targetId required ');
	END IF;
	IF error_msg <> '' THEN
		SET error_msg := concat(error_msg, 'when inserting record into ', thisTableName);
		signal sqlstate '45000' set message_text = error_msg;
	END IF;

	# Force values on create
	SET NEW.id := ordered_uuid(UUID());
	SET NEW.createdDate := current_timestamp();
	SET NEW.modifiedDate := current_timestamp();
	SET NEW.isDeleted := 0;
	SET NEW.deletedDate := NULL;
	SET NEW.deletedBy := NULL;
	SET NEW.changeCount := 0;
	# Standard change token formula
	SET NEW.changeToken := md5(CONCAT(CAST(NEW.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));

	# Archive table
	INSERT INTO
		a_relationship
	(
		id
		,createdDate
		,createdBy
		,modifiedDate
		,modifiedBy
		,isDeleted
		,deletedDate
		,deletedBy
		,changeCount
		,changeToken
		,sourceId
		,targetId
		,description
		,classificationPM
	) values (
		NEW.id
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.isDeleted
		,NEW.deletedDate
		,NEW.deletedBy
		,NEW.changeCount
		,NEW.changeToken
		,NEW.sourceId
		,NEW.targetId
		,NEW.description
		,NEW.classificationPM
	);

	# Specific field level changes
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'sourceId', newValue = hex(NEW.sourceId);
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'targetId', newValue = hex(NEW.targetId);
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'description', newValue = NEW.description;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'classificationPM', newValue = NEW.classificationPM;
END;
