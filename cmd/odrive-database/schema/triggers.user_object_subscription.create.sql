delimiter //
SELECT 'Creating trigger ti_user_object_subscription' as Action
//
CREATE TRIGGER ti_user_object_subscription
BEFORE INSERT ON user_object_subscription FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'user_object_subscription';

	# Rules
	# objectId must be specified
	IF NEW.objectId IS NULL OR NEW.objectId = '' THEN
		set error_msg := concat(error_msg, 'Field objectId required ');
	END IF;
	IF error_msg <> '' THEN
		SET error_msg := concat(error_msg, 'when inserting record into ',thisTableName);
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
		a_user_object_subscription
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
		,onCreate
		,onUpdate
		,onDelete
		,recursive
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
		,NEW.onCreate
		,NEW.onUpdate
		,NEW.onDelete
		,NEW.recursive
	);

	# Specific field level changes
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'objectId', newValue = hex(NEW.objectId);
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'onCreate', newValue = NEW.onCreate;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'onUpdate', newValue = NEW.onUpdate;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'onDelete', newValue = NEW.onDelete;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'recursive', newValue = NEW.recursive;

END
//
SELECT 'Creating trigger tu_user_object_subscription' as Action
//
CREATE TRIGGER tu_user_object_subscription
BEFORE UPDATE ON user_object_subscription FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'user_object_subscription';

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
	# modifiedBy required
	IF (NEW.modifiedBy IS NULL OR NEW.modifiedBy = '') AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Field modifiedBy is required ');
	END IF;
	# objectId cannot be changed
	IF (NEW.objectId <> OLD.objectId) AND length(error_msg) < 77 THEN
		SET error_msg := concat(error_msg, 'Unable to set objectId ');
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
    
	# Archive table
	INSERT INTO
		a_user_object_subscription
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
		,onCreate
		,onUpdate
		,onDelete
		,recursive
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
		,NEW.onCreate
		,NEW.onUpdate
		,NEW.onDelete
		,NEW.recursive
	);

	# Specific field level changes
	IF NEW.objectId <> OLD.objectId THEN
		# not possible
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'objectId', newValue = hex(NEW.objectId);
	END IF;
	IF NEW.onCreate <> OLD.onCreate THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'onCreate', newValue = NEW.onCreate;
	END IF;
	IF NEW.onUpdate <> OLD.onUpdate THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'onUpdate', newValue = NEW.onUpdate;
	END IF;
	IF NEW.onDelete <> OLD.onDelete THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'onDelete', newValue = NEW.onDelete;
	END IF;
	IF NEW.recursive <> OLD.recursive THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'recursive', newValue = NEW.recursive;
	END IF;

END
//
SELECT 'Creating trigger td_user_object_subscription' as Action
//
CREATE TRIGGER td_user_object_subscription
BEFORE DELETE ON user_object_subscription FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed. Use isDeleted, deletedDate and deletedBy';
	signal sqlstate '45000' set message_text = error_msg;
END
//
SELECT 'Creating trigger td_a_user_object_subscription' as Action
//
CREATE TRIGGER td_a_user_object_subscription
BEFORE DELETE ON a_user_object_subscription FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed on archive tables.';
	signal sqlstate '45000' set message_text = error_msg;
END
//
delimiter ;
