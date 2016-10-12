CREATE TRIGGER td_dbstate
BEFORE DELETE ON dbstate FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records from dbstate are not allowed.';
	signal sqlstate '45000' set message_text = error_msg;
END;
