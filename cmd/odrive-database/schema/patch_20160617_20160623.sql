delimiter //

# DDL Performed outside of function as explicit or implicit commits are not allowed
# Set collation and characterset
ALTER DATABASE CHARACTER SET utf8 COLLATE utf8_unicode_ci//
SET character_set_client = utf8//
SET collation_connection = @@collation_database//

  
DROP FUNCTION IF EXISTS PatchDatabase //    
CREATE FUNCTION PatchDatabase (
    expectedVersion varchar(255)
) RETURNS varchar(255)
BEGIN 
    DECLARE newVersion varchar(255);
    DECLARE currentVersion varchar(255);
    SELECT schemaVersion FROM dbstate INTO currentVersion;
    IF expectedVersion = currentVersion THEN
        # Update schema version
        update dbstate set schemaVersion = '20160623';
        # Get reported version
        select schemaVersion from dbstate INTO newVersion;
        # Report result
        return CONCAT('Patch successfully applied: ',currentVersion,' > ',newVersion); 
    ELSE 
        # Report failure
        return CONCAT('Patch not applied. Current schemaversion is ',currentVersion,' expected ',expectedVersion); 
  END IF; 
END//
DELIMITER ;

# Expect existing version to be 20160617
SELECT PatchDatabase('20160617');

DROP FUNCTION IF EXISTS PatchDatabase;



