DROP PROCEDURE IF EXISTS sp_drop_constraints;

DELIMITER //
CREATE PROCEDURE sp_drop_constraints(
   IN refschema VARCHAR(64) CHARSET utf8 COLLATE utf8_unicode_ci, 
   IN reftable VARCHAR(64) CHARSET utf8 COLLATE utf8_unicode_ci, 
   IN refcolumn VARCHAR(64) CHARSET utf8 COLLATE utf8_unicode_ci
)
BEGIN
    WHILE EXISTS(
        SELECT * FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
        WHERE 1
        AND REFERENCED_TABLE_SCHEMA COLLATE utf8_unicode_ci = refschema
        AND REFERENCED_TABLE_NAME COLLATE utf8_unicode_ci = reftable
        AND REFERENCED_COLUMN_NAME COLLATE utf8_unicode_ci = refcolumn
    ) DO
        BEGIN
            SET @sqlstmt = (
                SELECT CONCAT('ALTER TABLE ',TABLE_SCHEMA,'.',TABLE_NAME,' DROP FOREIGN KEY ',CONSTRAINT_NAME)
                FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
                WHERE 1
                AND REFERENCED_TABLE_SCHEMA COLLATE utf8_unicode_ci = refschema
                AND REFERENCED_TABLE_NAME COLLATE utf8_unicode_ci = reftable
                AND REFERENCED_COLUMN_NAME COLLATE utf8_unicode_ci = refcolumn
                LIMIT 1
            );
            PREPARE stmt1 FROM @sqlstmt;
            EXECUTE stmt1;
        END;
    END WHILE;
END//
DELIMITER ;

SET @SchemaName = database();
CALL sp_drop_constraints(@SchemaName, 'acm', 'id');
CALL sp_drop_constraints(@SchemaName, 'acm_accm', 'id');
CALL sp_drop_constraints(@SchemaName, 'acm_coicontrol', 'id');
CALL sp_drop_constraints(@SchemaName, 'acm_mac', 'id');
CALL sp_drop_constraints(@SchemaName, 'acm_project', 'id');
CALL sp_drop_constraints(@SchemaName, 'acm_share', 'id');
CALL sp_drop_constraints(@SchemaName, 'acmkey', 'id');
CALL sp_drop_constraints(@SchemaName, 'acmpart', 'id');
CALL sp_drop_constraints(@SchemaName, 'acmvalue', 'id');
CALL sp_drop_constraints(@SchemaName, 'object', 'id');
CALL sp_drop_constraints(@SchemaName, 'object_type', 'id');
CALL sp_drop_constraints(@SchemaName, 'property', 'id');
CALL sp_drop_constraints(@SchemaName, 'user', 'distinguishedName');

DROP PROCEDURE IF EXISTS sp_drop_constraints;
