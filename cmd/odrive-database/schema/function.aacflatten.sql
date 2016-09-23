DELIMITER //
CREATE FUNCTION aacflatten(dn varchar(255)) RETURNS varchar(255) DETERMINISTIC
BEGIN
    DECLARE o varchar(255);

    SET o := dn;
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

END//
DELIMITER ;
