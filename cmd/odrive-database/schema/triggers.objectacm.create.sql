delimiter //
SELECT 'Creating trigger ti_objectacm' as Action
//
CREATE TRIGGER ti_objectacm
BEFORE INSERT ON objectacm FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'objectacm';
	
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
		a_objectacm
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
		,acmId
	) VALUES (
		NEW.id
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.isDeleted
		,NEW.deletedDate
		,NEW.deletedBy
		,NEW.objectId
		,NEW.acmId
	);

	# Specific field level changes
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'objectId', newValue = hex(NEW.objectId);
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmId', newValue = hex(NEW.acmId);

END
//
SELECT 'Creating trigger tu_objectacm' as Action
//
CREATE TRIGGER tu_objectacm
BEFORE UPDATE ON objectacm FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'objectacm';	

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
	# objectId cannot be changed
	IF (NEW.objectId <> OLD.objectId) AND length(error_msg) < 80 THEN
		SET error_msg := concat(error_msg, 'Unable to set objectId ');
	END IF;
    # either isDeleted or acmId must be different
    IF (NEW.isDeleted = OLD.isDeleted) AND (NEW.acmId = OLD.acmId) THEN
        set error_msg := concat(error_msg, 'Must either change acm or mark as deleted ');
    END IF;
    # if error, report
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
	INSERT INTO a_objectacm
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
		,acmId
	) VALUES (
		NEW.id
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.isDeleted
		,NEW.deletedDate
		,NEW.deletedBy
		,NEW.objectId
		,NEW.acmId
	);

	# Specific field level changes (should only be acmId or isDeleted)
	IF NEW.acmId <> OLD.acmId THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmId', newValue = hex(NEW.acmId);
	END IF;
END
//
SELECT 'Creating trigger td_objectacm' as Action
//
CREATE TRIGGER td_objectacm
BEFORE DELETE ON objectacm FOR EACH ROW
BEGIN
	# DECLARE error_msg varchar(128) default 'Deleting records are not allowed. Use isDeleted, deletedDate, and deletedBy';
	# signal sqlstate '45000' set message_text = error_msg;
END
//
SELECT 'Creating trigger td_a_objectacm' as Action
//
CREATE TRIGGER td_a_objectacm
BEFORE DELETE ON a_objectacm FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed on archive table.';
	signal sqlstate '45000' set message_text = error_msg;
END
//
delimiter ;
