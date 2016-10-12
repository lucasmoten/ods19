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

END;
