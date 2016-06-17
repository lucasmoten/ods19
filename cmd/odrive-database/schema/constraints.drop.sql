USE metadatadb;

DROP PROCEDURE IF EXISTS sp_drop_constraints;

DELIMITER //
CREATE PROCEDURE sp_drop_constraints(refschema VARCHAR(64), reftable VARCHAR(64), refcolumn VARCHAR(64))
BEGIN
    WHILE EXISTS(
        SELECT * FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
        WHERE 1
        AND REFERENCED_TABLE_SCHEMA = refschema
        AND REFERENCED_TABLE_NAME = reftable
        AND REFERENCED_COLUMN_NAME = refcolumn
    ) DO
        BEGIN
            SET @sqlstmt = (
                SELECT CONCAT('ALTER TABLE ',TABLE_SCHEMA,'.',TABLE_NAME,' DROP FOREIGN KEY ',CONSTRAINT_NAME)
                FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
                WHERE 1
                AND REFERENCED_TABLE_SCHEMA = refschema
                AND REFERENCED_TABLE_NAME = reftable
                AND REFERENCED_COLUMN_NAME = refcolumn
                LIMIT 1
            );
            PREPARE stmt1 FROM @sqlstmt;
            EXECUTE stmt1;
        END;
    END WHILE;
END//
DELIMITER ;

CALL sp_drop_constraints('metadatadb', 'acm', 'id');
CALL sp_drop_constraints('metadatadb', 'acm_accm', 'id');
CALL sp_drop_constraints('metadatadb', 'acm_coicontrol', 'id');
CALL sp_drop_constraints('metadatadb', 'acm_mac', 'id');
CALL sp_drop_constraints('metadatadb', 'acm_project', 'id');
CALL sp_drop_constraints('metadatadb', 'acm_share', 'id');
CALL sp_drop_constraints('metadatadb', 'acmkey', 'id');
CALL sp_drop_constraints('metadatadb', 'acmpart', 'id');
CALL sp_drop_constraints('metadatadb', 'acmvalue', 'id');
CALL sp_drop_constraints('metadatadb', 'object', 'id');
CALL sp_drop_constraints('metadatadb', 'object_type', 'id');
CALL sp_drop_constraints('metadatadb', 'property', 'id');
CALL sp_drop_constraints('metadatadb', 'user', 'distinguishedName');

DROP PROCEDURE IF EXISTS sp_drop_constraints;
