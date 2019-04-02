DROP PROCEDURE IF EXISTS sp_Patch_20190225_fix_acmpart2_associations; 
DELIMITER //
CREATE PROCEDURE sp_Patch_20190225_fix_acmpart2_associations()
BEGIN
    -- declare variables
    DECLARE vACMID int default 0;
    DECLARE vACMName text default '';
    DECLARE vSHA256Hash char(64) default '';
    DECLARE vKeyValuePart varchar(2000) default '';
    DECLARE vKeyID int default 0;
    DECLARE vKeyName varchar(255) default '';
    DECLARE vValueID int default 0;
    DECLARE vValueName varchar(255) default '';
    DECLARE vPartID int default 0;
    DECLARE c_acm2_finished int default 0;
    DECLARE c_acm2 cursor for SELECT id, flattenedacm from acm2 where id not in (select acmid from acmpart2);
    DECLARE continue handler for not found set c_acm2_finished = 1;

    INSERT INTO migration_status SET description = '20190225_fix_acmpart2_associations repairing acm2 rows that are missing acmpart2 rows';
    ACMPOPULATE: BEGIN

        -- get list of acm2 rows that dont have acmpart2 rows
        OPEN c_acm2;
        -- with each acm2 row needing fixed (loop)
        get_acm2: LOOP
            -- get row into variables
            FETCH c_acm2 INTO vACMID, vACMName;
            IF c_acm2_finished = 1 THEN
                CLOSE c_acm2;
                LEAVE get_acm2;
            END IF;
            INSERT INTO migration_status SET description = concat('20190225_fix_acmpart2_associations analyzing acm with id ', vACMID);
            -- with each part (loop)
            get_acmnamepart2: LOOP
                -- check if still processing
                IF length(vACMName) = 0 THEN
                    LEAVE get_acmnamepart2;
                END IF;
                -- get next part as delineated by semicolon
                SET vKeyValuePart := substring_index(substring_index(vACMName, ';', 1), ';', -1);
                -- remove it from the head along with trailing semicolon leaving the tail to process on next loop
                SET vACMName := substr(vACMName, length(vKeyValuePart)+2);
                -- get key name
                SET vKeyName := substring_index(substring_index(vKeyValuePart, '=', 1), '=', -1);
                -- remove key from the part with trailing equal sign leaving only the values
                SET vKeyValuePart := substr(vKeyValuePart, length(vKeyName)+2);
                -- insert the key name and get its id
                IF (SELECT 1=1 FROM acmkey2 WHERE name = vKeyName) IS NULL THEN
                    INSERT INTO migration_status SET description = concat('20190225_fix_acmpart2_associations adding missing acm key ', vKeyName, ' to acmkey2');
                    INSERT INTO acmkey2 (name) VALUES (vKeyName);
                    SET vKeyID := LAST_INSERT_ID();
                ELSE
                    SELECT id INTO vKeyID FROM acmkey2 WHERE name = vKeyName LIMIT 1;
                END IF;
                -- with each key/value (loop)
                get_acmvalue2: LOOP
                    -- check if still processing values
                    IF length(vKeyValuePart) = 0 THEN
                        LEAVE get_acmvalue2;
                    END IF;
                    -- get next value as delineated by comma
                    SET vValueName := substring_index(substring_index(vKeyValuePart, ',', 1), ',', -1);
                    -- remove it from the head along with trailing comma leaving the tail to process on next loop
                    SET vKeyValuePart := substr(vKeyValuePart, length(vValueName)+2);
                    -- insert the value name and get its id
                    IF (SELECT 1=1 FROM acmvalue2 WHERE name = vValueName) IS NULL THEN
                        INSERT INTO migration_status SET description = concat('20190225_fix_acmpart2_associations adding missing acm value ', vValueName, ' to acmvalue2');
                        INSERT INTO acmvalue2 SET name = vValueName;
                        SET vValueID := LAST_INSERT_ID();
                    ELSE
                        SELECT id INTO vValueID FROM acmvalue2 WHERE name = vValueName LIMIT 1;
                    END IF;
                    -- create association row between acm, key and value
                    IF (SELECT 1=1 FROM acmpart2 WHERE acmid = vACMID and acmkeyid = vKeyID and acmvalueid = vValueID) IS NULL THEN
                        INSERT INTO migration_status SET description = concat('20190225_fix_acmpart2_associations for acm with id ', vACMID, ': associating key ',vKeyName,' with value ',vValueName);
                        INSERT INTO acmpart2 (acmid, acmkeyid, acmvalueid) VALUES (vACMID, vKeyID, vValueID);
                        SET vPartID := LAST_INSERT_ID();
                    ELSE
                        SELECT id INTO vPartID FROM acmpart2 WHERE acmid = vACMID and acmkeyid = vKeyID and acmvalueid = vValueID LIMIT 1;
                    END IF;
                -- end loop processing values
                END LOOP get_acmvalue2;
            -- end loop for getting part
            END LOOP get_acmnamepart2;
        -- end loop for acm2
        END LOOP get_acm2;
    END ACMPOPULATE;
    INSERT INTO migration_status SET description = '20190225_fix_acmpart2_associations data repairing acm2 rows that are missing acmpart2 rows is done.';
END;
//
DELIMITER ;
CALL sp_Patch_20190225_fix_acmpart2_associations();
DROP PROCEDURE IF EXISTS sp_Patch_20190225_fix_acmpart2_associations; 
INSERT INTO migration_status SET description = '20190225_fix_acmpart2_associations clearing user ao cache and acm associations to force rebuild.';
DELETE FROM useraocachepart;
DELETE FROM useraocache;
DELETE FROM useracm;
