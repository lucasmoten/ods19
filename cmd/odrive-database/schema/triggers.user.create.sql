delimiter //
SELECT 'Creating trigger ti_user' as Action
//
CREATE TRIGGER ti_user
BEFORE INSERT ON user FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'user';
	DECLARE count_user int default 0;

	# Rules
	# distinguishedName must be specified
	IF NEW.distinguishedName IS NULL OR NEW.distinguishedName = '' THEN
		set error_msg := concat(error_msg, 'Field distinguishedName required ');
	END IF;
	# distinguishedName must be unique
	SELECT COUNT(*) FROM user WHERE distinguishedName = NEW.distinguishedName INTO count_user;
	IF count_user > 0 THEN
		SET error_msg := concat(error_msg, 'Field distinguishedName must be unique ');
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
	SET NEW.changeCount := 0;
	# Standard change token formula
	SET NEW.changeToken := md5(CONCAT(CAST(NEW.ID AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));

	# Archive table
	INSERT INTO
		a_user
	(
		id
		,createdDate
		,createdBy
		,modifiedDate
		,modifiedBy
		,changeCount
		,changeToken
		,distinguishedName
		,displayName
		,email
	) values (
		NEW.ID
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.changeCount
		,NEW.changeToken
		,NEW.distinguishedName
		,NEW.displayName
		,NEW.email
	);

	# Specific field level changes
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.ID, tableName = thisTableName, columnName = 'distinguishedName', newValue = NEW.distinguishedName;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.ID, tableName = thisTableName, columnName = 'displayName', newValue = NEW.displayName;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.ID, tableName = thisTableName, columnName = 'email', newValue = NEW.email;

END
//
SELECT 'Creating trigger tu_user' as Action
//
CREATE TRIGGER tu_user
BEFORE UPDATE ON user FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'user';

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
	# changeToken must be given and match the record
	IF (NEW.changeToken IS NULL OR NEW.changeToken = '') and length(error_msg) < 73 THEN
		SET error_msg := concat(error_msg, 'Field changeToken required ');
	END IF;
	IF (NEW.changeToken <> OLD.changeToken) and length(error_msg) < 71 THEN
		SET error_msg := concat(error_msg, 'Field changeToken must match ');
	END IF;
	# distinguishedName cannot be changed
	IF (NEW.distinguishedName <> OLD.distinguishedName) AND length(error_msg) < 68 THEN
		SET error_msg := concat(error_msg, 'Unable to set distinguishedName ');
	END IF;
	IF length(error_msg) > 0 THEN
		SET error_msg := concat(error_msg, 'when updating record');
		signal sqlstate '45000' set message_text = error_msg;
	END IF;

	# Force values on modify
	SET NEW.modifiedDate := current_timestamp();
	SET NEW.changeCount := OLD.changeCount + 1;
	# Standard change token formula
	SET NEW.changeToken := md5(CONCAT(CAST(OLD.ID AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));

	# Archive table
	INSERT INTO
		a_user
	(
		id
		,createdDate
		,createdBy
		,modifiedDate
		,modifiedBy
		,changeCount
		,changeToken
		,distinguishedName
		,displayName
		,email
	) values (
		NEW.ID
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.changeCount
		,NEW.changeToken
		,NEW.distinguishedName
		,NEW.displayName
		,NEW.email
	);

	# Specific field level changes
	IF NEW.distinguishedName <> OLD.distinguishedName THEN
		# not possible
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.ID, tableName = thisTableName, columnName = 'distinguishedName', newValue = NEW.distinguishedName;
	END IF;
	IF NEW.displayName <> OLD.displayName THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.ID, tableName = thisTableName, columnName = 'displayName', newValue = NEW.displayName;
	END IF;
	IF NEW.email <> OLD.email THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.ID, tableName = thisTableName, columnName = 'email', newValue = NEW.email;
	END IF;

END
//
SELECT 'Creating trigger td_user' as Action
//
CREATE TRIGGER td_user
BEFORE DELETE ON user FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed.';
	signal sqlstate '45000' set message_text = error_msg;
END
//
SELECT 'Creating trigger td_a_user' as Action
//
CREATE TRIGGER td_a_user
BEFORE DELETE ON a_user FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed on archive tables.';
	signal sqlstate '45000' set message_text = error_msg;
END
//
delimiter ;
