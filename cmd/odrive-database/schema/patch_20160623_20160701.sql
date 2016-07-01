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

DROP TRIGGER IF EXISTS ti_object_permission;
DROP TRIGGER IF EXISTS tu_object_permission;
DROP TRIGGER IF EXISTS td_object_permission;
DROP TRIGGER IF EXISTS td_a_object_permission;
ALTER TABLE object_permission ADD permissionIV binary(32);
ALTER TABLE object_permission ADD permissionMAC binary(32);
ALTER TABLE a_object_permission ADD permissionIV binary(32);
ALTER TABLE a_object_permission ADD permissionMAC binary(32);

source triggers.object_permission.create.sql;
source function.bitwise256_xor.sql;
source function.keys.sql;
source procedure.migrate_keys.sql;
source procedure.rotate_keys.sql;

# Update schema version
UPDATE dbstate SET schemaVersion = '20160701';
