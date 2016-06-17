delimiter //
SELECT 'Creating trigger ti_object_type_property' as Action
//
CREATE TRIGGER ti_object_type_property
BEFORE INSERT ON object_type_property FOR EACH ROW
BEGIN
	DECLARE thisTableName varchar(128) default 'object_type_property';

	# Force values on create
	SET NEW.id := ordered_uuid(UUID());
	SET NEW.createdDate := curent_timestamp();
	SET NEW.modifiedDate := current_timestamp();
	SET NEW.modifiedBy := NEW.createdBy;
	SET NEW.isDeleted := 0;
	SET NEW.deletedDate := NULL;
	SET NEW.deletedBy := NULL;

	# No archive table

	# Specific field level changes
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'typeId', newValue = hex(typeId);
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'propertyId', newValue = hex(propertyId);

END
//
SELECT 'Creating trigger tu_object_type_property' as Action
//
CREATE TRIGGER tu_object_type_property
BEFORE UPDATE ON object_type_property FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'object_type_property';

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
	# typeId cannot be changed
	IF (NEW.typeId <> OLD.typeId) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set typeId ');
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
SELECT 'Creating trigger td_object_type_property' as Action
//
CREATE TRIGGER td_object_type_property
BEFORE DELETE ON object_type_property FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed. Use isDeleted, deletedDate and deletedBy';
	signal sqlstate '45000' set message_text = error_msg;
END
//
delimiter ;
