delimiter //
SELECT 'Creating trigger ti_object_permission' as Action
//
CREATE TRIGGER ti_object_permission
BEFORE INSERT ON object_permission FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'object_permission';
	DECLARE count_object int default 0;

	# Rules
	# ObjectId must be specified
	SELECT COUNT(*) FROM object WHERE isDeleted = 0 AND id = NEW.objectId INTO count_object;
	IF count_object = 0 THEN
		SET error_msg := concat(error_msg, 'Field objectId required ');
	END IF;
	# Grantee must be specified
	IF NEW.grantee IS NULL OR NEW.grantee = '' THEN
		SET error_msg := concat(error_msg, 'Field grantee required ');
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
	SET NEW.changeCount := 0;
	# Standard change token formula
	SET NEW.changeToken := md5(CONCAT(CAST(NEW.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));

	# Archive table
	INSERT INTO
		a_object_permission
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
		,objectId
		,grantee
		,allowCreate
		,allowRead
		,allowUpdate
		,allowDelete
		,allowShare
		,explicitShare
		,encryptKey
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
		,NEW.objectId
		,NEW.grantee
		,NEW.allowCreate
		,NEW.allowRead
		,NEW.allowUpdate
		,NEW.allowDelete
		,NEW.allowShare
		,NEW.explicitShare
		,NEW.encryptKey
	);

	# Specific field level changes
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'objectId', newValue = hex(NEW.objectId);
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'grantee', newValue = NEW.grantee;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowCreate', newValue = NEW.allowCreate;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowRead', newValue = NEW.allowRead;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowUpdate', newValue = NEW.allowUpdate;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowDelete', newValue = NEW.allowDelete;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowShare', newValue = NEW.allowShare;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'explicitShare', newValue = NEW.explicitShare;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'encryptKey', newValue = hex(NEW.encryptKey);

END
//
SELECT 'Creating trigger tu_object_permission' as Action
//
CREATE TRIGGER tu_object_permission
BEFORE UPDATE ON object_permission FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'object_permission';

	# Rules
	# id cannot be changed
	IF (NEW.id <> OLD.id) AND length(error_msg) < 83 THEN
		SET error_msg := concat(error_msg, 'Unable to set id ');
	END IF;
	# objectId cannot be changed
	IF (NEW.objectId <> OLD.objectId) AND length(error_msg) < 77 THEN
		SET error_msg := concat(error_msg, 'Unable to set objectId ');
	END IF;
	# createdDate cannot be changed
	IF (NEW.createdDate <> OLD.createdDate) AND length(error_msg) < 74 THEN
		SET error_msg := concat(error_msg, 'Unable to set createdDate ');
	END IF;
	# createdBy cannot be changed
	IF (NEW.createdBy <> OLD.createdBy) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set createdBy ');
	END IF;
	# changeCount cannot be changed
	IF (NEW.changeCount <> OLD.changeCount) AND length(error_msg) < 74 THEN
		SET error_msg := concat(error_msg, 'Unable to set changeCount ');
	END IF;
	# changeToken must be given and match the record
	IF (NEW.changeToken IS NULL OR NEW.changeToken = '') and length(error_msg) < 73 THEN
		SET error_msg := concat(error_msg, 'Field changeToken required ');
	END IF;
	IF (NEW.changeToken <> OLD.changeToken) and length(error_msg) < 71 THEN
		SET error_msg := concat(error_msg, 'Field changeToken must match ');
	END IF;
	# grantee cannot be changed
	IF (NEW.grantee <> OLD.grantee) AND length(error_msg) < 78 THEN
		SET error_msg := concat(error_msg, 'Unable to set grantee ');
	END IF;
	# deleted MUST be set
	IF (NEW.isDeleted = OLD.isDeleted) AND length(error_msg) < 77 THEN
		SET error_msg := concat(error_msg, 'Permission not marked deleted ');
	END IF;
	# deletedBy must be set
	IF (NEW.deletedBy IS NULL) AND length(error_msg) < 75 THEN
		SET error_msg := concat(error_msg, 'Field deletedBy required ');
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
	SET NEW.changeCount := OLD.changeCount + 1;
	# Standard change token formula
	SET NEW.changeToken := md5(CONCAT(CAST(OLD.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));

	# Archive table
	INSERT INTO
		a_object_permission
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
		,objectId
		,grantee
		,allowCreate
		,allowRead
		,allowUpdate
		,allowDelete
		,allowShare
		,explicitShare
		,encryptKey
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
		,NEW.objectId
		,NEW.grantee
		,NEW.allowCreate
		,NEW.allowRead
		,NEW.allowUpdate
		,NEW.allowDelete
		,NEW.allowShare
		,NEW.explicitShare
		,NEW.encryptKey
	);

	# Specific field level changes
	IF NEW.objectId <> OLD.objectId THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'objectId', newValue = hex(NEW.objectId);
	END IF;
	IF NEW.grantee <> OLD.grantee THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'grantee', newValue = NEW.grantee;
	END IF;
	IF NEW.allowCreate <> OLD.allowCreate THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowCreate', newValue = NEW.allowCreate;
	END IF;
	IF NEW.allowRead <> OLD.allowRead THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowRead', newValue = NEW.allowRead;
	END IF;
	IF NEW.allowUpdate <> OLD.allowUpdate THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowUpdate', newValue = NEW.allowUpdate;
	END IF;
	IF NEW.allowDelete <> OLD.allowDelete THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowDelete', newValue = NEW.allowDelete;
	END IF;
	IF NEW.allowShare <> OLD.allowShare THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowShare', newValue = NEW.allowShare;
	END IF;
	IF NEW.explicitShare <> OLD.explicitShare THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'explicitShare', newValue = NEW.explicitShare;
	END IF;
	IF NEW.encryptKey <> OLD.encryptKey THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'encryptKey', newValue = hex(NEW.encryptKey);
	END IF;

END
//
SELECT 'Creating trigger td_object_permission' as Action
//
CREATE TRIGGER td_object_permission
BEFORE DELETE ON object_permission FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed. Use isDeleted, deletedDate, and deletedBy';
	signal sqlstate '45000' set message_text = error_msg;
END
//
SELECT 'Creating trigger td_a_object_permission' as Action
//
CREATE TRIGGER td_a_object_permission
BEFORE DELETE ON a_object_permission FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed on archive tables.';
	signal sqlstate '45000' set message_text = error_msg;
END
//
delimiter ;
