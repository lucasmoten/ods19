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
END;
