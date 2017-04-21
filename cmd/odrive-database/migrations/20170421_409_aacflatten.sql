-- +migrate Up

INSERT INTO migration_status SET description = '20170421_409_aacflatten.sql recreating aacflatten function';

drop function if exists aacflatten;
-- +migrate StatementBegin
CREATE FUNCTION aacflatten(dn varchar(255)) RETURNS varchar(255) DETERMINISTIC
BEGIN
    DECLARE o varchar(255);

    SET o := LOWER(dn);
    -- empty list
    SET o := REPLACE(o, ' ', '');
    SET o := REPLACE(o, ',', '');
    SET o := REPLACE(o, '=', '');
    SET o := REPLACE(o, '''', '');
    SET o := REPLACE(o, ':', '');
    SET o := REPLACE(o, '(', '');
    SET o := REPLACE(o, ')', '');
    SET o := REPLACE(o, '$', '');
    SET o := REPLACE(o, '[', '');
    SET o := REPLACE(o, ']', '');
    SET o := REPLACE(o, '{', '');
    SET o := REPLACE(o, ']', '');
    SET o := REPLACE(o, '|', '');
    SET o := REPLACE(o, '\\', '');
    -- underscore list
    SET o := REPLACE(o, '.', '_');
    SET o := REPLACE(o, '-', '_');
    RETURN o;
END;
-- +migrate StatementEnd

-- dbstate
DROP TRIGGER IF EXISTS ti_dbstate;
INSERT INTO migration_status SET description = '20170421_409_aacflatten.sql setting schema version';
-- +migrate StatementBegin
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
	SET NEW.createdDate := current_timestamp(6);
	# Modified Date
	SET NEW.modifiedDate := current_timestamp(6);
	# Version should be changed if the schema changes
	SET NEW.schemaversion := '20170421'; 
	# Identifier is randomized as a GUID
	SET NEW.identifier := concat(@@hostname, '-', left(uuid(),8));
END;
-- +migrate StatementEnd
update dbstate set schemaVersion = '20170421';

-- +migrate Down


DROP TRIGGER IF EXISTS ti_dbstate;

-- +migrate StatementBegin
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
	SET NEW.createdDate := current_timestamp(6);
	# Modified Date
	SET NEW.modifiedDate := current_timestamp(6);
	# Version should be changed if the schema changes
	SET NEW.schemaversion := '20170331'; 
	# Identifier is randomized as a GUID
	SET NEW.identifier := concat(@@hostname, '-', left(uuid(),8));
END;
-- +migrate StatementEnd
update dbstate set schemaVersion = '20170331';