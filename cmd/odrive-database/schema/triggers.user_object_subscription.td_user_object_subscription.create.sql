CREATE TRIGGER td_user_object_subscription
BEFORE DELETE ON user_object_subscription FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed. Use isDeleted, deletedDate and deletedBy';
	signal sqlstate '45000' set message_text = error_msg;
END
