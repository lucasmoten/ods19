CREATE TRIGGER ti_user_object_favorite
BEFORE INSERT ON user_object_favorite FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'user_object_favorite';
	DECLARE count_favorite int default 0;

	# Rules
	# objectId must be specified
	IF NEW.objectId IS NULL OR NEW.objectId = '' THEN
		set error_msg := concat(error_msg, 'Field objectId required ');
	END IF;
	# objectId must be unique for the creator
	SELECT COUNT(*) FROM user_object_favorite WHERE createdBy = NEW.createdBy AND objectId = NEW.objectId AND isDeleted = 0 INTO count_favorite;
	IF count_favorite > 0 THEN
		SET error_msg := concat(error_msg, 'Field objectId must be unique ');
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

	# Archive table
	INSERT INTO
		a_user_object_favorite
	(
		id
		,createdDate
		,createdBy
		,modifiedDate
		,modifiedBy
		,isDeleted
		,deletedDate
		,deletedBy
		,objectId
	) values (
		NEW.id
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.isDeleted
		,NEW.deletedDate
		,NEW.deletedBy
		,NEW.objectId
	);

	# Specific field level changes
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'objectId', newValue = hex(NEW.objectId);
END
