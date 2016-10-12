CREATE TRIGGER td_user_object_favorite
BEFORE DELETE ON user_object_favorite FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed. User isDeleted, deletedDate, and deletedBy';
	signal sqlstate '45000' set message_text = error_msg;
END;
