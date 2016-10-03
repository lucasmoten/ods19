CREATE TRIGGER td_a_user_object_subscription
BEFORE DELETE ON a_user_object_subscription FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed on archive tables.';
	signal sqlstate '45000' set message_text = error_msg;
END
