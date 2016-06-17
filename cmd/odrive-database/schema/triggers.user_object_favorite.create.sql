delimiter //
SELECT 'Creating trigger ti_user_object_favorite' as Action
//
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
//
SELECT 'Creating trigger tu_user_object_favorite' as Action
//
CREATE TRIGGER tu_user_object_favorite
BEFORE UPDATE ON user_object_favorite FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'user_object_favorite';

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
	# objectId cannot be changed
	IF (NEW.objectId <> OLD.objectId) AND length(error_msg) < 77 THEN
		SET error_msg := concat(error_msg, 'Unable to set objectId ');
	END IF;
	# deletedBy must be set
	IF (NEW.deletedBy IS NULL OR NEW.deletedBy = '') AND (NEW.modifiedBy IS NULL OR NEW.modifiedBy = '') AND length(error_msg) < 75 THEN
		SET error_msg := concat(error_msg, 'Field deletedBy or modifiedBy required ');
	END IF;
	IF length(error_msg) > 0 THEN
		SET error_msg := concat(error_msg, 'when updating record');
		signal sqlstate '45000' set message_text = error_msg;
	END IF;

	# Force values on modify
	SET NEW.modifiedDate := current_timestamp();
   	IF NEW.modifiedBy IS NULL OR NEW.modifiedBy = '' THEN
		SET NEW.modifiedBy := NEW.deletedBy;
	END IF;
	SET NEW.isDeleted = 1;
	SET NEW.deletedDate := current_timestamp();
	IF NEW.deletedBy IS NULL OR NEW.deletedBy = '' THEN
		SET NEW.deletedBy := NEW.modifiedBy;
	END IF;

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

END
//
SELECT 'Creating trigger td_user_object_favorite' as Action
//
CREATE TRIGGER td_user_object_favorite
BEFORE DELETE ON user_object_favorite FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed. User isDeleted, deletedDate, and deletedBy';
	signal sqlstate '45000' set message_text = error_msg;
END
//
SELECT 'Creating trigger td_a_user_object_favorite' as Action
//
CREATE TRIGGER td_a_user_object_favorite
BEFORE DELETE ON a_user_object_favorite FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed on archive tables.';
	signal sqlstate '45000' set message_text = error_msg;
END
//
delimiter ;
