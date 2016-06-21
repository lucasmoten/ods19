delimiter //
SELECT 'Creating trigger ti_dbstate' as Action
//
CREATE TRIGGER ti_dbstate
BEFORE INSERT ON dbstate FOR EACH ROW
BEGIN
	DECLARE count_rows int default 0;

	# Rules
	# Can only be one record
	SELECT count(0) FROM dbstate INTO count_rows;
	IF count_rows > 0 THEN
		signal sqlstate '45000' set message_text = 'Only one record is allowed in dbstate table.';
	END IF;

	# Force values on create
	# Created Date
	SET NEW.createdDate := current_timestamp();
	# Modified Date
	SET NEW.modifiedDate := current_timestamp();
	# Version should be changed if the schema changes
	SET NEW.schemaversion := '20160617'; 
	# Identifier is randomized as a GUID
	SET NEW.identifier := concat(@@hostname, '-', left(uuid(),8));
END
//
SELECT 'Creating trigger tu_dbstate' as Action
//
CREATE TRIGGER tu_dbstate
BEFORE UPDATE ON dbstate FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	# Rules
	# createdDate cannot be changed
	IF (NEW.createdDate <> OLD.createdDate) AND length(error_msg) < 74 THEN
		signal sqlstate '45000' set message_text = 'Unable to set createdDate ';
	END IF;
	# identifier cannot be changed
	IF (NEW.identifier <> OLD.identifier) THEN
		signal sqlstate '45000' set message_text = 'Identifier cannot be changed';
	END IF;
	# version must be different
	IF (NEW.schemaversion = OLD.schemaversion) THEN
		signal sqlstate '45000' set message_text = 'Version must be changed';
	END IF;

	# Force values
	# modifiedDate
	SET NEW.modifiedDate = current_timestamp();
END
//
SELECT 'Creating trigger td_dbstate' as Action
//
CREATE TRIGGER td_dbstate
BEFORE DELETE ON dbstate FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records from dbstate are not allowed.';
	signal sqlstate '45000' set message_text = error_msg;
END
//
delimiter ;
