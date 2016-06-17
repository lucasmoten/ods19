delimiter //
SELECT 'Creating trigger ti_acmpart' as Action
//
CREATE TRIGGER ti_acmpart
BEFORE INSERT ON acmpart FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'acmpart';
	
	#Rules
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
		a_acmpart
	(
		id
		,createdDate
		,createdBy
		,modifiedDate
		,modifiedBy
		,isDeleted
		,deletedDate
		,deletedBy
		,acmId
		,acmKeyId
		,acmValueId
	) VALUES (
		NEW.id
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.isDeleted
		,NEW.deletedDate
		,NEW.deletedBy
		,NEW.acmId
		,NEW.acmKeyId
		,NEW.acmValueId
	);

	# Specific field level changes
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmId', newValue = hex(NEW.acmId);
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmKeyId', newValue = hex(NEW.acmKeyId);
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmValueId', newValue = hex(NEW.acmValueId);

END
//
SELECT 'Creating trigger tu_acmpart' as Action
//
CREATE TRIGGER tu_acmpart
BEFORE UPDATE ON acmpart FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'acmpart';	

	# Rules
	# This is a many-to-many table, so effectively, nothing is allowed to change
	# id cannot be changed
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
	# acmId cannot be changed
	IF (NEW.acmId <> OLD.acmId) AND length(error_msg) < 80 THEN
		SET error_msg := concat(error_msg, 'Unable to set acmId ');
	END IF;
	# acmKeyId cannot be changed
	IF (NEW.acmKeyId <> OLD.acmKeyId) AND length(error_msg) < 74 THEN
		SET error_msg := concat(error_msg, 'Unable to set acmKeyId ');
	END IF;
	# acmValueId cannot be changed
	IF (NEW.acmValueId <> OLD.acmValueId) AND length(error_msg) < 74 THEN
		SET error_msg := concat(error_msg, 'Unable to set acmValueId ');
	END IF;
	IF length(error_msg) > 0 THEN
		SET error_msg := concat(error_msg, 'when updating record');
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

	# Archive table
	INSERT INTO a_acmpart
	(
		id
		,createdDate
		,createdBy
		,modifiedDate
		,modifiedBy
		,isDeleted
		,deletedDate
		,deletedBy
		,acmId
		,acmKeyId
		,acmValueId
	) VALUES (
		NEW.id
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.isDeleted
		,NEW.deletedDate
		,NEW.deletedBy
		,NEW.acmId
		,NEW.acmKeyId
		,NEW.acmValueId
	);

	# Specific field level changes (should be none. Only change isDeleted is allowed)
	IF NEW.acmId <> OLD.acmId THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmId', newValue = hex(NEW.acmId);
	END IF;
	IF NEW.acmKeyId <> OLD.acmKeyId THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmKeyId', newValue = hex(NEW.acmKeyId);
	END IF;
	IF NEW.acmValueId <> OLD.acmValueId THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmValueId', newValue = hex(NEW.acmValueId);
	END IF;
END
//
SELECT 'Creating trigger td_acmpart' as Action
//
CREATE TRIGGER td_acmpart
BEFORE DELETE ON acmpart FOR EACH ROW
BEGIN
	# DECLARE error_msg varchar(128) default 'Deleting records are not allowed. Use isDeleted, deletedDate, and deletedBy';
	# signal sqlstate '45000' set message_text = error_msg;
END
//
SELECT 'Creating trigger td_a_acmpart' as Action
//
CREATE TRIGGER td_a_acmpart
BEFORE DELETE ON a_acmpart FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed on archive table.';
	signal sqlstate '45000' set message_text = error_msg;
END
//
delimiter ;
