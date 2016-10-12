CREATE TRIGGER td_user
BEFORE DELETE ON user FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed.';
	signal sqlstate '45000' set message_text = error_msg;
END;
