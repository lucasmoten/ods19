CREATE TRIGGER td_a_objectacm
BEFORE DELETE ON a_objectacm FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed on archive table.';
	signal sqlstate '45000' set message_text = error_msg;
END;
