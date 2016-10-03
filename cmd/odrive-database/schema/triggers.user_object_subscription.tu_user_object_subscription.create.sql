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
