# This script migrates the database schema from version 20160701 to 20160822

delimiter //

CREATE PROCEDURE PatchDatabaseTo20160822(
    expectedVersion varchar(255)    
) 
BEGIN
    DECLARE newVersion varchar(255);
    DECLARE currentVersion varchar(255);
    SELECT '20160822' INTO newVersion;
    SELECT schemaVersion FROM dbstate INTO currentVersion;
    IF expectedVersion = currentVersion THEN
        # Add new columns to object, a_object tables
        ALTER TABLE a_object ADD COLUMN containsUSPersonsData varchar(255) null;
        ALTER TABLE a_object ADD COLUMN exemptFromFOIA varchar(255) null;
        ALTER TABLE object ADD COLUMN containsUSPersonsData varchar(255) null;
        ALTER TABLE object ADD COLUMN exemptFromFOIA varchar(255) null;

        # Migration
        UPDATE object SET containsUSPersonsData = 'Yes' WHERE IsUSPersonsData = 1;
        UPDATE object SET containsUSPersonsData = 'No' WHERE IsUSPersonsData = 0;
        UPDATE object SET containsUSPersonsData = 'Unknown' WHERE containsUSPersonsData IS NULL;
        UPDATE object SET exemptFromFOIA = 'Yes' WHERE IsFOIAExempt = 1;
        UPDATE object SET exemptFromFOIA = 'No' WHERE IsFOIAExempt = 0;
        UPDATE object SET exemptFromFOIA = 'Unknown' WHERE exemptFromFOIA IS NULL;
        UPDATE a_object SET containsUSPersonsData = 'Yes' WHERE IsUSPersonsData = 1;
        UPDATE a_object SET containsUSPersonsData = 'No' WHERE IsUSPersonsData = 0;
        UPDATE a_object SET containsUSPersonsData = 'Unknown' WHERE containsUSPersonsData IS NULL;
        UPDATE a_object SET exemptFromFOIA = 'Yes' WHERE IsFOIAExempt = 1;
        UPDATE a_object SET exemptFromFOIA = 'No' WHERE IsFOIAExempt = 0;
        UPDATE a_object SET exemptFromFOIA = 'Unknown' WHERE exemptFromFOIA IS NULL;
        UPDATE field_changes SET columnName = 'containsUSPersonsData', newValue = 'Yes' WHERE tableName = 'object' and columnName = 'IsUSPersonsData' and newValue = '1';
        UPDATE field_changes SET columnName = 'containsUSPersonsData', newValue = 'No' WHERE tableName = 'object' and columnName = 'IsUSPersonsData' and newValue = '0';
        UPDATE field_changes SET columnName = 'exemptFromFOIA', newValue = 'Yes' where tableName = 'object' and columnName = 'IsFOIAExempt' and newValue = '1';
        UPDATE field_changes SET columnName = 'exemptFromFOIA', newValue = 'No' where tableName = 'object' and columnName = 'IsFOIAExempt' and newValue = '0';

        # Drop old columns
        ALTER TABLE a_object DROP COLUMN IsUSPersonsData;
        ALTER TABLE a_object DROP COLUMN IsFOIAExempt;
        ALTER TABLE object DROP column IsUSPersonsData;
        ALTER TABLE object DROP column IsFOIAExempt;
                
        # Update schema version
        UPDATE dbstate SET schemaVersion = newVersion;    
    ELSE
        IF currentVersion <> newVersion THEN
            # signal failure
            signal sqlstate '45000' set message_text = 'Database Schema Version is different then Expected Version';
        END IF;
    END IF;
END;
//

delimiter ;

# Remove triggers for object
DROP TRIGGER IF EXISTS ti_object;
DROP TRIGGER IF EXISTS tu_object;
DROP TRIGGER IF EXISTS td_object;
DROP TRIGGER IF EXISTS td_a_object;

# Apply Patch
SET @expectedVersion = '20160701';
CALL PatchDatabaseTo20160822(@expectedVersion);
DROP PROCEDURE PatchDatabaseTo20160822;

# Rebuild triggers for object
source triggers.object.create.sql
