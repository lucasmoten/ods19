CREATE TRIGGER td_property
BEFORE DELETE ON property FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default 'Deleting records are not allowed. Use isDeleted, deletedDate, and deletedBy';
	signal sqlstate '45000' set message_text = error_msg;
END;
