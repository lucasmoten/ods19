CREATE TRIGGER td_field_changes
BEFORE DELETE ON field_changes FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records from field_changes history tracking table are not allowed.';
	signal sqlstate '45000' set message_text = error_msg;
END
