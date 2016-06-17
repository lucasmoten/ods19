delimiter //
SELECT 'Creating trigger ti_object_type' as Action
//
CREATE TRIGGER ti_object_type
BEFORE INSERT ON object_type FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'object_type';
	DECLARE count_name int default 0;

	# Rules
	# Name must be unique for non-deletedBy
	SELECT COUNT(*) FROM object_type WHERE isDeleted = 0 AND name = NEW.name INTO count_name;
	IF count_name > 0 THEN
		SET error_msg := concat(error_msg, 'Field name must be unique ');
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
	# Assign Owner if not set
	IF NEW.ownedBy IS NULL OR NEW.ownedBy = '' THEN
		SET NEW.ownedBy := NEW.createdBy;
	END IF;
	SET NEW.changeCount := 0;
	# Standard change token formula
	SET NEW.changeToken := md5(CONCAT(CAST(NEW.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));

	# Archive table
	INSERT INTO
		a_object_type
	(
		id
		,createdDate
		,createdBy
		,modifiedDate
		,modifiedBy
		,isDeleted
		,deletedDate
		,deletedBy
		,ownedBy
		,changeCount
		,changeToken
		,name
		,description
		,contentConnector
	) values (
		NEW.id
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.isDeleted
		,NEW.deletedDate
		,NEW.deletedBy
		,NEW.ownedBy
		,NEW.changeCount
		,NEW.changeToken
		,NEW.name
		,NEW.description
		,NEW.contentConnector
	);

	# Specific field level changes
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'name', newValue = NEW.name;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'description', newValue = NEW.description;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentConnector', newValue = NEW.contentConnector;

END
//
SELECT 'Creating trigger tu_object_type' as Action
//
CREATE TRIGGER tu_object_type
BEFORE UPDATE ON object_type FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'object_type';
	DECLARE count_name int default 0;

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
	# changeCount cannot be changed
	IF (NEW.changeCount <> OLD.changeCount) AND length(error_msg) < 74 THEN
		SET error_msg := concat(error_msg, 'Unable to set changeCount ');
	END IF;
	# Name must be unique for non-deletedBy
	SELECT COUNT(*) FROM object_type WHERE isDeleted = 0 AND name = NEW.name AND id <> OLD.id INTO count_name;
	IF (count_name > 0) AND length(error_msg) < 74 THEN
		SET error_msg := concat(error_msg, 'Field name must be unique ');
	END IF;
	# changeToken must be given and match the record
	IF (NEW.changeToken IS NULL OR NEW.changeToken = '') and length(error_msg) < 73 THEN
		SET error_msg := concat(error_msg, 'Field changeToken required ');
	END IF;
	IF (NEW.changeToken <> OLD.changeToken) and length(error_msg) < 71 THEN
		SET error_msg := concat(error_msg, 'Field changeToken must match ');
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
    SET NEW.changeCount := OLD.changeCount + 1;
	# Standard change token formula
	SET NEW.changeToken := md5(CONCAT(CAST(OLD.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));
	# Assign Owner if not set
	IF NEW.ownedBy IS NULL OR NEW.ownedBy = '' THEN
		SET NEW.ownedBy := NEW.createdBy;
	END IF;

	# Archive table
	INSERT INTO
		a_object_type
	(
		id
		,createdDate
		,createdBy
		,modifiedDate
		,modifiedBy
		,isDeleted
		,deletedDate
		,deletedBy
		,ownedBy
		,changeCount
		,changeToken
		,name
		,description
		,contentConnector
	) values (
		NEW.id
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.isDeleted
		,NEW.deletedDate
		,NEW.deletedBy
		,NEW.ownedBy
		,NEW.changeCount
		,NEW.changeToken
		,NEW.name
		,NEW.description
		,NEW.contentConnector
	);

	# Specific field level changes
	IF NEW.name <> OLD.name THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'name', newValue = NEW.name;
	END IF;
	IF NEW.description <> OLD.description THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'description', newValue = NEW.description;
	END IF;
	IF NEW.contentConnector <> OLD.contentConnector THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentConnector', newValue = NEW.contentConnector;
	END IF;

END
//
SELECT 'Creating trigger td_object_type' as Action
//
CREATE TRIGGER td_object_type
BEFORE DELETE ON object_type FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed. Use isDeleted, deletedDate, and deletedBy';
	signal sqlstate '45000' set message_text = error_msg;
END
//
SELECT 'Creating trigger td_a_object_type' as Action
//
CREATE TRIGGER td_a_object_type
BEFORE DELETE ON a_object_type FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed on archive tables.';
	signal sqlstate '45000' set message_text = error_msg;
END
//
delimiter ;
