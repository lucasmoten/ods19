delimiter //
SELECT 'Create trigger ti_object_property' as Action
//
CREATE TRIGGER ti_object_property
BEFORE INSERT ON object_property FOR EACH ROW
BEGIN
	DECLARE thisTableName varchar(128) default 'object_property';

	# Force values on create
	SET NEW.id := ordered_uuid(UUID());
	SET NEW.createdDate := current_timestamp();
	SET NEW.modifiedDate := current_timestamp();
	SET NEW.modifiedBy := NEW.createdBy;
	SET NEW.isDeleted := 0;
	SET NEW.deletedDate := NULL;
	SET NEW.deletedBy := NULL;

	# No archive table for this many-to-many relationship table. 
	
	# Specific field level changes
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'objectId', newValue = hex(NEW.objectId);
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'propertyId', newValue = hex(NEW.propertyId);
	
END
//
SELECT 'Creating trigger tu_object_property' as Action
//
CREATE TRIGGER tu_object_property
BEFORE UPDATE ON object_property FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'object_property';

	# Rules
	# only deletes are allowed.
	# id cannot bechanged
	IF (NEW.id <> OLD.id) AND length(error_msg) < 83 THEN
		SET error_msg := concat(error_msg, 'Unable to set id ');
	END IF;
	# createdDate cannot be changed
	IF (NEW.createdDate <> OLD.createdDate) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set createdDate ');
	END IF;
	# createdBy cannot be changed
	IF (NEW.createdBy <> OLD.createdBy) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set createdBy ');
	END IF;
	# objectId cannot be changed
	IF (NEW.objectId <> OLD.objectId) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set objectId ');
	END IF;
	# propertyId cannot be changed
	IF (NEW.propertyId <> OLD.propertyId) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set propertyId ');
	END IF;
	IF length(error_msg) > 0 THEN
		SET error_msg := concat(error_msg, 'when updating record');
		signal sqlstate '45000' set message_text = error_msg;
	END IF;
	
	# Force values on modify
    # The only modification allowed is to mark as deleted...
	SET NEW.modifiedDate := current_timestamp();
   	IF NEW.modifiedBy IS NULL OR NEW.modifiedBy = '' THEN
		SET NEW.modifiedBy := NEW.deletedBy;
	END IF;
	SET NEW.isDeleted = 1;
	SET NEW.deletedDate := current_timestamp();
	IF NEW.deletedBy IS NULL OR NEW.deletedBy = '' THEN
		SET NEW.deletedBy := NEW.modifiedBy;
	END IF;    

    	
END
//
SELECT 'Creating trigger td_object_property' as Action
//
CREATE TRIGGER td_object_property
BEFORE DELETE ON object_property FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed. Use isDeleted, deletedDate, and deletedBy';
	signal sqlstate '45000' set message_text = error_msg;
END
//
delimiter ;
