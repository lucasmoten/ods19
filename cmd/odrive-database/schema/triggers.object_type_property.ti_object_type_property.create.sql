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

END;
