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
	# acmShare cannot be changed
	IF (NEW.acmShare <> OLD.acmShare) AND length(error_msg) < 78 THEN
		SET error_msg := concat(error_msg, 'Unable to set acmShare ');
	END IF;
	# permission cannot be changed
	IF (NEW.allowCreate <> OLD.allowCreate) AND length(error_msg) < 78 THEN
		SET error_msg := concat(error_msg, 'Unable to set allowCreate ');
	END IF;
	# permission cannot be changed
	IF (NEW.allowRead <> OLD.allowRead) AND length(error_msg) < 78 THEN
		SET error_msg := concat(error_msg, 'Unable to set allowRead ');
	END IF;
	# permission cannot be changed
	IF (NEW.allowUpdate <> OLD.allowUpdate) AND length(error_msg) < 78 THEN
		SET error_msg := concat(error_msg, 'Unable to set allowRead ');
	END IF;
	# permission cannot be changed
	IF (NEW.allowDelete <> OLD.allowDelete) AND length(error_msg) < 78 THEN
		SET error_msg := concat(error_msg, 'Unable to set allowDelete ');
	END IF;
	# permission cannot be changed
	IF (NEW.allowShare <> OLD.allowShare) AND length(error_msg) < 78 THEN
		SET error_msg := concat(error_msg, 'Unable to set allowShare ');
	END IF;
	
	#every immutable field has been checked for mutation at this point.
	#all other fields are mutable.

	#note... we need to always allow mutation of the encryptKey,permissionIV,permissionMAC, otherwise we will render things like
	#deleted fields unrecoverable.
	
	# Force values on modify
	# The only modification allowed is to mark as deleted...
	SET NEW.modifiedDate := current_timestamp();
	IF NEW.modifiedBy IS NULL OR NEW.modifiedBy = '' THEN
		SET NEW.modifiedBy := NEW.deletedBy;
	END IF;

	#either we are deleting... 		
	IF (NEW.isDeleted = 1 AND OLD.isDeleted = 0) THEN
		# deletedBy must be set
		IF (NEW.deletedBy IS NULL) AND length(error_msg) < 75 THEN
			SET error_msg := concat(error_msg, 'Field deletedBy required ');
		END IF;
		
		SET NEW.deletedDate := current_timestamp();
		IF NEW.deletedBy IS NULL OR NEW.deletedBy = '' THEN
			SET NEW.deletedBy := NEW.modifiedBy;
		END IF;
	ELSE
		#or updating keys
		IF (NEW.isDeleted <> OLD.isDeleted) THEN
			SET error_msg := concat(error_msg, 'Undelete is disallowed ');
		END IF;
		IF ((NEW.encryptKey = OLD.encryptKey) AND (NEW.permissionIV = OLD.permissionIV) AND (NEW.permissionMAC = OLD.permissionMAC)) THEN
			SET error_msg := concat(error_msg, 'We should be updating keys ');
		END IF;
	END IF; 

		
	IF length(error_msg) > 0 THEN
		SET error_msg := concat(error_msg, 'when updating record');
		signal sqlstate '45000' set message_text = error_msg;
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
        ,acmShare
		,allowCreate
		,allowRead
		,allowUpdate
		,allowDelete
		,allowShare
		,explicitShare
		,encryptKey
		,permissionIV
		,permissionMAC
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
        ,NEW.acmShare
		,NEW.allowCreate
		,NEW.allowRead
		,NEW.allowUpdate
		,NEW.allowDelete
		,NEW.allowShare
		,NEW.explicitShare
		,NEW.encryptKey
		,NEW.permissionIV
		,NEW.permissionMAC
	);

	# Specific field level changes
	IF NEW.objectId <> OLD.objectId THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'objectId', newValue = hex(NEW.objectId);
	END IF;
	IF NEW.grantee <> OLD.grantee THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'grantee', newValue = NEW.grantee;
	END IF;
	IF NEW.acmShare <> OLD.acmShare THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmShare', newTextValue = NEW.acmShare;
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
	IF NEW.permissionIV <> OLD.permissionIV THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'permissionIV', newValue = hex(NEW.permissionIV);
	END IF;
	IF NEW.permissionMAC <> OLD.permissionMAC THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'permissionMAC', newValue = hex(NEW.permissionMAC);
	END IF;

END;
