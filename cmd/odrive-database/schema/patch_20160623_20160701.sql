# This script migrates the database schema from version 20160623 to 20160701


# Remove foreign keys that are going away
ALTER TABLE object_acm DROP FOREIGN KEY fk_object_acm_createdBy;
ALTER TABLE object_acm DROP FOREIGN KEY fk_object_acm_deletedBy;
ALTER TABLE object_acm DROP FOREIGN KEY fk_object_acm_modifiedBy;
ALTER TABLE object_acm DROP FOREIGN KEY fk_object_acm_objectId;
ALTER TABLE object_acm DROP FOREIGN KEY fk_object_acm_acmKeyId;
ALTER TABLE object_acm DROP FOREIGN KEY fk_object_acm_acmValueId;

# Remove triggers that are going away
DROP TRIGGER IF EXISTS td_a_object_acm;
DROP TRIGGER IF EXISTS td_object_acm;
DROP TRIGGER IF EXISTS ti_object_acm;
DROP TRIGGER IF EXISTS tu_object_acm;

# Remove tables that are going away
DROP TABLE IF EXISTS a_object_acm;
DROP TABLE IF EXISTS object_acm;

# Update schema version
UPDATE dbstate SET schemaVersion = '20160701';
