-- +migrate Up

-- This is a combined migration script of all migrations in the first half of 2017 with some improvements to
-- avoid repetitive drop/creates and back to back cursor calls iterating records due to instnaces that have
-- not updated recently.

CREATE TABLE IF NOT EXISTS migration_status
(
    id int unsigned not null auto_increment
    ,description varchar(255)
    ,CONSTRAINT pk_migration_status PRIMARY KEY (id)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;

-- Drop triggers (all for tables being dropped, and ti/tu for those being kept and recreated later)
DROP TRIGGER IF EXISTS td_acm;
DROP TRIGGER IF EXISTS ti_acm;
DROP TRIGGER IF EXISTS tu_acm;
DROP TRIGGER IF EXISTS ti_acmgrantee;
DROP TRIGGER IF EXISTS tu_acmgrantee;
DROP TRIGGER IF EXISTS td_acmkey;
DROP TRIGGER IF EXISTS ti_acmkey;
DROP TRIGGER IF EXISTS tu_acmkey;
DROP TRIGGER IF EXISTS td_acmpart;
DROP TRIGGER IF EXISTS ti_acmpart;
DROP TRIGGER IF EXISTS tu_acmpart;
DROP TRIGGER IF EXISTS td_acmvalue;
DROP TRIGGER IF EXISTS ti_acmvalue;
DROP TRIGGER IF EXISTS tu_acmvalue;
DROP TRIGGER IF EXISTS td_a_acm;
DROP TRIGGER IF EXISTS td_a_acmkey;
DROP TRIGGER IF EXISTS td_a_acmpart;
DROP TRIGGER IF EXISTS td_a_acmvalue;
DROP TRIGGER IF EXISTS td_a_objectacm;
DROP TRIGGER IF EXISTS td_a_object_tag;
DROP TRIGGER IF EXISTS td_a_relationship;
DROP TRIGGER IF EXISTS td_a_user_object_favorite;
DROP TRIGGER IF EXISTS td_a_user_object_subscription;
DROP TRIGGER IF EXISTS ti_dbstate;
DROP TRIGGER IF EXISTS tu_dbstate;
DROP TRIGGER IF EXISTS td_field_changes;
DROP TRIGGER IF EXISTS ti_object;
DROP TRIGGER IF EXISTS tu_object;
DROP TRIGGER IF EXISTS td_objectacm;
DROP TRIGGER IF EXISTS ti_objectacm;
DROP TRIGGER IF EXISTS tu_objectacm;
DROP TRIGGER IF EXISTS ti_object_permission;
DROP TRIGGER IF EXISTS tu_object_permission;
DROP TRIGGER IF EXISTS ti_object_property;
DROP TRIGGER IF EXISTS tu_object_property;
DROP TRIGGER IF EXISTS td_object_tag;
DROP TRIGGER IF EXISTS ti_object_tag;
DROP TRIGGER IF EXISTS tu_object_tag;
DROP TRIGGER IF EXISTS ti_object_type;
DROP TRIGGER IF EXISTS tu_object_type;
DROP TRIGGER IF EXISTS ti_object_type_property;
DROP TRIGGER IF EXISTS tu_object_type_property;
DROP TRIGGER IF EXISTS ti_property;
DROP TRIGGER IF EXISTS tu_property;
DROP TRIGGER IF EXISTS td_relationship;
DROP TRIGGER IF EXISTS ti_relationship;
DROP TRIGGER IF EXISTS tu_relationship;
DROP TRIGGER IF EXISTS ti_user;
DROP TRIGGER IF EXISTS tu_user;
DROP TRIGGER IF EXISTS td_user_object_favorite;
DROP TRIGGER IF EXISTS ti_user_object_favorite;
DROP TRIGGER IF EXISTS tu_user_object_favorite;
DROP TRIGGER IF EXISTS td_user_object_subscription;
DROP TRIGGER IF EXISTS ti_user_object_subscription;
DROP TRIGGER IF EXISTS tu_user_object_subscription;

SET FOREIGN_KEY_CHECKS=0;

-- Drop foreign keys and indexes (but not primary keys)
INSERT INTO migration_status SET description = '20170630_combined drop foreign keys and indexes';
DROP PROCEDURE IF EXISTS sp_Patch_20170630_drop_keys_and_indexes_raw;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170630_drop_keys_and_indexes_raw()
proc_label: BEGIN
    -- only do this if not yet 20170630
    IF EXISTS( select null from dbstate where schemaversion = '20170630') THEN
        LEAVE proc_label;
    END IF;
    proc_main: BEGIN
        -- derived from constraints.create.sql
        INSERT INTO migration_status SET description = '20170630_combined drop foreign keys and indexes from constraints.create.sql';
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acm' and binary constraint_name = 'fk_acm_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acm_createdBy';
            ALTER TABLE `acm` DROP FOREIGN KEY `fk_acm_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acm' and binary index_name = 'fk_acm_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acm_createdBy';
            ALTER TABLE `acm` DROP INDEX `fk_acm_createdBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acm' and binary constraint_name = 'fk_acm_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acm_deletedBy';
            ALTER TABLE `acm` DROP FOREIGN KEY `fk_acm_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acm' and binary index_name = 'fk_acm_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acm_deletedBy';
            ALTER TABLE `acm` DROP INDEX `fk_acm_deletedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acm' and binary constraint_name = 'fk_acm_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acm_modifiedBy';
            ALTER TABLE `acm` DROP FOREIGN KEY `fk_acm_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acm' and binary index_name = 'fk_acm_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acm_modifiedBy';
            ALTER TABLE `acm` DROP INDEX `fk_acm_modifiedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmgrantee' and binary constraint_name = 'fk_acmgrantee_userDistinguishedName') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmgrantee_userDistinguishedName';
            ALTER TABLE `acmgrantee` DROP FOREIGN KEY `fk_acmgrantee_userDistinguishedName`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmgrantee' and binary index_name = 'fk_acmgrantee_userDistinguishedName') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acmgrantee_userDistinguishedName';
            ALTER TABLE `acmgrantee` DROP INDEX `fk_acmgrantee_userDistinguishedName`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmkey' and binary constraint_name = 'fk_acmkey_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmkey_createdBy';
            ALTER TABLE `acmkey` DROP FOREIGN KEY `fk_acmkey_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmkey' and binary index_name = 'fk_acmkey_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acmkey_createdBy';
            ALTER TABLE `acmkey` DROP INDEX `fk_acmkey_createdBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmkey' and binary constraint_name = 'fk_acmkey_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmkey_deletedBy';
            ALTER TABLE `acmkey` DROP FOREIGN KEY `fk_acmkey_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmkey' and binary index_name = 'fk_acmkey_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acmkey_deletedBy';
            ALTER TABLE `acmkey` DROP INDEX `fk_acmkey_deletedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmkey' and binary constraint_name = 'fk_acmkey_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmkey_modifiedBy';
            ALTER TABLE `acmkey` DROP FOREIGN KEY `fk_acmkey_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmkey' and binary index_name = 'fk_acmkey_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acmkey_modifiedby';
            ALTER TABLE `acmkey` DROP INDEX `fk_acmkey_modifiedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmpart' and binary constraint_name = 'fk_acmpart_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmpart_createdBy';
            ALTER TABLE `acmpart` DROP FOREIGN KEY `fk_acmpart_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmpart' and binary index_name = 'fk_acmpart_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acmpart_createdBy';
            ALTER TABLE `acmpart` DROP INDEX `fk_acmpart_createdBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmpart' and binary constraint_name = 'fk_acmpart_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmpart_deletedBy';
            ALTER TABLE `acmpart` DROP FOREIGN KEY `fk_acmpart_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmpart' and binary index_name = 'fk_acmpart_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acmpart_deletedBy';
            ALTER TABLE `acmpart` DROP INDEX `fk_acmpart_deletedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmpart' and binary constraint_name = 'fk_acmpart_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmpart_modifiedBy';
            ALTER TABLE `acmpart` DROP FOREIGN KEY `fk_acmpart_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmpart' and binary index_name = 'fk_acmpart_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acmpart_modifiedBy';
            ALTER TABLE `acmpart` DROP INDEX `fk_acmpart_modifiedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmpart' and binary constraint_name = 'fk_acmpart_acmId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmpart_acmId';
            ALTER TABLE `acmpart` DROP FOREIGN KEY `fk_acmpart_acmId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmpart' and binary index_name = 'fk_acmpart_acmId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acmpart_acmId';
            ALTER TABLE `acmpart` DROP INDEX `fk_acmpart_acmId`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmpart' and binary constraint_name = 'fk_acmpart_acmKeyId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmpart_acmKeyId';
            ALTER TABLE `acmpart` DROP FOREIGN KEY `fk_acmpart_acmKeyId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmpart' and binary index_name = 'fk_acmpart_acmKeyId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmpart_acmKeyId';
            ALTER TABLE `acmpart` DROP INDEX `fk_acmpart_acmKeyId`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmpart' and binary constraint_name = 'fk_acmpart_acmValueId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmpart_acmValueId';
            ALTER TABLE `acmpart` DROP FOREIGN KEY `fk_acmpart_acmValueId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmpart' and binary index_name = 'fk_acmpart_acmValueId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acmpart_acmValueId';
            ALTER TABLE `acmpart` DROP INDEX `fk_acmpart_acmValueId`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmvalue' and binary constraint_name = 'fk_acmvalue_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmvalue_createdBy';
            ALTER TABLE `acmvalue` DROP FOREIGN KEY `fk_acmvalue_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmvalue' and binary index_name = 'fk_acmvalue_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acmvalue_createdBy';
            ALTER TABLE `acmvalue` DROP INDEX `fk_acmvalue_createdBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmvalue' and binary constraint_name = 'fk_acmvalue_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmvalue_deletedBy';
            ALTER TABLE `acmvalue` DROP FOREIGN KEY `fk_acmvalue_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmvalue' and binary index_name = 'fk_acmvalue_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acmvalue_deletedBy';
            ALTER TABLE `acmvalue` DROP INDEX `fk_acmvalue_deletedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmvalue' and binary constraint_name = 'fk_acmvalue_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmvalue_modifiedBy';
            ALTER TABLE `acmvalue` DROP FOREIGN KEY `fk_acmvalue_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmvalue' and binary index_name = 'fk_acmvalue_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acmvalue_modifiedBy';
            ALTER TABLE `acmvalue` DROP INDEX `fk_acmvalue_modifiedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_createdBy';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_createdBy';
            ALTER TABLE `object` DROP INDEX `fk_object_createdBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_deletedBy';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_deletedBy';
            ALTER TABLE `object` DROP INDEX `fk_object_deletedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_expungedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_expungedBy';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_expungedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_expungedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_expungedBy';
            ALTER TABLE `object` DROP INDEX `fk_object_expungedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_modifiedBy';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_modifiedBy';
            ALTER TABLE `object` DROP INDEX `fk_object_modifiedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_ownedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_ownedBy';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_ownedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_ownedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_ownedBy';
            ALTER TABLE `object` DROP INDEX `fk_object_ownedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_ownedByNew') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_ownedByNew';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_ownedByNew`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_ownedByNew') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_ownedByNew';
            ALTER TABLE `object` DROP INDEX `fk_object_ownedByNew`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_parentId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_parentId';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_parentId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_parentId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_parentId';
            ALTER TABLE `object` DROP INDEX `fk_object_parentId`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_typeId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_typeId';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_typeId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_typeId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_typeId';
            ALTER TABLE `object` DROP INDEX `fk_object_typeId`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'objectacm' and binary constraint_name = 'fk_objectacm_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_objectacm_createdBy';
            ALTER TABLE `objectacm` DROP FOREIGN KEY `fk_objectacm_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'objectacm' and binary index_name = 'fk_objectacm_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_objectacm_createdBy';
            ALTER TABLE `objectacm` DROP INDEX `fk_objectacm_createdBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'objectacm' and binary constraint_name = 'fk_objectacm_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_objectacm_deletedBy';
            ALTER TABLE `objectacm` DROP FOREIGN KEY `fk_objectacm_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'objectacm' and binary index_name = 'fk_objectacm_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_objectacm_deletedBy';
            ALTER TABLE `objectacm` DROP INDEX `fk_objectacm_deletedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'objectacm' and binary constraint_name = 'fk_objectacm_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_objectacm_modifiedBy';
            ALTER TABLE `objectacm` DROP FOREIGN KEY `fk_objectacm_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'objectacm' and binary index_name = 'fk_objectacm_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_objectacm_modifiedBy';
            ALTER TABLE `objectacm` DROP INDEX `fk_objectacm_modifiedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'objectacm' and binary constraint_name = 'fk_objectacm_objectId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_objectacm_objectId';
            ALTER TABLE `objectacm` DROP FOREIGN KEY `fk_objectacm_objectId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'objectacm' and binary index_name = 'fk_objectacm_objectId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_objectacm_objectId';
            ALTER TABLE `objectacm` DROP INDEX `fk_objectacm_objectId`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'objectacm' and binary constraint_name = 'fk_objectacm_acmId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_objectacm_acmId';
            ALTER TABLE `objectacm` DROP FOREIGN KEY `fk_objectacm_acmId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'objectacm' and binary index_name = 'fk_objectacm_acmId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_objectacm_acmId';
            ALTER TABLE `objectacm` DROP INDEX `fk_objectacm_acmId`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_createdBy';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_createdBy';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_createdBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_deletedBy';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_deletedBy';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_deletedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_grantee') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_grantee';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_grantee`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_grantee') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_grantee';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_grantee`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_modifiedBy';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_modifiedBy';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_modifiedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_objectId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_objectId';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_objectId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_objectId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_objectId';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_objectId`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_property' and binary constraint_name = 'fk_object_property_objectId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_property_objectId';
            ALTER TABLE `object_property` DROP FOREIGN KEY `fk_object_property_objectId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_property' and binary index_name = 'fk_object_property_objectId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_property_objectId';
            ALTER TABLE `object_property` DROP INDEX `fk_object_property_objectId`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_property' and binary constraint_name = 'fk_object_property_propertyId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_property_propertyId';
            ALTER TABLE `object_property` DROP FOREIGN KEY `fk_object_property_propertyId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_property' and binary index_name = 'fk_object_property_propertyId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_property_propertyId';
            ALTER TABLE `object_property` DROP INDEX `fk_object_property_propertyId`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_tag' and binary constraint_name = 'fk_object_tag_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_tag_createdBy';
            ALTER TABLE `object_tag` DROP FOREIGN KEY `fk_object_tag_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_tag' and binary index_name = 'fk_object_tag_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_tag_createdBy';
            ALTER TABLE `object_tag` DROP INDEX `fk_object_tag_createdBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_tag' and binary constraint_name = 'fk_object_tag_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_tag_deletedBy';
            ALTER TABLE `object_tag` DROP FOREIGN KEY `fk_object_tag_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_tag' and binary index_name = 'fk_object_tag_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_tag_deletedBy';
            ALTER TABLE `object_tag` DROP INDEX `fk_object_tag_deletedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_tag' and binary constraint_name = 'fk_object_tag_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_tag_modifiedBy';
            ALTER TABLE `object_tag` DROP FOREIGN KEY `fk_object_tag_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_tag' and binary index_name = 'fk_object_tag_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_tag_modifiedBy';
            ALTER TABLE `object_tag` DROP INDEX `fk_object_tag_modifiedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_tag' and binary constraint_name = 'fk_object_tag_objectId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_tag_objectId';
            ALTER TABLE `object_tag` DROP FOREIGN KEY `fk_object_tag_objectId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_tag' and binary index_name = 'fk_object_tag_objectId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_tag_objectId';
            ALTER TABLE `object_tag` DROP INDEX `fk_object_tag_objectId`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_type' and binary constraint_name = 'fk_object_type_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_type_createdBy';
            ALTER TABLE `object_type` DROP FOREIGN KEY `fk_object_type_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_type' and binary index_name = 'fk_object_type_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_type_createdBy';
            ALTER TABLE `object_type` DROP INDEX `fk_object_type_createdBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_type' and binary constraint_name = 'fk_object_type_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_type_deletedBy';
            ALTER TABLE `object_type` DROP FOREIGN KEY `fk_object_type_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_type' and binary index_name = 'fk_object_type_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_type_deletedBy';
            ALTER TABLE `object_type` DROP INDEX `fk_object_type_deletedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_type' and binary constraint_name = 'fk_object_type_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_type_modifiedBy';
            ALTER TABLE `object_type` DROP FOREIGN KEY `fk_object_type_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_type' and binary index_name = 'fk_object_type_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_type_modifiedBy';
            ALTER TABLE `object_type` DROP INDEX `fk_object_type_modifiedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_type' and binary constraint_name = 'fk_object_type_ownedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_type_ownedBy';
            ALTER TABLE `object_type` DROP FOREIGN KEY `fk_object_type_ownedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_type' and binary index_name = 'fk_object_type_ownedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_type_ownedBy';
            ALTER TABLE `object_type` DROP INDEX `fk_object_type_ownedBy`;
        END IF;    
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_type_property' and binary constraint_name = 'fk_object_type_property_propertyId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_type_property_propertyId';
            ALTER TABLE `object_type_property` DROP FOREIGN KEY `fk_object_type_property_propertyId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_type_property' and binary index_name = 'fk_object_type_property_propertyId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_type_property_propertyId';
            ALTER TABLE `object_type_property` DROP INDEX `fk_object_type_property_propertyId`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_type_property' and binary constraint_name = 'fk_object_type_property_typeId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_type_property_typeId';
            ALTER TABLE `object_type_property` DROP FOREIGN KEY `fk_object_type_property_typeId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_type_property' and binary index_name = 'fk_object_type_property_typeId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_type_property_typeId';
            ALTER TABLE `object_type_property` DROP INDEX `fk_object_type_property_typeId`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'property' and binary constraint_name = 'fk_property_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_property_createdBy';
            ALTER TABLE `property` DROP FOREIGN KEY `fk_property_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'property' and binary index_name = 'fk_property_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_property_createdBy';
            ALTER TABLE `property` DROP INDEX `fk_property_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'property' and binary constraint_name = 'fk_property_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_property_deletedBy';
            ALTER TABLE `property` DROP FOREIGN KEY `fk_property_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'property' and binary index_name = 'fk_property_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_property_deletedBy';
            ALTER TABLE `property` DROP INDEX `fk_property_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'property' and binary constraint_name = 'fk_property_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_property_modifiedBy';
            ALTER TABLE `property` DROP FOREIGN KEY `fk_property_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'property' and binary index_name = 'fk_property_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_property_modifiedBy';
            ALTER TABLE `property` DROP INDEX `fk_property_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'relationship' and binary constraint_name = 'fk_relationship_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_relationship_createdBy';
            ALTER TABLE `relationship` DROP FOREIGN KEY `fk_relationship_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'relationship' and binary index_name = 'fk_relationship_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_relationship_createdBy';
            ALTER TABLE `relationship` DROP INDEX `fk_relationship_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'relationship' and binary constraint_name = 'fk_relationship_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_relationship_deletedBy';
            ALTER TABLE `relationship` DROP FOREIGN KEY `fk_relationship_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'relationship' and binary index_name = 'fk_relationship_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_relationship_deletedBy';
            ALTER TABLE `relationship` DROP INDEX `fk_relationship_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'relationship' and binary constraint_name = 'fk_relationship_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_relationship_modifiedBy';
            ALTER TABLE `relationship` DROP FOREIGN KEY `fk_relationship_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'relationship' and binary index_name = 'fk_relationship_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_relationship_modifiedBy';
            ALTER TABLE `relationship` DROP INDEX `fk_relationship_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'relationship' and binary constraint_name = 'fk_relationship_sourceId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_relationship_sourceId';
            ALTER TABLE `relationship` DROP FOREIGN KEY `fk_relationship_sourceId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'relationship' and binary index_name = 'fk_relationship_sourceId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_relationship_sourceId';
            ALTER TABLE `relationship` DROP INDEX `fk_relationship_sourceId`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'relationship' and binary constraint_name = 'fk_relationship_targetId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_relationship_targetId';
            ALTER TABLE `relationship` DROP FOREIGN KEY `fk_relationship_targetId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'relationship' and binary index_name = 'fk_relationship_targetId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_relationship_targetId';
            ALTER TABLE `relationship` DROP INDEX `fk_relationship_targetId`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'user_object_favorite' and binary constraint_name = 'fk_user_object_favorite_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_user_object_favorite_createdBy';
            ALTER TABLE `user_object_favorite` DROP FOREIGN KEY `fk_user_object_favorite_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'user_object_favorite' and binary index_name = 'fk_user_object_favorite_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_user_object_favorite_createdBy';
            ALTER TABLE `user_object_favorite` DROP INDEX `fk_user_object_favorite_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'user_object_favorite' and binary constraint_name = 'fk_user_object_favorite_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_user_object_favorite_deletedBy';
            ALTER TABLE `user_object_favorite` DROP FOREIGN KEY `fk_user_object_favorite_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'user_object_favorite' and binary index_name = 'fk_user_object_favorite_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_user_object_favorite_deletedBy';
            ALTER TABLE `user_object_favorite` DROP INDEX `fk_user_object_favorite_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'user_object_favorite' and binary constraint_name = 'fk_user_object_favorite_objectId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_user_object_favorite_objectId';
            ALTER TABLE `user_object_favorite` DROP FOREIGN KEY `fk_user_object_favorite_objectId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'user_object_favorite' and binary index_name = 'fk_user_object_favorite_objectId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_user_object_favorite_objectId';
            ALTER TABLE `user_object_favorite` DROP INDEX `fk_user_object_favorite_objectId`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'user_object_subscription' and binary constraint_name = 'fk_user_object_subscription_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_user_object_subscription_createdBy';
            ALTER TABLE `user_object_subscription` DROP FOREIGN KEY `fk_user_object_subscription_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'user_object_subscription' and binary index_name = 'fk_user_object_subscription_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_user_object_subscription_createdBy';
            ALTER TABLE `user_object_subscription` DROP INDEX `fk_user_object_subscription_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'user_object_subscription' and binary constraint_name = 'fk_user_object_subscription_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_user_object_subscription_deletedBy';
            ALTER TABLE `user_object_subscription` DROP FOREIGN KEY `fk_user_object_subscription_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'user_object_subscription' and binary index_name = 'fk_user_object_subscription_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_user_object_subscription_deletedBy';
            ALTER TABLE `user_object_subscription` DROP INDEX `fk_user_object_subscription_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'user_object_subscription' and binary constraint_name = 'fk_user_object_subscription_objectId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_user_object_subscription_objectId';
            ALTER TABLE `user_object_subscription` DROP FOREIGN KEY `fk_user_object_subscription_objectId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'user_object_subscription' and binary index_name = 'fk_user_object_subscription_objectId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_user_object_subscription_objectId';
            ALTER TABLE `user_object_subscription` DROP INDEX `fk_user_object_subscription_objectId`;
        END IF;
        -- derived from 2_idx_object_permission.sql
        INSERT INTO migration_status SET description = '20170630_combined drop foreign keys and indexes from 2_idx_object_permission.sql';
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_createdBy';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_createdBy';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'ix_object_permission_createdby') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop ix_object_permission_createdby';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `ix_object_permission_createdby`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'ix_object_permission_createdby') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_object_permission_createdby';
            ALTER TABLE `object_permission` DROP INDEX `ix_object_permission_createdby`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'ix_object_permission_allowread') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop ix_object_permission_allowread';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `ix_object_permission_allowread`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'ix_object_permission_allowread') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_object_permission_allowread';
            ALTER TABLE `object_permission` DROP INDEX `ix_object_permission_allowread`;
        END IF;
        -- derived from 3_ownedby_fk_and_triggers.sql
        INSERT INTO migration_status SET description = '20170630_combined drop foreign keys and indexes from 3_ownedby_fk_and_triggers.sql';
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_ownedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_ownedBy';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_ownedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_ownedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_ownedBy';
            ALTER TABLE `object` DROP INDEX `fk_object_ownedBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_ownedByNew') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_ownedByNew';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_ownedByNew`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_ownedByNew') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_ownedByNew';
            ALTER TABLE `object` DROP INDEX `fk_object_ownedByNew`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'ix_ownedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop ix_ownedBy';
            ALTER TABLE `object` DROP FOREIGN KEY `ix_ownedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_ownedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_ownedBy';
            ALTER TABLE `object` DROP INDEX `ix_ownedBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'a_object' and binary constraint_name = 'ix_ownedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop ix_ownedBy';
            ALTER TABLE `a_object` DROP FOREIGN KEY `ix_ownedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'a_object' and binary index_name = 'ix_ownedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_ownedBy';
            ALTER TABLE `a_object` DROP INDEX `ix_ownedBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_type' and binary constraint_name = 'fk_object_type_ownedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_type_ownedBy';
            ALTER TABLE `object_type` DROP FOREIGN KEY `fk_object_type_ownedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_type' and binary index_name = 'fk_object_type_ownedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_type_ownedBy';
            ALTER TABLE `object_type` DROP INDEX `fk_object_type_ownedBy`;
        END IF;
        -- derived from 20170331_409_ao_acm_performance.sql
        INSERT INTO migration_status SET description = '20170630_combined drop foreign keys and indexes from 20170331_409_ao_acm_performance.sql';
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmkey2' and binary constraint_name = 'ix_acmvalue2_name') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop ix_acmvalue2_name from acmkey2';
            ALTER TABLE `acmkey2` DROP FOREIGN KEY `ix_acmvalue2_name`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmkey2' and binary index_name = 'ix_acmvalue2_name') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_acmvalue2_name from acmkey2';
            ALTER TABLE `acmkey2` DROP INDEX `ix_acmvalue2_name`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmvalue2' and binary constraint_name = 'ix_acmvalue2_name') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop ix_acmvalue2_name';
            ALTER TABLE `acmvalue2` DROP FOREIGN KEY `ix_acmvalue2_name`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmvalue2' and binary index_name = 'ix_acmvalue2_name') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_acmvalue2_name';
            ALTER TABLE `acmvalue2` DROP INDEX `ix_acmvalue2_name`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmpart2' and binary constraint_name = 'fk_acmpart2_acmid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmpart2_acmid';
            ALTER TABLE `acmpart2` DROP FOREIGN KEY `fk_acmpart2_acmid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmpart2' and binary index_name = 'fk_acmpart2_acmid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acmpart2_acmid';
            ALTER TABLE `acmpart2` DROP INDEX `fk_acmpart2_acmid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmpart2' and binary constraint_name = 'fk_acmpart2_acmkeyid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmpart2_acmkeyid';
            ALTER TABLE `acmpart2` DROP FOREIGN KEY `fk_acmpart2_acmkeyid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmpart2' and binary index_name = 'fk_acmpart2_acmkeyid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acmpart2_acmkeyid';
            ALTER TABLE `acmpart2` DROP INDEX `fk_acmpart2_acmkeyid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'acmpart2' and binary constraint_name = 'fk_acmpart2_acmvalueid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_acmpart2_acmvalueid';
            ALTER TABLE `acmpart2` DROP FOREIGN KEY `fk_acmpart2_acmvalueid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'acmpart2' and binary index_name = 'fk_acmpart2_acmvalueid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_acmpart2_acmvalueid';
            ALTER TABLE `acmpart2` DROP INDEX `fk_acmpart2_acmvalueid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'useraocachepart' and binary constraint_name = 'fk_useraocachepart_userid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_useraocachepart_userid';
            ALTER TABLE `useraocachepart` DROP FOREIGN KEY `fk_useraocachepart_userid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'useraocachepart' and binary index_name = 'fk_useraocachepart_userid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_useraocachepart_userid';
            ALTER TABLE `useraocachepart` DROP INDEX `fk_useraocachepart_userid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'useraocachepart' and binary constraint_name = 'fk_useraocachepart_userkeyid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_useraocachepart_userkeyid';
            ALTER TABLE `useraocachepart` DROP FOREIGN KEY `fk_useraocachepart_userkeyid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'useraocachepart' and binary index_name = 'fk_useraocachepart_userkeyid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_useraocachepart_userkeyid';
            ALTER TABLE `useraocachepart` DROP INDEX `fk_useraocachepart_userkeyid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'useraocachepart' and binary constraint_name = 'fk_useraocachepart_uservalueid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_useraocachepart_uservalueid';
            ALTER TABLE `useraocachepart` DROP FOREIGN KEY `fk_useraocachepart_uservalueid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'useraocachepart' and binary index_name = 'fk_useraocachepart_uservalueid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_useraocachepart_uservalueid';
            ALTER TABLE `useraocachepart` DROP INDEX `fk_useraocachepart_uservalueid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'useraocache' and binary constraint_name = 'fk_useraocache_userid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_useraocachepart_userid';
            ALTER TABLE `useraocache` DROP FOREIGN KEY `fk_useraocache_userid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'useraocache' and binary index_name = 'fk_useraocache_userid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_useraocachepart_userid';
            ALTER TABLE `useraocache` DROP INDEX `fk_useraocache_userid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'useracm' and binary constraint_name = 'fk_useracm_userid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_useracm_userid';
            ALTER TABLE `useracm` DROP FOREIGN KEY `fk_useracm_userid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'useracm' and binary index_name = 'fk_useracm_userid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_useracm_userid';
            ALTER TABLE `useracm` DROP INDEX `fk_useracm_userid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'useracm' and binary constraint_name = 'fk_useracm_acmid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_useracm_acmid';
            ALTER TABLE `useracm` DROP FOREIGN KEY `fk_useracm_acmid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'useracm' and binary index_name = 'fk_useracm_acmid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_useracm_acmid';
            ALTER TABLE `useracm` DROP INDEX `fk_useracm_acmid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_acmid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_acmid';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_acmid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_acmid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_acmid';
            ALTER TABLE `object` DROP INDEX `fk_object_acmid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_ownedbyid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_ownedbyid';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_ownedbyid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_ownedbyid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_ownedbyid';
            ALTER TABLE `object` DROP INDEX `fk_object_ownedbyid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_grantee') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_grantee';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_grantee`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_grantee') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_grantee';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_grantee`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'ix_grantee') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop ix_grantee';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `ix_grantee`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'ix_grantee') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_grantee';
            ALTER TABLE `object_permission` DROP INDEX `ix_grantee`;
        END IF;        
        -- derived from 20170508_409_permissiongrantee.sql
        INSERT INTO migration_status SET description = '20170630_combined drop foreign keys and indexes from 20170508_409_permissiongrantee.sql';
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_createdbyid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_createdbyid';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_createdbyid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_createdbyid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_createdbyid';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_createdbyid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_granteeid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_granteeid';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_granteeid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_granteeid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_granteeid';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_granteeid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_deletedBy';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_deletedBy';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_modifiedBy';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_modifiedBy';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_granteeid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_granteeid';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_granteeid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_granteeid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_granteeid';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_granteeid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_createdBy';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_createdBy';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'ix_object_permission_createdby') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_object_permission_createdby';
            ALTER TABLE `object_permission` DROP INDEX `ix_object_permission_createdby`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_createdbyid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_createdbyid';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_createdbyid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_createdbyid') THEN	
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_createdbyid';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_createdbyid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_grantee') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_grantee';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_grantee`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'ix_grantee') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_grantee';
            ALTER TABLE `object_permission` DROP INDEX `ix_grantee`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_grantee') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_grantee';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_grantee`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_objectId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_objectId';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_objectId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_objectId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_objectId';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_objectId`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_objectid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_permission_objectid';
            ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_objectid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_objectid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_permission_objectid';
            ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_objectid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'ix_objectId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_objectId';
            ALTER TABLE `object_permission` DROP INDEX `ix_objectId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'ix_isDeleted') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_isDeleted';
            ALTER TABLE `object_permission` DROP INDEX `ix_isDeleted`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and index_name like 'ix_object_permission_allowread') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_object_permission_allowread';
            ALTER TABLE `object_permission` DROP INDEX `ix_object_permission_allowread`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_acmid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_acmid';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_acmid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_acmid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_acmid';
            ALTER TABLE `object` DROP INDEX `fk_object_acmid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_createdBy';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_createdBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_createdBy';
            ALTER TABLE `object` DROP INDEX `fk_object_createdBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_deletedBy';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_deletedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_deletedBy';
            ALTER TABLE `object` DROP INDEX `fk_object_deletedBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_expungedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_expungedBy';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_expungedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_expungedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_expungedBy';
            ALTER TABLE `object` DROP INDEX `fk_object_expungedBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_modifiedBy';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_modifiedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_modifiedBy';
            ALTER TABLE `object` DROP INDEX `fk_object_modifiedBy`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_ownedbyid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_ownedbyid';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_ownedbyid`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_ownedbyid') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_ownedbyid';
            ALTER TABLE `object` DROP INDEX `fk_object_ownedbyid`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_parentId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_parentId';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_parentId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_parentId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_parentId';
            ALTER TABLE `object` DROP INDEX `fk_object_parentId`;
        END IF;
        IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_typeId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop fk_object_typeId';
            ALTER TABLE `object` DROP FOREIGN KEY `fk_object_typeId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_typeId') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_typeId';
            ALTER TABLE `object` DROP INDEX `fk_object_typeId`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_createdDate') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_createdDate';
            ALTER TABLE `object` DROP INDEX `ix_createdDate`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_modifiedDate') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_modifiedDate';
            ALTER TABLE `object` DROP INDEX `ix_modifiedDate`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_ownedBy') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_ownedBy';
            ALTER TABLE `object` DROP INDEX `ix_ownedBy`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'idx_object_description') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index idx_object_description';
            ALTER TABLE `object` DROP INDEX `idx_object_description`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_ownedByNew') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index fk_object_ownedByNew';
            ALTER TABLE `object` DROP INDEX `fk_object_ownedByNew`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_name') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_name';
            ALTER TABLE `object` DROP INDEX `ix_name`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_isDeleted') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_isDeleted';
            ALTER TABLE `object` DROP INDEX `ix_isDeleted`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_object_createdDate') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_object_createdDate';
            ALTER TABLE `object` DROP INDEX `ix_object_createdDate`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_object_modifiedDate') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_object_modifiedDate';
            ALTER TABLE `object` DROP INDEX `ix_object_modifiedDate`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_object_name') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_object_name';
            ALTER TABLE `object` DROP INDEX `ix_object_name`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_object_isdeleted') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_object_isdeleted';
            ALTER TABLE `object` DROP INDEX `ix_object_isdeleted`;
        END IF;
        IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_object_description') THEN
            INSERT INTO migration_status SET description = '20170630_combined drop index ix_object_description';
            ALTER TABLE `object` DROP INDEX `ix_object_description`;
        END IF;
    end proc_main;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170630_drop_keys_and_indexes_raw;    

-- Remaining tables from 20170331_409_ao_acm_performance.sql, without foreign keys and indexes
INSERT INTO migration_status SET description = '20170630_combined creating tables for ao acm performance';
CREATE TABLE IF NOT EXISTS acm2
(
    id int unsigned not null auto_increment
    ,sha256hash char(64) not null
    ,flattenedacm text not null
    ,CONSTRAINT pk_acm2 PRIMARY KEY (id)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;
CREATE TABLE IF NOT EXISTS acmkey2
(
    id int unsigned not null auto_increment
    ,name varchar(255) not null
    ,CONSTRAINT pk_acmkey2 PRIMARY KEY (id)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;
CREATE TABLE IF NOT EXISTS acmvalue2
(
    id int unsigned not null auto_increment
    ,name varchar(255) not null
    ,CONSTRAINT pk_acmvalue2 PRIMARY KEY (id)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;
CREATE TABLE IF NOT EXISTS acmpart2
(
    id int unsigned not null auto_increment
    ,acmid int unsigned not null
    ,acmkeyid int unsigned not null
    ,acmvalueid int unsigned null
    ,CONSTRAINT pk_acmpart2 PRIMARY KEY (id)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;
CREATE TABLE IF NOT EXISTS useraocachepart
(
    id int unsigned not null auto_increment
    ,userid binary(16) not null
    ,isAllowed tinyint not null default 0
    ,userkeyid int unsigned not null
    ,uservalueid int unsigned null
    ,CONSTRAINT pk_useraocachepart PRIMARY KEY (id)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;
CREATE TABLE IF NOT EXISTS useraocache
(
    id int unsigned not null auto_increment
    ,userid binary(16) not null
    ,isCaching tinyint not null default 1
    ,cacheDate timestamp(6) not null
    ,sha256hash char(64) not null
    ,CONSTRAINT pk_useraocache PRIMARY KEY (id)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;
CREATE TABLE IF NOT EXISTS useracm
(
    id int unsigned not null auto_increment
    ,userid binary(16) not null
    ,acmid int unsigned not null
    ,CONSTRAINT pk_useracm PRIMARY KEY (id)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;

-- Add or remove columns for tables from 20170331_409_ao_acm_performance.sql
DROP PROCEDURE IF EXISTS sp_Patch_20170630_tablecolumns;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170630_tablecolumns()
BEGIN
    DECLARE has_a_object_acmid int default false;
    DECLARE has_a_object_ispdfavailable int default false;
    DECLARE has_a_object_isstreamstored int default false;
    DECLARE has_a_object_ownedbyid int default false;
    DECLARE has_a_object_ownedbynew int default false;
    DECLARE has_a_object_permission_createdbyid int default false;
    DECLARE has_a_object_permission_granteeid int default false;
    DECLARE has_object_acmid int default false;
    DECLARE has_object_ispdfavailable int default false;
    DECLARE has_object_isstreamstored int default false;
    DECLARE has_object_ownedbyid int default false;
    DECLARE has_object_ownedbynew int default false;
    DECLARE has_object_permission_createdbyid int default false;
    DECLARE has_object_permission_granteeid int default false;

    -- a_object table
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'a_object' and column_name = 'acmid') THEN
        set has_a_object_acmid := true;
    END IF;
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'a_object' and column_name = 'ispdfavailable') THEN
        set has_a_object_ispdfavailable := true;
    END IF;
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'a_object' and column_name = 'isstreamstored') THEN
        set has_a_object_isstreamstored := true;
    END IF;    
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'a_object' and column_name = 'ownedbyid') THEN
        set has_a_object_ownedbyid := true;
    END IF;
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'a_object' and column_name = 'ownedbynew') THEN
        set has_a_object_ownedbynew := true;
    END IF;
    IF ((NOT has_a_object_acmid) OR (NOT has_a_object_ownedbyid)) THEN
        INSERT INTO migration_status SET description = '20170630_combined adding columns to a_object';
        IF ((NOT has_a_object_acmid) AND (NOT has_a_object_ownedbyid)) THEN
            ALTER TABLE a_object ADD COLUMN acmid int unsigned null, ADD COLUMN ownedbyid int unsigned null;
            set has_a_object_acmid := true;
            set has_a_object_ownedbyid := true;
        END IF;
        IF (NOT has_a_object_acmid) THEN
            ALTER TABLE a_object ADD COLUMN acmid int unsigned null;
            set has_a_object_acmid := true;
        END IF;
        IF (NOT has_a_object_ownedbyid) THEN
            ALTER TABLE a_object ADD COLUMN ownedbyid int unsigned null;
            set has_a_object_ownedbyid := true;
        END IF;
    END IF;
    IF ((has_a_object_ispdfavailable) OR (has_a_object_isstreamstored) OR (has_a_object_ownedbynew)) THEN
        INSERT INTO migration_status SET description = '20170630_combined removing columns from a_object';
        IF ((has_a_object_ispdfavailable) AND (has_a_object_isstreamstored) AND (has_a_object_ownedbynew)) THEN
            ALTER TABLE a_object DROP COLUMN ispdfavailable, DROP COLUMN isstreamstored, DROP COLUMN ownedbynew;
            set has_a_object_ispdfavailable := false;
            set has_a_object_isstreamstored := false;
            set has_a_object_ownedbynew := false;
        END IF;
        IF has_a_object_ispdfavailable THEN
            ALTER TABLE a_object DROP COLUMN ispdfavailable;
            set has_a_object_ispdfavailable := false;
        END IF;
        IF has_a_object_isstreamstored THEN
            ALTER TABLE a_object DROP COLUMN isstreamstored;
            set has_a_object_isstreamstored := false;
        END IF;
        IF has_a_object_ownedbynew THEN
            ALTER TABLE a_object DROP COLUMN ownedbynew;
            set has_a_object_ownedbynew := false;
        END IF;
    END IF;

    -- a_object_permission table
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'a_object_permission' and column_name = 'createdbyid') THEN
        set has_a_object_permission_createdbyid := true;
    END IF;
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'a_object_permission' and column_name = 'granteeid') THEN
        set has_a_object_permission_granteeid := true;
    END IF;
    IF ((NOT has_a_object_permission_createdbyid) OR (NOT has_a_object_permission_granteeid)) THEN
        INSERT INTO migration_status SET description = '20170630_combined adding columns to a_object_permission';
        IF ((NOT has_a_object_permission_createdbyid) AND (NOT has_a_object_permission_granteeid)) THEN
            ALTER TABLE a_object_permission ADD COLUMN createdbyid int unsigned null, ADD COLUMN granteeid int unsigned null;
            set has_a_object_permission_createdbyid := true;
            set has_a_object_permission_granteeid := true;
        END IF;
        IF (NOT has_a_object_permission_createdbyid) THEN
            ALTER TABLE a_object_permission ADD COLUMN createdbyid int unsigned null;
            set has_a_object_permission_createdbyid := true;
        END IF;
        IF (NOT has_a_object_permission_granteeid) THEN
            ALTER TABLE a_object_permission ADD COLUMN granteeid int unsigned null;
            set has_a_object_permission_granteeid := true;
        END IF;
    END IF;    

    -- object table
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'object' and column_name = 'acmid') THEN
        set has_object_acmid := true;
    END IF;
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'object' and column_name = 'ispdfavailable') THEN
        set has_object_ispdfavailable := true;
    END IF;
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'object' and column_name = 'isstreamstored') THEN
        set has_object_isstreamstored := true;
    END IF;    
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'object' and column_name = 'ownedbyid') THEN
        set has_object_ownedbyid := true;
    END IF;
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'object' and column_name = 'ownedbynew') THEN
        set has_object_ownedbynew := true;
    END IF;    
    IF ((NOT has_object_acmid) OR (NOT has_object_ownedbyid)) THEN
        INSERT INTO migration_status SET description = '20170630_combined adding columns to object';
        IF ((NOT has_object_acmid) AND (NOT has_object_ownedbyid)) THEN
            ALTER TABLE object ADD COLUMN acmid int unsigned null, ADD COLUMN ownedbyid int unsigned null;
            set has_object_acmid := true;
            set has_object_ownedbyid := true;
        END IF;
        IF (NOT has_object_acmid) THEN
            ALTER TABLE object ADD COLUMN acmid int unsigned null;
            set has_object_acmid := true;
        END IF;
        IF (NOT has_object_ownedbyid) THEN
            ALTER TABLE object ADD COLUMN ownedbyid int unsigned null;
            set has_object_ownedbyid := true;
        END IF;
    END IF;
    IF ((has_object_ispdfavailable) OR (has_object_isstreamstored) OR (has_object_ownedbynew)) THEN
        INSERT INTO migration_status SET description = '20170630_combined removing columns from object';
        IF ((has_object_ispdfavailable) AND (has_object_isstreamstored) AND (has_object_ownedbynew)) THEN
            ALTER TABLE object DROP COLUMN ispdfavailable, DROP COLUMN isstreamstored, DROP COLUMN ownedbynew;
            set has_object_ispdfavailable := false;
            set has_object_isstreamstored := false;
            set has_object_ownedbynew := false;
        END IF;
        IF has_object_ispdfavailable THEN
            ALTER TABLE object DROP COLUMN ispdfavailable;
            set has_object_ispdfavailable := false;
        END IF;
        IF has_object_isstreamstored THEN
            ALTER TABLE object DROP COLUMN isstreamstored;
            set has_object_isstreamstored := false;
        END IF;
        IF has_object_ownedbynew THEN
            ALTER TABLE object DROP COLUMN ownedbynew;
            set has_object_ownedbynew := false;
        END IF;
    END IF;

    -- object_permission table
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'object_permission' and column_name = 'createdbyid') THEN
        set has_object_permission_createdbyid := true;
    END IF;
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'object_permission' and column_name = 'granteeid') THEN
        set has_object_permission_granteeid := true;
    END IF;
    IF ((NOT has_object_permission_createdbyid) OR (NOT has_object_permission_granteeid)) THEN
        INSERT INTO migration_status SET description = '20170630_combined adding columns to object_permission';
        IF ((NOT has_object_permission_createdbyid) AND (NOT has_object_permission_granteeid)) THEN
            ALTER TABLE object_permission ADD COLUMN createdbyid int unsigned null, ADD COLUMN granteeid int unsigned null;
            set has_object_permission_createdbyid := true;
            set has_object_permission_granteeid := true;
        END IF;
        IF (NOT has_object_permission_createdbyid) THEN
            ALTER TABLE object_permission ADD COLUMN createdbyid int unsigned null;
            set has_object_permission_createdbyid := true;
        END IF;
        IF (NOT has_object_permission_granteeid) THEN
            ALTER TABLE object_permission ADD COLUMN granteeid int unsigned null;
            set has_object_permission_granteeid := true;
        END IF;
    END IF;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170630_tablecolumns;

-- functions needed created or replaced
INSERT INTO migration_status SET description = '20170630_combined recreating function aacflatten';
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
INSERT INTO migration_status SET description = '20170630_combined recreating function calcresourcestring';
DROP FUNCTION IF EXISTS calcResourceString;
-- +migrate StatementBegin
CREATE FUNCTION calcResourceString(vOriginalString varchar(300)) RETURNS varchar(300)
BEGIN
    DECLARE vParts int default 0;
    DECLARE vPart1 varchar(255) default ''; -- type
    DECLARE vPart2 varchar(255) default ''; -- user dn, short group name, project name
    DECLARE vPart3 varchar(255) default ''; -- user display name, short group display name, full project display name
    DECLARE vPart4 varchar(255) default ''; -- full group name
    DECLARE vPart5 varchar(255) default ''; -- full display name
    DECLARE vLowerString varchar(300) default '';
    DECLARE vResourceString varchar(300) default '';
    DECLARE vGrantee varchar(255) default '';
    DECLARE vDisplayName varchar(255) default '';

    -- As of 2017-05-05 Always forced to lowercase
    SET vLowerString := LOWER(vOriginalString);
    SELECT (length(vLowerString)-length(replace(vLowerString,'/',''))) + 1 INTO vParts;
    SELECT substring_index(vLowerString,'/',1) INTO vPart1;
    SELECT substring_index(substring_index(vLowerString,'/',2),'/',-1) INTO vPart2;
    SELECT substring_index(substring_index(vLowerString,'/',3),'/',-1) INTO vPart3;
    SELECT substring_index(substring_index(vLowerString,'/',4),'/',-1) INTO vPart4;
    SELECT substring_index(substring_index(vLowerString,'/',5),'/',-1) INTO vPart5;

    IF vParts > 1 AND (vPart1 = 'user' or vPart1 = 'group') THEN
        -- Calculate resource string and grantee, check if exists in acmgrantee, inserting as needed
        IF vPart1 = 'user' THEN
            SET vResourceString := CONCAT(vPart1, '/', vPart2);
            SET vGrantee := aacflatten(vPart2);
            IF (SELECT 1=1 FROM acmgrantee WHERE binary resourcestring = vResourceString) IS NULL THEN
                IF (select 1=1 FROM acmgrantee WHERE binary grantee = vGrantee) IS NULL THEN
                    IF vParts > 2 THEN
                        SET vDisplayName := vPart3;
                    ELSE
                        SET vDisplayName := replace(replace(substring_index(vPart2,',',1),'cn=',''),'CN=','');
                    END IF;
                    INSERT INTO acmgrantee SET 
                        grantee = vGrantee, 
                        resourcestring = vResourceString, 
                        projectName = null, 
                        projectDisplayName = null,
                        groupName = null,
                        userDistinguishedName = vPart2,
                        displayName = vDisplayName;
                ELSE
                    SELECT resourcestring INTO vResourceString FROM acmgrantee WHERE binary grantee = vGrantee LIMIT 1;
                END IF;
            END IF;
        END IF;
        IF vPart1 = 'group' THEN
            IF vParts <= 3 THEN
                -- Pseudo group (i.e., Everyone)
                SET vResourceString := CONCAT(vPart1, '/', vPart2);
                SET vGrantee := aacflatten(vPart2);
                IF (SELECT 1=1 FROM acmgrantee WHERE binary resourcestring = vResourceString) IS NULL THEN
                    IF (select 1=1 FROM acmgrantee WHERE binary grantee = vGrantee) IS NULL THEN
                        IF vParts > 2 THEN
                            SET vDisplayName := vPart3;
                        ELSE
                            SET vDisplayName := vPart2;
                        END IF;
                        INSERT INTO acmgrantee SET 
                            grantee = vGrantee, 
                            resourcestring = vResourceString, 
                            projectName = null, 
                            projectDisplayName = null,
                            groupName = vPart2,
                            userDistinguishedName = null,
                            displayName = vDisplayName;
                    ELSE
                        SELECT resourcestring INTO vResourceString FROM acmgrantee WHERE binary grantee = vGrantee LIMIT 1;
                    END IF;
                END IF;                                
            END IF;
            IF vParts > 3 THEN
                -- Typical groups
                SET vResourceString := CONCAT(vPart1, '/', vPart2, '/', vPart3, '/', vPart4);
                SET vGrantee := aacflatten(CONCAT(vPart2,'_',vPart4));
                IF (SELECT 1=1 FROM acmgrantee WHERE binary resourcestring = vResourceString) IS NULL THEN
                    IF (select 1=1 FROM acmgrantee WHERE binary grantee = vGrantee) IS NULL THEN
                        IF vParts > 4 THEN
                            SET vDisplayName := vPart5;
                        ELSE
                            SET vDisplayName := CONCAT(vPart3, ' ', vPart4);
                        END IF;
                        INSERT INTO acmgrantee SET 
                            grantee = vGrantee, 
                            resourcestring = vResourceString, 
                            projectName = vPart2, 
                            projectDisplayName = vPart3,
                            groupName = vPart4,
                            userDistinguishedName = null,
                            displayName = vDisplayName;
                    ELSE
                        SELECT resourcestring INTO vResourceString FROM acmgrantee WHERE binary grantee = vGrantee LIMIT 1;
                    END IF;
                END IF;   
            END IF;
        END IF;
        -- See if grantee exists in acmvalue2
        IF (SELECT 1=1 FROM acmvalue2 WHERE binary name = vGrantee) IS NULL THEN
            INSERT INTO acmvalue2 SET name = vGrantee;
        END IF;        
    ELSE
        SET vResourceString := '';
    END IF;
	RETURN vResourceString;
END;
-- +migrate StatementEnd
INSERT INTO migration_status SET description = '20170630_combined recreating function calcGranteeIDFromResourceString';
DROP FUNCTION IF EXISTS calcGranteeIDFromResourceString;
-- +migrate StatementBegin
CREATE FUNCTION calcGranteeIDFromResourceString(vOriginalString varchar(300)) RETURNS int unsigned
BEGIN
    DECLARE vResourceString varchar(300) default '';
    DECLARE vGrantee varchar(255) default '';
    DECLARE vID int unsigned default 0;

    SET vResourceString := vOriginalString;
    IF (SELECT 1=1 FROM acmgrantee WHERE resourcestring = vResourceString) IS NULL THEN
        SET vResourceString := calcResourceString(vResourceString);
    END IF;
    SELECT grantee INTO vGrantee FROM acmgrantee WHERE resourceString = vResourceString LIMIT 1;
    SELECT id INTO vID FROM acmvalue2 WHERE name = vGrantee LIMIT 1;
    RETURN vID;
END;
-- +migrate StatementEnd

INSERT INTO migration_status SET description = '20170630_combined transform acmid to new indexed tables';
DROP PROCEDURE IF EXISTS sp_Patch_20170630_transform_acmid;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170630_transform_acmid()
proc_label: BEGIN
    -- only do this if not yet 20170630
    IF EXISTS( select null from dbstate where schemaversion in ('20170508','20170630')) THEN
        LEAVE proc_label;
    END IF;
    INSERT INTO migration_status SET description = '20170630_combined migrate acm to populate acm2 tables';
    ACMMIGRATE: BEGIN
        DECLARE ACMMIGRATECOUNT int default 0;
        DECLARE ACMMIGRATETOTAL int default 0;
        DECLARE vACMID int default 0;
        DECLARE vACMName text default '';
        DECLARE vSHA256Hash char(64) default '';
        DECLARE vKeyID int default 0;
        DECLARE vKeyName varchar(255) default '';
        DECLARE vValueID int default 0;
        DECLARE vValueName varchar(255) default '';
        DECLARE vPartID int default 0;
        DECLARE c_acm_finished int default 0;
        DECLARE c_acm cursor for SELECT a.name acmname, ak.name keyname, av.name valuename 
            from acm a 
            inner join acmpart ap on a.id = ap.acmid 
            inner join acmkey ak on ap.acmkeyid = ak.id 
            inner join acmvalue av on ap.acmvalueid = av.id;
        DECLARE continue handler for not found set c_acm_finished = 1;
        OPEN c_acm;
        SELECT count(0) INTO ACMMIGRATETOTAL
            from acm a 
            inner join acmpart ap on a.id = ap.acmid 
            inner join acmkey ak on ap.acmkeyid = ak.id 
            inner join acmvalue av on ap.acmvalueid = av.id;
        get_acm: LOOP
            FETCH c_acm INTO vACMName, vKeyName, vValueName;
            IF c_acm_finished = 1 THEN
                CLOSE c_acm;
                LEAVE get_acm;
            END IF;
            SET ACMMIGRATECOUNT := ACMMIGRATECOUNT + 1;
            IF floor(ACMMIGRATECOUNT/5000) = ceiling(ACMMIGRATECOUNT/5000) THEN
                INSERT INTO migration_status SET description = concat('20170630_combined.sql migrate acm to populate acm2 tables (', ACMMIGRATECOUNT, ' of ', ACMMIGRATETOTAL, ')');
            END IF;
            -- value
            IF (SELECT 1=1 FROM acmvalue2 WHERE name = vValueName) IS NULL THEN
                INSERT INTO acmvalue2 SET name = vValueName;
                SET vValueID := LAST_INSERT_ID();
            ELSE
                SELECT id INTO vValueID FROM acmvalue2 WHERE name = vValueName LIMIT 1;
            END IF;
            -- key
            IF (SELECT 1=1 FROM acmkey2 WHERE name = vKeyName) IS NULL THEN
                INSERT INTO acmkey2 (name) VALUES (vKeyName);
                SET vKeyID := LAST_INSERT_ID();
            ELSE
                SELECT id INTO vKeyID FROM acmkey2 WHERE name = vKeyName LIMIT 1;
            END IF;
            -- acm
            IF (SELECT 1=1 FROM acm2 WHERE flattenedacm = vACMName) IS NULL THEN
                INSERT INTO acm2 (sha256hash, flattenedacm) VALUES (sha2(vACMName, 256), vACMName);
                SET vACMID := LAST_INSERT_ID();
            ELSE
                SELECT id INTO vACMID FROM acm2 WHERE flattenedacm = vACMName LIMIT 1;
            END IF;
            -- part
            IF (SELECT 1=1 FROM acmpart2 WHERE acmid = vACMID and acmkeyid = vKeyID and acmvalueid = vValueID) IS NULL THEN
                INSERT INTO acmpart2 (acmid, acmkeyid, acmvalueid) VALUES (vACMID, vKeyID, vValueID);
                SET vPartID := LAST_INSERT_ID();
            ELSE
                SELECT id INTO vPartID FROM acmpart2 WHERE acmid = vACMID and acmkeyid = vKeyID and acmvalueid = vValueID LIMIT 1;
            END IF;
        END LOOP get_acm;
    END ACMMIGRATE;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170630_transform_acmid;

INSERT INTO migration_status SET description = '20170630_combined normalize acmgrantee to lowercase';
DROP PROCEDURE IF EXISTS sp_Patch_20170630_update_acmgrantee;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170630_update_acmgrantee()
proc_label: BEGIN
    -- only do this if not yet 20170630
    IF EXISTS( select null from dbstate where schemaversion = '20170630') THEN
        LEAVE proc_label;
    END IF;
    UPDATE acmgrantee SET 
        grantee = lower(grantee)
        ,resourcestring = lower(resourcestring)
        ,projectname = lower(projectname)
        ,projectdisplayname = lower(projectdisplayname)
        ,groupname = lower(groupname)
        ,userdistinguishedname = lower(userdistinguishedname)
        ,displayname = lower(displayname)
    ;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170630_update_acmgrantee;

INSERT INTO migration_status SET description = '20170630_combined fixing invalid grantee in acmgrantee';
DROP PROCEDURE IF EXISTS sp_Patch_20170630_grantee_flattening;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170630_grantee_flattening()
proc_label: BEGIN
    -- only do this if not yet 20170630
    IF EXISTS( select null from dbstate where schemaversion = '20170630') THEN
        LEAVE proc_label;
    END IF;

    /*
    Fixes this kind of scenario.. problem is on these two records in acmgrantee .. this is caused by fake data in our dao unit tests. production won't necessarily have this.
    +-----------------------------------------------------------------------------------+----------------------------------------------------------------------------------------+
    | grantee                                                                           | resourcestring                                                                         |
    +-----------------------------------------------------------------------------------+----------------------------------------------------------------------------------------+
    | CN=[DAOTEST]test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US | user/CN=[DAOTEST]test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US |
    | cndaotesttesttester01ou_s_governmentouchimeraoudaeoupeoplecus                     | user/cndaotesttesttester01ou_s_governmentouchimeraoudaeoupeoplecus                     |
        -- the first has the wrong grantee, but good values otherwise.
        -- the second has the correct grantee, and erroneous values otherwise.
        -- cant do this update acmgrantee call since it results in both records having the same value for grantee which is the primary key.
    */
    DEDUPEACMGRANTEE: BEGIN
        DECLARE vFlattenedGrantee varchar(255);
        DECLARE vGrantee varchar(255);
        DECLARE vResourceString varchar(300);
        DECLARE vPrevFlattenedGrantee varchar(255) default '';
        DECLARE c_acmgrantee_finished int default 0;        
        DECLARE c_acmgrantee cursor FOR 
            select lower(aacflatten(grantee)), grantee, resourcestring 
            from acmgrantee where lower(aacflatten(grantee)) in (
                select flattenedgrantee 
                from (
                    select lower(aacflatten(grantee)) flattenedgrantee, count(0) 
                    from acmgrantee 
                    group by lower(aacflatten(grantee)) 
                    having count(0) > 1
                ) as g
            );
        DECLARE continue handler for not found set c_acmgrantee_finished = 1;
        INSERT INTO migration_status SET description = '20170630_combined fixing invalid grantee in acmgrantee. checking for duplicate acmgrantees as identified by lower(aacflatten(grantee))';        
        OPEN c_acmgrantee;
        get_acmgrantee: LOOP
            FETCH c_acmgrantee INTO vFlattenedGrantee, vGrantee, vResourceString;
            IF c_acmgrantee_finished = 1 THEN
                CLOSE c_acmgrantee;
                LEAVE get_acmgrantee;
            END IF;
            IF vPrevFlattenedGrantee = vFlattenedGrantee THEN
                -- seen before, lets get rid of it.  We're indescriminate about its quality
                INSERT INTO migration_status SET description = concat('20170630_combined removing duplicate record in acmgrantee, grantee=', vGrantee, ', resourceString=', vResourceString);                
                DELETE FROM acmgrantee WHERE grantee = vGrantee AND resourceString = vResourceString;
            END IF;
            SET vPrevFlattenedGrantee := vFlattenedGrantee;
        END LOOP get_acmgrantee;
    END DEDUPEACMGRANTEE;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170630_grantee_flattening;

-- Updates to object, which will leverage the changes to acm. After this section is finished, migration will continue
-- and services should hopefully not end up with errors when fielding GET requests
INSERT INTO migration_status SET description = '20170630_combined update object table with ownedby, ownedbyid, acmid';
DROP PROCEDURE IF EXISTS sp_Patch_20170630_update_object_table;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170630_update_object_table()
proc_label: BEGIN
    -- only do this if not yet 20170508 or 20170630
    IF EXISTS( select null from dbstate where schemaversion in ('20170508','20170630')) THEN
        LEAVE proc_label;
    END IF;
    INSERT INTO migration_status SET description = '20170630_combined assign ownedbyid and acmid';
    proc_main: BEGIN
        DECLARE counter int default 0;
        DECLARE vObjectID binary(16) default 0;
        DECLARE vOwnedBy varchar(255) default '';
        DECLARE vOwnedByID int unsigned default 0;
        DECLARE vACMID int unsigned default 0;
        DECLARE c_object_finished int default 0;
        DECLARE c_object cursor for SELECT id FROM object WHERE ownedbyid is null or ownedbyid = 0 or acmid is null or acmid = 0;
        DECLARE continue handler for not found set c_object_finished = 1;
        OPEN c_object;
        get_object: LOOP
            FETCH c_object INTO vObjectID;
            IF c_object_finished = 1 THEN
                CLOSE c_object;
                LEAVE get_object;
            END IF;
            SET counter := counter + 1;
            IF floor(counter/5000) = ceiling(counter/5000) THEN
                INSERT INTO migration_status SET description = concat('20170630_combined assign ownedbyid and acmid (>', counter, ' objects)');
            END IF;
            REVISIONS: BEGIN
                DECLARE vAID int default 0;
                DECLARE vPrevAID int default -1;
                DECLARE vPrevOwnedBy varchar(255) default '';                
                DECLARE vChangeCount int default 0;
                DECLARE vPrevChangeCount int default -1;
                DECLARE vFlattenedACM text default '';
                DECLARE c_revision_finished int default 0;
                DECLARE c_revision cursor FOR 
                    SELECT d.id, d.ownedby, d.changecount, d.acm FROM (
                        SELECT a_id id, ownedby, changecount, modifieddate, '' acm
                        FROM a_object 
                        WHERE id = vObjectID
                        UNION ALL
                        SELECT -1 a_id, '', -1 id, oa.modifieddate, a.name acm
                        FROM a_objectacm oa INNER JOIN acm a on oa.acmid = a.id
                        WHERE oa.objectid = vObjectID
                    ) AS d ORDER BY d.modifieddate;
                DECLARE continue handler for not found set c_revision_finished = 1;
                SET vACMID := 0;
                OPEN c_revision;
                get_revision: LOOP
                    FETCH c_revision INTO vAID, vOwnedBy, vChangeCount, vFlattenedACM;
                    IF c_revision_finished = 1 THEN
                        CLOSE c_revision;
                        LEAVE get_revision;
                    END IF;
                    IF vOwnedBy <> '' THEN
                        SET vOwnedBy := calcResourceString(vOwnedBy);
                        SET vPrevOwnedBy := vOwnedBy;
                    ELSE
                        SET vOwnedBy := vPrevOwnedBy;
                    END IF;
                    SET vOwnedByID := calcGranteeIDFromResourceString(vOwnedBy);
                    -- Row represents Object Revision
                    IF vAID <> -1 AND vChangeCount <> -1 THEN
                        SET vPrevAID := vAID;
                        SET vPrevChangeCount := vChangeCount;
                        IF vACMID > 0 THEN
                            UPDATE a_object SET ownedby = vOwnedBy, ownedbyid = vOwnedByID, acmid = vACMID WHERE a_id = vPrevAID AND changecount = vPrevChangeCount;
                        END IF;
                    END IF;
                    -- Row represents ACM changed
                    IF length(vFlattenedACM) > 0 THEN
                        SELECT id INTO vACMID FROM acm2 WHERE flattenedacm = vFlattenedACM LIMIT 1;
                        UPDATE a_object SET ownedby = vOwnedBy, ownedbyid = vOwnedByID, acmid = vACMID WHERE a_id = vPrevAID AND changecount = vPrevChangeCount;
                    END IF;
                END LOOP get_revision;
            END REVISIONS;
            UPDATE object SET ownedby = vOwnedBy, ownedbyid = vOwnedByID, acmid = vACMID WHERE id = vObjectID;
        END LOOP get_object;
    END proc_main;   
END;
-- +migrate StatementEnd
CALL sp_Patch_20170630_update_object_table;

INSERT INTO migration_status SET description = '20170630_combined update permissions with createdbyid, grantee, and revised permissiomac from grantee corrections';
DROP PROCEDURE IF EXISTS sp_Patch_20170630_update_permissions;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170630_update_permissions()
proc_label: BEGIN
    -- only do this if not yet 20170508 or 20170630
    IF EXISTS( select null from dbstate where schemaversion in ('20170508','20170630')) THEN
        LEAVE proc_label;
    END IF;
    proc_main: BEGIN
        DECLARE counter int default 0;
        DECLARE counterr int default 0;
        DECLARE vID binary(16) default 0;
        DECLARE vObjectID binary(16) default 0;
        DECLARE vCreatedBy varchar(255) default '';
        DECLARE vGrantee varchar(255) default '';
        DECLARE vAllowCreate tinyint(1) default 0;
        DECLARE vAllowRead tinyint(1) default 0;
        DECLARE vAllowUpdate tinyint(1) default 0;
        DECLARE vAllowDelete tinyint(1) default 0;
        DECLARE vAllowShare tinyint(1) default 0;
        DECLARE vEncryptKey binary(32) default 0;
        DECLARE vCreatedByID int unsigned default 0;
        DECLARE vGranteeID int unsigned default 0;
        DECLARE vPermissionMAC binary(32) default 0;
        DECLARE c_permission_finished int default 0;
        DECLARE c_permission cursor for 
            SELECT id, objectid 
            FROM object_permission
            WHERE granteeid is null or granteeid = 0 or createdbyid is null or createdbyid = 0
            ;
        DECLARE continue handler for not found set c_permission_finished = 1;
        OPEN c_permission;
        get_permission: LOOP
            FETCH c_permission INTO vID, vObjectID;
            IF c_permission_finished = 1 THEN
                CLOSE c_permission;
                LEAVE get_permission;
            END IF;
            SET counter := counter + 1;
            IF floor(counter/5000) = ceiling(counter/5000) THEN
                INSERT INTO migration_status SET description = concat('20170630_combined update permissions (>', counter, ' permission records)');
            END IF;
            REVISIONS: BEGIN
                DECLARE vAID int default 0;
                DECLARE c_revision_finished int default 0;
                DECLARE c_revision cursor FOR 
                    SELECT a_id, createdby, grantee, allowcreate, allowread, allowupdate, allowdelete, allowshare, encryptkey 
                    FROM a_object_permission 
                    WHERE id = vID and objectid = vObjectID
                    ORDER BY changecount asc;
                DECLARE continue handler for not found SET c_revision_finished = 1;
                OPEN c_revision;
                get_revision: LOOP
                    FETCH c_revision INTO vAID, vCreatedBy, vGrantee, vAllowCreate, vAllowRead, vAllowUpdate, vAllowDelete, vAllowShare, vEncryptKey;
                    IF c_revision_finished = 1 THEN
                        CLOSE c_revision;
                        LEAVE get_revision;
                    END IF;
                    SET counterr := counterr + 1;
                    IF floor(counterr/5000) = ceiling(counterr/5000) THEN
                        INSERT INTO migration_status SET description = concat('20170630_combined update permissions (>', counterr, ' archive permission records)');
                    END IF;
                    SET vGrantee := lower(aacflatten(vGrantee));
                    IF (SELECT 1=1 FROM acmvalue2 WHERE name = vGrantee) IS NULL THEN
                        INSERT INTO acmvalue2 SET name = vGrantee;
                        SET vGranteeID := LAST_INSERT_ID();
                    ELSE
                        SELECT id FROM acmvalue2 WHERE name = vGrantee INTO vGranteeID;
                    END IF;
                    SET vCreatedBy := lower(aacflatten(vCreatedBy));
                    IF (SELECT 1=1 FROM acmvalue2 WHERE name = vCreatedBy) IS NULL THEN
                        INSERT INTO acmvalue2 SET name = vCreatedBy;
                        SET vCreatedByID := LAST_INSERT_ID();
                    ELSE
                        SELECT id FROM acmvalue2 WHERE name = vCreatedBy INTO vCreatedByID;
                    END IF;
                    SET vPermissionMAC := new_keymac('${OD_ENCRYPT_MASTERKEY}',vGrantee,vAllowCreate,vAllowRead,vAllowUpdate,vAllowDelete,vAllowShare,hex(vEncryptKey));
                    UPDATE a_object_permission SET grantee = vGrantee, granteeid = vGranteeID, createdbyid = vCreatedByID, permissionmac = unhex(vPermissionMAC) where a_id = vAID;
                END LOOP get_revision;
            END REVISIONS;
            UPDATE object_permission SET grantee = vGrantee, granteeid = vGranteeID, createdbyid = vCreatedByID, permissionmac = unhex(vPermissionMAC) WHERE id = vID; 
        END LOOP get_permission;
    END proc_main;   
END;
-- +migrate StatementEnd
CALL sp_Patch_20170630_update_permissions;
DROP PROCEDURE IF EXISTS sp_Patch_20170630_update_permissions; 

-- Drop tables no longer needed per issue #666
INSERT INTO migration_status SET description = '20170630_combined drop tables no longer needed';
DROP PROCEDURE IF EXISTS sp_Patch_20170630_drop_tables;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170630_drop_tables()
proc_label: BEGIN
    -- only do this if not yet 20170630
    IF EXISTS( select null from dbstate where schemaversion = '20170630') THEN
        LEAVE proc_label;
    END IF;
    DROP TABLE IF EXISTS a_acm;
    DROP TABLE IF EXISTS a_acmkey;
    DROP TABLE IF EXISTS a_acmpart;
    DROP TABLE IF EXISTS a_acmvalue;
    DROP TABLE IF EXISTS a_object_tag;
    DROP TABLE IF EXISTS a_objectacm;
    DROP TABLE IF EXISTS a_relationship;
    DROP TABLE IF EXISTS a_user_object_favorite;
    DROP TABLE IF EXISTS a_user_object_subscription;
    DROP TABLE IF EXISTS acm;
    DROP TABLE IF EXISTS acmkey;
    DROP TABLE IF EXISTS acmpart;
    DROP TABLE IF EXISTS acmvalue;
    DROP TABLE IF EXISTS field_changes;
    DROP TABLE IF EXISTS object_pathing;
    DROP TABLE IF EXISTS object_tag;
    DROP TABLE IF EXISTS objectacm;
    DROP TABLE IF EXISTS relationship;
    DROP TABLE IF EXISTS user_object_favorite;
    DROP TABLE IF EXISTS user_object_subscription;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170630_drop_tables();

-- TODO: Recreate triggers without those getting dropped
INSERT INTO migration_status SET description = '20170630_combined create triggers';


DROP TRIGGER IF EXISTS ti_acmgrantee;
-- +migrate StatementBegin
CREATE TRIGGER ti_acmgrantee
BEFORE INSERT ON acmgrantee FOR EACH ROW
BEGIN
    DECLARE error_msg varchar(128) default '';
    DECLARE thisTableName varchar(128) default 'acmgrantee';
    # All fields lowercase
    IF NEW.grantee IS NOT NULL THEN
        SET NEW.grantee := LOWER(NEW.grantee);
    END IF;
    IF NEW.resourceString IS NOT NULL THEN
        SET NEW.resourceString := LOWER(NEW.resourceString);
    END IF;
    IF NEW.projectName IS NOT NULL THEN
        SET NEW.projectName := LOWER(NEW.projectName);
    END IF;
    IF NEW.projectDisplayName IS NOT NULL THEN
        SET NEW.projectDisplayName := LOWER(NEW.projectDisplayName);
    END IF;
    IF NEW.groupName IS NOT NULL THEN
        SET NEW.groupName := LOWER(NEW.groupName);
    END IF;
    IF NEW.userDistinguishedName IS NOT NULL THEN
        SET NEW.userDistinguishedName := LOWER(NEW.userDistinguishedName);
    END IF;
    IF NEW.displayName IS NOT NULL THEN
        SET NEW.displayName := LOWER(NEW.displayName);
    END IF;    
    # Check required fields
    IF NEW.grantee IS NULL THEN
        SET error_msg := concat(error_msg, 'Field grantee must be set ');
    ELSE
        IF (SELECT 1=1 FROM acmgrantee WHERE binary grantee = NEW.grantee) IS NOT NULL THEN
            SET error_msg := concat(error_msg, 'Field grantee must be unique ');
        END IF;
    END IF;
    IF NEW.resourceString IS NULL THEN
        SET error_msg := concat(error_msg, 'Field resourceString must be set ');
    ELSE
        IF (SELECT 1=1 FROM acmgrantee WHERE binary resourceString = NEW.resourceString) IS NOT NULL THEN
            SET error_msg := concat(error_msg, 'Field resourceString must be unique ');
        END IF;
    END IF;
    IF length(error_msg) > 0 THEN
        SET error_msg := concat(error_msg, 'when inserting record');
        signal sqlstate '45000' set message_text = error_msg;
    END IF;
    # Add grantee to acmvalue2 is not yet present
    IF (SELECT 1=1 FROM acmvalue2 WHERE binary name = NEW.grantee) IS NULL THEN
        INSERT INTO acmvalue2 SET name = NEW.grantee;
    END IF;
END;
-- +migrate StatementEnd
DROP TRIGGER IF EXISTS ti_dbstate;
INSERT INTO migration_status SET description = '20170630_combined setting schema version';
-- +migrate StatementBegin
CREATE TRIGGER ti_dbstate
BEFORE INSERT ON dbstate FOR EACH ROW
BEGIN
	# Rules
	# Can only be one record
    IF EXISTS (select null from dbstate) THEN
		signal sqlstate '45000' set message_text = 'Only one record is allowed in dbstate table.';
	END IF;
	# Force values on create
	# Created Date
	SET NEW.createdDate := current_timestamp(6);
	# Modified Date
	SET NEW.modifiedDate := current_timestamp(6);
	# Version should be changed if the schema changes
	SET NEW.schemaversion := '20170630'; 
	# Identifier is randomized as a GUID
	SET NEW.identifier := concat(@@hostname, '-', left(uuid(),8));
END;
-- +migrate StatementEnd
DROP TRIGGER IF EXISTS tu_dbstate;
-- +migrate StatementBegin
CREATE TRIGGER tu_dbstate
BEFORE UPDATE ON dbstate FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	# Rules
	# createdDate cannot be changed
	IF (NEW.createdDate <> OLD.createdDate) AND length(error_msg) < 74 THEN
		signal sqlstate '45000' set message_text = 'Unable to set createdDate ';
	END IF;
	# identifier cannot be changed
	IF (NEW.identifier <> OLD.identifier) THEN
		signal sqlstate '45000' set message_text = 'Identifier cannot be changed';
	END IF;
	# version must be different
	IF (NEW.schemaversion = OLD.schemaversion) THEN
		signal sqlstate '45000' set message_text = 'Version must be changed';
	END IF;

	# Force values
	# modifiedDate
	SET NEW.modifiedDate = current_timestamp(6);
END;
-- +migrate StatementEnd
DROP TRIGGER IF EXISTS ti_object;
-- +migrate StatementBegin
CREATE TRIGGER ti_object
BEFORE INSERT ON object FOR EACH ROW
BEGIN
    DECLARE error_msg varchar(128) default '';
    DECLARE thisTableName varchar(128) default 'object';
    DECLARE type_contentConnector varchar(2000) default '';

    # Rules
    # Type must be specified
    IF NOT EXISTS (select null from object_type where isdeleted = 0 and id = new.typeid) THEN
        SET error_msg := concat(error_msg, 'Field typeId required ');
    END IF;
    # US Persons Data is NULLed if empty (will change to Unknown)
    IF NEW.containsUSPersonsData IS NOT NULL AND NEW.containsUSPersonsData = '' THEN
        SET NEW.containsUSPersonsData := NULL;
    END IF;
    # FOIA Exempt is NULLed if empty (will change to Unknown)
    IF NEW.exemptFromFOIA IS NOT NULL AND NEW.exemptFromFOIA = '' THEN
        SET NEW.exemptFromFOIA := NULL;
    END IF;
    # ParentId must be valid if specified
    IF NEW.parentId IS NOT NULL AND NEW.parentId = '' THEN
        SET NEW.parentId := NULL;
    END IF;
    IF NEW.parentId IS NOT NULL THEN
        IF NOT EXISTS (select null from object where isdeleted = 0 and id = new.parentid) THEN
            SET error_msg := concat(error_msg, 'Field parentId must be valid ');
        END IF;
    END IF;
    IF error_msg <> '' THEN
        SET error_msg := concat(error_msg, 'when inserting record into ', thisTableName);
        signal sqlstate '45000' set message_text = error_msg;
    END IF;

    # Force values on create
    SET NEW.id := ordered_uuid(UUID());
    SET NEW.createdDate := current_timestamp(6);
    SET NEW.modifiedDate := current_timestamp(6);
    SET NEW.modifiedBy := NEW.createdBy;
    SET NEW.isDeleted := 0;
    SET NEW.deletedDate := NULL;
    SET NEW.deletedBy := NULL;
    SET NEW.isAncestorDeleted := 0;
    SET NEW.isExpunged := 0;
    SET NEW.expungedDate := NULL;
    SET NEW.expungedBy := NULL;
    # Assign Owner if not set
    IF NEW.ownedBy IS NULL OR NEW.ownedBy = '' THEN
        SET NEW.ownedBy := concat('user/', NEW.createdBy);
    END IF;
    SET NEW.ownedBy := calcResourceString(NEW.ownedBy);
    SET NEW.ownedByID := calcGranteeIDFromResourceString(NEW.ownedBy);
    SET NEW.changeCount := 0;
    # Standard change token formula
    SET NEW.changeToken := md5(CONCAT(CAST(NEW.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));
    # Assign contentConnector if not set
    IF NEW.contentConnector IS NULL OR NEW.contentConnector = '' THEN
        SELECT contentConnector FROM object_type WHERE isDeleted = 0 AND id = NEW.typeId INTO type_contentConnector;
        SET NEW.contentConnector := type_contentConnector;
    END IF;
    # Assign US Persons Data if not set
    IF NEW.containsUSPersonsData IS NULL THEN
        SET NEW.containsUSPersonsData := 'Unknown';
    END IF;
    # Assign FOIA Exempt status if not set
    IF NEW.exemptFromFOIA IS NULL THEN
        SET NEW.exemptFromFOIA := 'Unknown';
    END IF;


    # Archive table
    INSERT INTO
        a_object
    (
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,isAncestorDeleted
        ,isExpunged
        ,expungedDate
        ,expungedBy
        ,changeCount
        ,changeToken
        ,ownedBy
        ,typeId
        ,name
        ,description
        ,parentId
        ,contentConnector
        ,rawAcm
        ,contentType
        ,contentSize
        ,contentHash
        ,encryptIV
        ,containsUSPersonsData
        ,exemptFromFOIA
        ,acmId
        ,ownedById
    ) values (
        NEW.id
        ,NEW.createdDate
        ,NEW.createdBy
        ,NEW.modifiedDate
        ,NEW.modifiedBy
        ,NEW.isDeleted
        ,NEW.deletedDate
        ,NEW.deletedBy
        ,NEW.isAncestorDeleted
        ,NEW.isExpunged
        ,NEW.expungedDate
        ,NEW.expungedBy
        ,NEW.changeCount
        ,NEW.changeToken
        ,NEW.ownedBy
        ,NEW.typeId
        ,NEW.name
        ,NEW.description
        ,NEW.parentId
        ,NEW.contentConnector
        ,NEW.rawAcm
        ,NEW.contentType
        ,NEW.contentSize
        ,NEW.contentHash
        ,NEW.encryptIV
        ,NEW.containsUSPersonsData
        ,NEW.exemptFromFOIA
        ,NEW.acmId
        ,NEW.ownedById
    );
END;
-- +migrate StatementEnd
DROP TRIGGER IF EXISTS tu_object;
-- +migrate StatementBegin
CREATE TRIGGER tu_object
BEFORE UPDATE ON object FOR EACH ROW
BEGIN
    DECLARE error_msg varchar(128) default '';
    DECLARE thisTableName varchar(128) default 'object';
    DECLARE type_contentConnector varchar(2000) default '';

    # Rules
    # id cannot be changed
    IF (NEW.id <> OLD.id) AND length(error_msg) < 83 THEN
        SET error_msg := concat(error_msg, 'Unable to set id ');
    END IF;
    # createdDate cannot be changed
    IF (NEW.createdDate <> OLD.createdDate) AND length(error_msg) < 74 THEN
        SET error_msg := concat(error_msg, 'Unable to set createdDate ');
    END IF;
    # createdBy cannot be changed
    IF (NEW.createdBy <> OLD.createdBy) AND length(error_msg) < 76 THEN
        SET error_msg := concat(error_msg, 'Unable to set createdBy ');
    END IF;
    # changeCount cannot be changed
    IF (NEW.changeCount <> OLD.changeCount) AND length(error_msg) < 74 THEN
        SET error_msg := concat(error_msg, 'Unable to set changeCount ');
    END IF;
    # changeToken must be given and match the record
    IF (NEW.changeToken IS NULL OR NEW.changeToken = '') and length(error_msg) < 73 THEN
        SET error_msg := concat(error_msg, 'Field changeToken required ');
    END IF;
    IF (NEW.changeToken <> OLD.changeToken) and length(error_msg) < 71 THEN
        SET error_msg := concat(error_msg, 'Field changeToken must match ');
    END IF;
    # TypeId must be specified
    IF NOT EXISTS (select null from object_type where isdeleted = 0 and id = new.typeid) THEN
        SET error_msg := concat(error_msg, 'Field typeId required ');
    END IF;
    # US Persons Data is set to old value if null/empty
    IF NEW.containsUSPersonsData IS NULL OR NEW.containsUSPersonsData = '' THEN
        SET NEW.containsUSPersonsData := OLD.containsUSPersonsData;
    END IF;
    # FOIA Exempt is set to old value if null/empty
    IF NEW.exemptFromFOIA IS NULL OR NEW.exemptFromFOIA = '' THEN
        SET NEW.exemptFromFOIA := OLD.exemptFromFOIA;
    END IF;    
    # ParentId must be valid if specified
    IF NEW.parentId IS NOT NULL AND LENGTH(NEW.parentId) = 0 THEN
        SET NEW.parentId := NULL;
    END IF;
    IF NEW.parentId IS NOT NULL THEN
        IF NOT EXISTS (select null from object where (isDeleted = 0 or (NEW.IsDeleted <> OLD.IsDeleted)) AND id = NEW.parentId) THEN
            SET error_msg := concat(error_msg, 'Field parentId must be valid ');
        END IF;
    END IF;    
    IF length(error_msg) > 0 THEN
        SET error_msg := concat(error_msg, 'when updating record');
        signal sqlstate '45000' set message_text = error_msg;
    END IF;

    # Force values on modify
    SET NEW.modifiedDate := current_timestamp(6);
    IF (NEW.isDeleted <> OLD.isDeleted) THEN
        IF  (NEW.IsDeleted = 1) THEN
            SET NEW.deletedDate := current_timestamp(6);
            SET NEW.deletedBy := NEW.modifiedBy;
        ELSE
            SET NEW.deletedDate := NULL;
            SET NEW.deletedBy := NULL;
        END IF;                
    END IF;
    SET NEW.changeCount := OLD.changeCount + 1;
    # Standard change token formula
    SET NEW.changeToken := md5(CONCAT(CAST(OLD.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));
    # Assign Owner if not set
    IF NEW.ownedBy IS NULL OR NEW.ownedBy = '' THEN
        SET NEW.ownedBy := concat('user/', NEW.createdBy);
    END IF;
    SET NEW.ownedBy := calcResourceString(NEW.ownedBy);
    # ownedByID derived from ownedBy
    SET NEW.ownedByID := calcGranteeIDFromResourceString(NEW.ownedBy);
    # Assign US Persons Data if not set
    IF NEW.containsUSPersonsData IS NULL THEN
        SET NEW.containsUSPersonsData = 'Unknown';
    END IF;
    # Assign FOIA Exempt status if not set
    IF NEW.exemptFromFOIA IS NULL THEN
        SET NEW.exemptFromFOIA = 'Unknown';
    END IF;

    # Archive table
    INSERT INTO
        a_object
    (
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,isAncestorDeleted
        ,isExpunged
        ,expungedDate
        ,expungedBy
        ,changeCount
        ,changeToken
        ,ownedBy
        ,typeId
        ,name
        ,description
        ,parentId
        ,contentConnector
        ,rawAcm
        ,contentType
        ,contentSize
        ,contentHash
        ,encryptIV
        ,containsUSPersonsData
        ,exemptFromFOIA
        ,acmId
        ,ownedById
    ) values (
        NEW.id
        ,NEW.createdDate
        ,NEW.createdBy
        ,NEW.modifiedDate
        ,NEW.modifiedBy
        ,NEW.isDeleted
        ,NEW.deletedDate
        ,NEW.deletedBy
        ,NEW.isAncestorDeleted
        ,NEW.isExpunged
        ,NEW.expungedDate
        ,NEW.expungedBy
        ,NEW.changeCount
        ,NEW.changeToken
        ,NEW.ownedBy
        ,NEW.typeId
        ,NEW.name
        ,NEW.description
        ,NEW.parentId
        ,NEW.contentConnector
        ,NEW.rawAcm
        ,NEW.contentType
        ,NEW.contentSize
        ,NEW.contentHash
        ,NEW.encryptIV
        ,NEW.containsUSPersonsData
        ,NEW.exemptFromFOIA
        ,NEW.acmId
        ,NEW.ownedById
    );
END;
-- +migrate StatementEnd
DROP TRIGGER IF EXISTS ti_object_permission;
-- +migrate StatementBegin
CREATE TRIGGER ti_object_permission
BEFORE INSERT ON object_permission FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'object_permission';
	DECLARE vCreatedByID int unsigned default 0;
	DECLARE vGranteeID int unsigned default 0;

	# Rules
	# ObjectId must be specified
    IF NOT EXISTS (select null from object where id = new.objectid) THEN
		SET error_msg := concat(error_msg, 'Field objectId required ');
	END IF;
	# Grantee must be specified
	IF NEW.grantee IS NULL OR NEW.grantee = '' THEN
		SET error_msg := concat(error_msg, 'Field grantee required ');
	END IF;
    # ACM Share must be specified
    IF NEW.acmShare IS NULL OR NEW.acmShare = '' THEN
        SET error_msg := concat(error_msg, 'Field acmShare required ');
    END IF;
	IF error_msg <> '' THEN
		SET error_msg := concat(error_msg, 'when inserting record into ', thisTableName);
		signal sqlstate '45000' set message_text = error_msg;
	END IF;

	# Force values on create
	SET NEW.id := ordered_uuid(UUID());
	SET NEW.createdDate := current_timestamp(6);
	SET NEW.modifiedDate := current_timestamp(6);
	SET NEW.modifiedBy := NEW.createdBy;
	SET NEW.isDeleted := 0;
	SET NEW.deletedDate := NULL;
	SET NEW.deletedBy := NULL;
	SET NEW.changeCount := 0;
	SELECT id FROM acmvalue2 WHERE name = aacflatten(NEW.CreatedBy) INTO vCreatedByID;
	SELECT id FROM acmvalue2 WHERE name = aacflatten(NEW.Grantee) INTO vGranteeID;
	SET NEw.createdByID := vCreatedByID;
	SET NEW.granteeID := vGranteeID;
	
	# Standard change token formula
	SET NEW.changeToken := md5(CONCAT(CAST(NEW.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));

	# Archive table
	INSERT INTO
		a_object_permission
	(
		id
		,createdDate
		,createdBy
		,modifiedDate
		,modifiedBy
		,isDeleted
		,deletedDate
		,deletedBy
		,changeCount
		,changeToken
		,objectId
		,grantee
        ,acmShare
		,allowCreate
		,allowRead
		,allowUpdate
		,allowDelete
		,allowShare
		,explicitShare
		,encryptKey
		,permissionIV
		,permissionMAC
		,createdByID
		,granteeID
	) values (
		NEW.id
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.isDeleted
		,NEW.deletedDate
		,NEW.deletedBy
		,NEW.changeCount
		,NEW.changeToken
		,NEW.objectId
		,NEW.grantee
        ,NEW.acmShare
		,NEW.allowCreate
		,NEW.allowRead
		,NEW.allowUpdate
		,NEW.allowDelete
		,NEW.allowShare
		,NEW.explicitShare
		,NEW.encryptKey
		,NEW.permissionIV
		,NEW.permissionMAC
		,NEW.createdByID
		,NEW.granteeID
	);
END;
-- +migrate StatementEnd    
DROP TRIGGER IF EXISTS tu_object_permission;
-- +migrate StatementBegin
CREATE TRIGGER tu_object_permission
BEFORE UPDATE ON object_permission FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'object_permission';
	
	# Rules
	# id cannot be changed
	IF (NEW.id <> OLD.id) AND length(error_msg) < 83 THEN
		SET error_msg := concat(error_msg, 'Unable to set id ');
	END IF;
	# objectId cannot be changed
	IF (NEW.objectId <> OLD.objectId) AND length(error_msg) < 77 THEN
		SET error_msg := concat(error_msg, 'Unable to set objectId ');
	END IF;
	# createdDate cannot be changed
	IF (NEW.createdDate <> OLD.createdDate) AND length(error_msg) < 74 THEN
		SET error_msg := concat(error_msg, 'Unable to set createdDate ');
	END IF;
	# createdBy cannot be changed
	IF (NEW.createdBy <> OLD.createdBy) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set createdBy ');
	END IF;
	# changeCount cannot be changed
	IF (NEW.changeCount <> OLD.changeCount) AND length(error_msg) < 74 THEN
		SET error_msg := concat(error_msg, 'Unable to set changeCount ');
	END IF;
	# changeToken must be given and match the record
	IF (NEW.changeToken IS NULL OR NEW.changeToken = '') and length(error_msg) < 73 THEN
		SET error_msg := concat(error_msg, 'Field changeToken required ');
	END IF;
	IF (NEW.changeToken <> OLD.changeToken) and length(error_msg) < 71 THEN
		SET error_msg := concat(error_msg, 'Field changeToken must match ');
	END IF;
	# grantee cannot be changed
	IF (NEW.grantee <> OLD.grantee) AND length(error_msg) < 78 THEN
		SET error_msg := concat(error_msg, 'Unable to set grantee ');
	END IF;
	# acmShare cannot be changed
	IF (NEW.acmShare <> OLD.acmShare) AND length(error_msg) < 78 THEN
		SET error_msg := concat(error_msg, 'Unable to set acmShare ');
	END IF;
	# permission cannot be changed
	IF (NEW.allowCreate <> OLD.allowCreate) AND length(error_msg) < 78 THEN
		SET error_msg := concat(error_msg, 'Unable to set allowCreate ');
	END IF;
	# permission cannot be changed
	IF (NEW.allowRead <> OLD.allowRead) AND length(error_msg) < 78 THEN
		SET error_msg := concat(error_msg, 'Unable to set allowRead ');
	END IF;
	# permission cannot be changed
	IF (NEW.allowUpdate <> OLD.allowUpdate) AND length(error_msg) < 78 THEN
		SET error_msg := concat(error_msg, 'Unable to set allowRead ');
	END IF;
	# permission cannot be changed
	IF (NEW.allowDelete <> OLD.allowDelete) AND length(error_msg) < 78 THEN
		SET error_msg := concat(error_msg, 'Unable to set allowDelete ');
	END IF;
	# permission cannot be changed
	IF (NEW.allowShare <> OLD.allowShare) AND length(error_msg) < 78 THEN
		SET error_msg := concat(error_msg, 'Unable to set allowShare ');
	END IF;
	
	#every immutable field has been checked for mutation at this point.
	#all other fields are mutable.

	#note... we need to always allow mutation of the encryptKey,permissionIV,permissionMAC, otherwise we will render things like
	#deleted fields unrecoverable.
	
	# Force values on modify
	# The only modification allowed is to mark as deleted...
	SET NEW.modifiedDate := current_timestamp(6);
	IF NEW.modifiedBy IS NULL OR NEW.modifiedBy = '' THEN
		SET NEW.modifiedBy := NEW.deletedBy;
	END IF;

	#either we are deleting... 		
	IF (NEW.isDeleted = 1 AND OLD.isDeleted = 0) THEN
		# deletedBy must be set
		IF (NEW.deletedBy IS NULL) AND length(error_msg) < 75 THEN
			SET error_msg := concat(error_msg, 'Field deletedBy required ');
		END IF;
		
		SET NEW.deletedDate := current_timestamp(6);
		IF NEW.deletedBy IS NULL OR NEW.deletedBy = '' THEN
			SET NEW.deletedBy := NEW.modifiedBy;
		END IF;
	ELSE
		#or updating keys
		IF (NEW.isDeleted <> OLD.isDeleted) THEN
			SET error_msg := concat(error_msg, 'Undelete is disallowed ');
		END IF;
		IF ((NEW.encryptKey = OLD.encryptKey) AND (NEW.permissionIV = OLD.permissionIV) AND (NEW.permissionMAC = OLD.permissionMAC)) THEN
			SET error_msg := concat(error_msg, 'We should be updating keys ');
		END IF;
	END IF; 
	IF length(error_msg) > 0 THEN
		SET error_msg := concat(error_msg, 'when updating record');
		signal sqlstate '45000' set message_text = error_msg;
	END IF;
	
	SET NEW.changeCount := OLD.changeCount + 1;
	
	# Standard change token formula
	SET NEW.changeToken := md5(CONCAT(CAST(OLD.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));

	# Archive table
	INSERT INTO
		a_object_permission
	(
		id
		,createdDate
		,createdBy
		,modifiedDate
		,modifiedBy
		,isDeleted
		,deletedDate
		,deletedBy
		,changeCount
		,changeToken
		,objectId
		,grantee
        ,acmShare
		,allowCreate
		,allowRead
		,allowUpdate
		,allowDelete
		,allowShare
		,explicitShare
		,encryptKey
		,permissionIV
		,permissionMAC
		,createdByID
		,granteeID
	) values (
		NEW.id
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.isDeleted
		,NEW.deletedDate
		,NEW.deletedBy
		,NEW.changeCount
		,NEW.changeToken
		,NEW.objectId
		,NEW.grantee
        ,NEW.acmShare
		,NEW.allowCreate
		,NEW.allowRead
		,NEW.allowUpdate
		,NEW.allowDelete
		,NEW.allowShare
		,NEW.explicitShare
		,NEW.encryptKey
		,NEW.permissionIV
		,NEW.permissionMAC
		,NEW.createdByID
		,NEW.granteeID
	);    
END;
-- +migrate StatementEnd
DROP TRIGGER IF EXISTS ti_object_property;
-- +migrate StatementBegin
CREATE TRIGGER ti_object_property
BEFORE INSERT ON object_property FOR EACH ROW
BEGIN
	DECLARE thisTableName varchar(128) default 'object_property';
	# Force values on create
	SET NEW.id := ordered_uuid(UUID());
	SET NEW.createdDate := current_timestamp(6);
	SET NEW.modifiedDate := current_timestamp(6);
	SET NEW.modifiedBy := NEW.createdBy;
	SET NEW.isDeleted := 0;
	SET NEW.deletedDate := NULL;
	SET NEW.deletedBy := NULL;
	# No archive table for this many-to-many relationship table. 	
END;
-- +migrate StatementEnd
DROP TRIGGER IF EXISTS tu_object_property;
-- +migrate StatementBegin
CREATE TRIGGER tu_object_property
BEFORE UPDATE ON object_property FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'object_property';
	# Rules
	# only deletes are allowed.
	# id cannot bechanged
	IF (NEW.id <> OLD.id) AND length(error_msg) < 83 THEN
		SET error_msg := concat(error_msg, 'Unable to set id ');
	END IF;
	# createdDate cannot be changed
	IF (NEW.createdDate <> OLD.createdDate) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set createdDate ');
	END IF;
	# createdBy cannot be changed
	IF (NEW.createdBy <> OLD.createdBy) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set createdBy ');
	END IF;
	# objectId cannot be changed
	IF (NEW.objectId <> OLD.objectId) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set objectId ');
	END IF;
	# propertyId cannot be changed
	IF (NEW.propertyId <> OLD.propertyId) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set propertyId ');
	END IF;
	IF length(error_msg) > 0 THEN
		SET error_msg := concat(error_msg, 'when updating record');
		signal sqlstate '45000' set message_text = error_msg;
	END IF;
	# Force values on modify
    # The only modification allowed is to mark as deleted...
	SET NEW.modifiedDate := current_timestamp(6);
   	IF NEW.modifiedBy IS NULL OR NEW.modifiedBy = '' THEN
		SET NEW.modifiedBy := NEW.deletedBy;
	END IF;
	SET NEW.isDeleted = 1;
	SET NEW.deletedDate := current_timestamp(6);
	IF NEW.deletedBy IS NULL OR NEW.deletedBy = '' THEN
		SET NEW.deletedBy := NEW.modifiedBy;
	END IF;	
END;
-- +migrate StatementEnd
DROP TRIGGER IF EXISTS ti_object_type;
-- +migrate StatementBegin
CREATE TRIGGER ti_object_type
BEFORE INSERT ON object_type FOR EACH ROW
BEGIN
    DECLARE error_msg varchar(128) default '';
    DECLARE thisTableName varchar(128) default 'object_type';
    # Rules
    # Name must be unique for non-deletedBy
    IF EXISTS (select null from object_type where isdeleted = 0 and name = new.name) THEN
        SET error_msg := concat(error_msg, 'Field name must be unique ');
    END IF;
    IF error_msg <> '' THEN
        SET error_msg := concat(error_msg, 'when inserting record into ', thisTableName);
        signal sqlstate '45000' set message_text = error_msg;
    END IF;
    # Force values on create
    SET NEW.id := ordered_uuid(UUID());
    SET NEW.createdDate := current_timestamp(6);
    SET NEW.modifiedDate := current_timestamp(6);
    SET NEW.modifiedBy := NEW.createdBy;
    SET NEW.isDeleted := 0;
    SET NEW.deletedDate := NULL;
    SET NEW.deletedBy := NULL;
    # Assign Owner if not set
    IF NEW.ownedBy IS NULL OR NEW.ownedBy = '' THEN
        SET NEW.ownedBy := concat('user/', NEW.createdBy);
    END IF;
    SET NEW.changeCount := 0;
    # Standard change token formula
    SET NEW.changeToken := md5(CONCAT(CAST(NEW.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));
    # Archive table
    INSERT INTO
        a_object_type
    (
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,ownedBy
        ,changeCount
        ,changeToken
        ,name
        ,description
        ,contentConnector
    ) values (
        NEW.id
        ,NEW.createdDate
        ,NEW.createdBy
        ,NEW.modifiedDate
        ,NEW.modifiedBy
        ,NEW.isDeleted
        ,NEW.deletedDate
        ,NEW.deletedBy
        ,NEW.ownedBy
        ,NEW.changeCount
        ,NEW.changeToken
        ,NEW.name
        ,NEW.description
        ,NEW.contentConnector
    );
END;
-- +migrate StatementEnd
DROP TRIGGER IF EXISTS tu_object_type;
-- +migrate StatementBegin
CREATE TRIGGER tu_object_type
BEFORE UPDATE ON object_type FOR EACH ROW
BEGIN
    DECLARE error_msg varchar(128) default '';
    DECLARE thisTableName varchar(128) default 'object_type';
    # Rules
    # id cannot be changed
    IF (NEW.id <> OLD.id) AND length(error_msg) < 83 THEN
        SET error_msg := concat(error_msg, 'Unable to set id ');
    END IF;
    # createdDate cannot be changed
    IF (NEW.createdDate <> OLD.createdDate) AND length(error_msg) < 74 THEN
        SET error_msg := concat(error_msg, 'Unable to set createdDate ');
    END IF;
    # createdBy cannot be changed
    IF (NEW.createdBy <> OLD.createdBy) AND length(error_msg) < 76 THEN
        SET error_msg := concat(error_msg, 'Unable to set createdBy ');
    END IF;
    # changeCount cannot be changed
    IF (NEW.changeCount <> OLD.changeCount) AND length(error_msg) < 74 THEN
        SET error_msg := concat(error_msg, 'Unable to set changeCount ');
    END IF;
    # Name must be unique for non-deletedBy
    IF EXISTS (select null from object_type where isdeleted = 0 and name = new.name and id <> old.id) THEN 
        SET error_msg := concat(error_msg, 'Field name must be unique ');
    END IF;
    # changeToken must be given and match the record
    IF (NEW.changeToken IS NULL OR NEW.changeToken = '') and length(error_msg) < 73 THEN
        SET error_msg := concat(error_msg, 'Field changeToken required ');
    END IF;
    IF (NEW.changeToken <> OLD.changeToken) and length(error_msg) < 71 THEN
        SET error_msg := concat(error_msg, 'Field changeToken must match ');
    END IF;
    IF length(error_msg) > 0 THEN
        SET error_msg := concat(error_msg, 'when updating record');
        signal sqlstate '45000' set message_text = error_msg;
    END IF;
    # Force values on modify
    SET NEW.modifiedDate := current_timestamp(6);
    IF (NEW.isDeleted <> OLD.isDeleted) THEN
        IF  (NEW.IsDeleted = 1) THEN
            SET NEW.deletedDate := current_timestamp(6);
            SET NEW.deletedBy := NEW.modifiedBy;
        ELSE
            SET NEW.deletedDate := NULL;
            SET NEW.deletedBy := NULL;
        END IF;                
    END IF;
    SET NEW.changeCount := OLD.changeCount + 1;
    # Standard change token formula
    SET NEW.changeToken := md5(CONCAT(CAST(OLD.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));
    # Assign Owner if not set
    IF NEW.ownedBy IS NULL OR NEW.ownedBy = '' THEN
        SET NEW.ownedBy := concat('user/', NEW.createdBy);
    END IF;
    # Archive table
    INSERT INTO
        a_object_type
    (
        id
        ,createdDate
        ,createdBy
        ,modifiedDate
        ,modifiedBy
        ,isDeleted
        ,deletedDate
        ,deletedBy
        ,ownedBy
        ,changeCount
        ,changeToken
        ,name
        ,description
        ,contentConnector
    ) values (
        NEW.id
        ,NEW.createdDate
        ,NEW.createdBy
        ,NEW.modifiedDate
        ,NEW.modifiedBy
        ,NEW.isDeleted
        ,NEW.deletedDate
        ,NEW.deletedBy
        ,NEW.ownedBy
        ,NEW.changeCount
        ,NEW.changeToken
        ,NEW.name
        ,NEW.description
        ,NEW.contentConnector
    );
END;
-- +migrate StatementEnd
DROP TRIGGER IF EXISTS ti_object_type_property;
-- +migrate StatementBegin
CREATE TRIGGER ti_object_type_property
BEFORE INSERT ON object_type_property FOR EACH ROW
BEGIN
	DECLARE thisTableName varchar(128) default 'object_type_property';
	# Force values on create
	SET NEW.id := ordered_uuid(UUID());
	SET NEW.createdDate := current_timestamp(6);
	SET NEW.modifiedDate := current_timestamp(6);
	SET NEW.modifiedBy := NEW.createdBy;
	SET NEW.isDeleted := 0;
	SET NEW.deletedDate := NULL;
	SET NEW.deletedBy := NULL;
	# No archive table
END;
-- +migrate StatementEnd
DROP TRIGGER IF EXISTS tu_object_type_property;
-- +migrate StatementBegin
CREATE TRIGGER tu_object_type_property
BEFORE UPDATE ON object_type_property FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'object_type_property';
	# Rules
	# only deletes are allowed.
	# id cannot bechanged
	IF (NEW.id <> OLD.id) AND length(error_msg) < 83 THEN
		SET error_msg := concat(error_msg, 'Unable to set id ');
	END IF;
	# createdDate cannot be changed
	IF (NEW.createdDate <> OLD.createdDate) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set createdDate ');
	END IF;
	# createdBy cannot be changed
	IF (NEW.createdBy <> OLD.createdBy) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set createdBy ');
	END IF;
	# typeId cannot be changed
	IF (NEW.typeId <> OLD.typeId) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set typeId ');
	END IF;
	# propertyId cannot be changed
	IF (NEW.propertyId <> OLD.propertyId) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set propertyId ');
	END IF;
	IF length(error_msg) > 0 THEN
		SET error_msg := concat(error_msg, 'when updating record');
		signal sqlstate '45000' set message_text = error_msg;
	END IF;
	# Force values on modify
	SET NEW.modifiedDate := current_timestamp(6);
   	IF NEW.modifiedBy IS NULL OR NEW.modifiedBy = '' THEN
		SET NEW.modifiedBy := NEW.deletedBy;
	END IF;
	SET NEW.isDeleted = 1;
	SET NEW.deletedDate := current_timestamp(6);
	IF NEW.deletedBy IS NULL OR NEW.deletedBy = '' THEN
		SET NEW.deletedBy := NEW.modifiedBy;
	END IF;    
END;
-- +migrate StatementEnd
DROP TRIGGER IF EXISTS ti_property;
-- +migrate StatementBegin
CREATE TRIGGER ti_property
BEFORE INSERT ON property FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'property';
	# Rules
	# name must be specified
	IF NEW.name IS NULL OR NEW.name = '' THEN
		SET error_msg := concat(error_msg, 'Field name required ');
	END IF;
	IF error_msg <> '' THEN
		SET error_msg := concat(error_msg, 'where inserting record into ', thisTableName);
		signal sqlstate '45000' set message_text = error_msg;
	END IF;
	# Force values on create
	SET NEW.id := ordered_uuid(UUID());
	SET NEW.createdDate := current_timestamp(6);
	SET NEW.modifiedDate := current_timestamp(6);
	SET NEW.modifiedBy := NEW.createdBy;
	SET NEW.isDeleted := 0;
	SET NEW.deletedDate := NULL;
	SET NEW.deletedBy := NULL;
	SET NEW.changeCount := 0;
	# Standard change token formula
	SET NEW.changeToken := md5(CONCAT(CAST(NEW.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));
	# Archive table
	INSERT INTO
		a_property
	(
		id
		,createdDate
		,createdBy
		,modifiedDate
		,modifiedBy
		,isDeleted
		,deletedDate
		,deletedBy
		,changeCount
		,changeToken
		,name
		,propertyValue
		,classificationPM
	) values (
		NEW.id
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.isDeleted
		,NEW.deletedDate
		,NEW.deletedBy
		,NEW.changeCount
		,NEW.changeToken
		,NEW.name
		,NEW.propertyValue
		,NEW.classificationPM
	);
END;
-- +migrate StatementEnd
DROP TRIGGER IF EXISTS tu_property;
-- +migrate StatementBegin
CREATE TRIGGER tu_property
BEFORE UPDATE ON property FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'property';
	# Rules
	# id cannot be changed
	IF (NEW.id <> OLD.id) AND length(error_msg) < 83 THEN
		SET error_msg := concat(error_msg, 'Unable to set id ');
	END IF;
	# createdDate cannot be changed
	IF (NEW.createdDate <> OLD.createdDate) AND length(error_msg) < 74 THEN
		SET error_msg := concat(error_msg, 'Unable to set createdDate ');
	END IF;
	# createdBy cannot be changed
	IF (NEW.createdBy <> OLD.createdBy) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set createdBy ');
	END IF;
	# changeCount cannot be changed
	IF (NEW.changeCount <> OLD.changeCount) AND length(error_msg) < 74 THEN
		SET error_msg := concat(error_msg, 'Unable to set changeCount ');
	END IF;
	# changeToken must be given and match the record
	IF (NEW.changeToken IS NULL OR NEW.changeToken = '') and length(error_msg) < 73 THEN
		SET error_msg := concat(error_msg, 'Field changeToken required ');
	END IF;
	IF (NEW.changeToken <> OLD.changeToken) and length(error_msg) < 71 THEN
		SET error_msg := concat(error_msg, 'Field changeToken must match ');
	END IF;
	IF length(error_msg) > 0 THEN
		SET error_msg := concat(error_msg, 'when updating record');
		signal sqlstate '45000' set message_text = error_msg;
	END IF;
	# Force values on modify
	SET NEW.modifiedDate := current_timestamp(6);
    IF (NEW.isDeleted <> OLD.isDeleted) THEN
        IF  (NEW.IsDeleted = 1) THEN
            SET NEW.deletedDate := current_timestamp(6);
            SET NEW.deletedBy := NEW.modifiedBy;
        ELSE
            SET NEW.deletedDate := NULL;
            SET NEW.deletedBy := NULL;
        END IF;                
    END IF;
    SET NEW.changeCount := OLD.changeCount + 1;
	# Standard change token formula
	SET NEW.changeToken := md5(CONCAT(CAST(OLD.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));
	# Archive table
	INSERT INTO
		a_property
	(
		id
		,createdDate
		,createdBy
		,modifiedDate
		,modifiedBy
		,isDeleted
		,deletedDate
		,deletedBy
		,changeCount
		,changeToken
		,name
		,propertyValue
		,classificationPM
	) values (
		NEW.id
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.isDeleted
		,NEW.deletedDate
		,NEW.deletedBy
		,NEW.changeCount
		,NEW.changeToken
		,NEW.name
		,NEW.propertyValue
		,NEW.classificationPM
	);
END;
-- +migrate StatementEnd
DROP TRIGGER IF EXISTS ti_user;
-- +migrate StatementBegin
CREATE TRIGGER ti_user
BEFORE INSERT ON user FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'user';
	# Rules
	# distinguishedName must be specified
	IF NEW.distinguishedName IS NULL OR NEW.distinguishedName = '' THEN
		set error_msg := concat(error_msg, 'Field distinguishedName required ');
	END IF;
	# distinguishedName must be unique
    IF EXISTS (select null from user where distinguishedname = new.distinguishedname) THEN
		SET error_msg := concat(error_msg, 'Field distinguishedName must be unique ');
	END IF;
	IF error_msg <> '' THEN
		SET error_msg := concat(error_msg, 'when inserting record into ', thisTableName);
		signal sqlstate '45000' set message_text = error_msg;
	END IF;
	# Force values on create
	SET NEW.id := ordered_uuid(UUID());
	SET NEW.createdDate := current_timestamp(6);
	SET NEW.modifiedDate := current_timestamp(6);
	SET NEW.modifiedBy := NEW.createdBy;
	SET NEW.changeCount := 0;
	# Standard change token formula
	SET NEW.changeToken := md5(CONCAT(CAST(NEW.ID AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));
	# Archive table
	INSERT INTO
		a_user
	(
		id
		,createdDate
		,createdBy
		,modifiedDate
		,modifiedBy
		,changeCount
		,changeToken
		,distinguishedName
		,displayName
		,email
	) values (
		NEW.ID
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.changeCount
		,NEW.changeToken
		,NEW.distinguishedName
		,NEW.displayName
		,NEW.email
	);
END;
-- +migrate StatementEnd
DROP TRIGGER IF EXISTS tu_user;
-- +migrate StatementBegin
CREATE TRIGGER tu_user
BEFORE UPDATE ON user FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'user';
	# Rules
	# id cannot be changed
	IF (NEW.id <> OLD.id) AND length(error_msg) < 83 THEN
		SET error_msg := concat(error_msg, 'Unable to set id ');
	END IF;
	# createdDate cannot be changed
	IF (NEW.createdDate <> OLD.createdDate) AND length(error_msg) < 74 THEN
		SET error_msg := concat(error_msg, 'Unable to set createdDate ');
	END IF;
	# createdBy cannot be changed
	IF (NEW.createdBy <> OLD.createdBy) AND length(error_msg) < 76 THEN
		SET error_msg := concat(error_msg, 'Unable to set createdBy ');
	END IF;
	# changeCount cannot be changed
	IF (NEW.changeCount <> OLD.changeCount) AND length(error_msg) < 74 THEN
		SET error_msg := concat(error_msg, 'Unable to set changeCount ');
	END IF;
	# changeToken must be given and match the record
	IF (NEW.changeToken IS NULL OR NEW.changeToken = '') and length(error_msg) < 73 THEN
		SET error_msg := concat(error_msg, 'Field changeToken required ');
	END IF;
	IF (NEW.changeToken <> OLD.changeToken) and length(error_msg) < 71 THEN
		SET error_msg := concat(error_msg, 'Field changeToken must match ');
	END IF;
	# distinguishedName cannot be changed
	IF (NEW.distinguishedName <> OLD.distinguishedName) AND length(error_msg) < 68 THEN
		SET error_msg := concat(error_msg, 'Unable to set distinguishedName ');
	END IF;
	IF length(error_msg) > 0 THEN
		SET error_msg := concat(error_msg, 'when updating record');
		signal sqlstate '45000' set message_text = error_msg;
	END IF;
	# Force values on modify
	SET NEW.modifiedDate := current_timestamp(6);
	SET NEW.changeCount := OLD.changeCount + 1;
	# Standard change token formula
	SET NEW.changeToken := md5(CONCAT(CAST(OLD.ID AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));
	# Archive table
	INSERT INTO
		a_user
	(
		id
		,createdDate
		,createdBy
		,modifiedDate
		,modifiedBy
		,changeCount
		,changeToken
		,distinguishedName
		,displayName
		,email
	) values (
		NEW.ID
		,NEW.createdDate
		,NEW.createdBy
		,NEW.modifiedDate
		,NEW.modifiedBy
		,NEW.changeCount
		,NEW.changeToken
		,NEW.distinguishedName
		,NEW.displayName
		,NEW.email
	);
END;
-- +migrate StatementEnd
INSERT INTO migration_status SET description = '20170630_combined create constraints';
DROP PROCEDURE IF EXISTS sp_Patch_20170630_create_constraints;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170630_create_constraints()
proc_label: BEGIN
    -- only do this if not yet 20170630
    IF EXISTS( select null from dbstate where schemaversion = '20170630') THEN
        LEAVE proc_label;
    END IF;
    -- acmgrantee
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on acmgrantee';
    ALTER TABLE acmgrantee
        ADD CONSTRAINT fk_acmgrantee_userdistinguishedname FOREIGN KEY (userdistinguishedname) REFERENCES user(distinguishedname)
        ,ADD KEY ix_acmgrantee_resourcestring (resourcestring);
    -- acmkey2
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on acmkey2';
    ALTER TABLE acmkey2
        ADD KEY ix_acmkey2_name (name);
    -- acmpart2
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on acmpart2';
    ALTER TABLE acmpart2
        ADD CONSTRAINT fk_acmpart2_acmid FOREIGN KEY (acmid) REFERENCES acm2(id)
        ,ADD CONSTRAINT fk_acmpart2_acmkeyid FOREIGN KEY (acmkeyid) REFERENCES acmkey2(id)
        ,ADD CONSTRAINT fk_acmpart2_acmvalueid FOREIGN KEY (acmvalueid) REFERENCES acmvalue2(id);
    -- acmvalue2
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on acmvalue2';
    ALTER TABLE acmvalue2
        ADD KEY ix_acmvalue2_name (name);
    -- object
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on object';
    ALTER TABLE object
        ADD CONSTRAINT fk_object_acmid FOREIGN KEY (acmid) REFERENCES acm2(id)
        ,ADD CONSTRAINT fk_object_createdby FOREIGN KEY (createdby) REFERENCES user(distinguishedname)
        ,ADD CONSTRAINT fk_object_deletedby FOREIGN KEY (deletedby) REFERENCES user(distinguishedname)
        ,ADD CONSTRAINT fk_object_expungedby FOREIGN KEY (expungedby) REFERENCES user(distinguishedname)
        ,ADD CONSTRAINT fk_object_modifiedby FOREIGN KEY (modifiedby) REFERENCES user(distinguishedname)
        ,ADD CONSTRAINT fk_object_ownedbyid FOREIGN KEY (ownedbyid) REFERENCES acmvalue2(id)
        ,ADD CONSTRAINT fk_object_parentid FOREIGN KEY (parentid) REFERENCES object(id)
        ,ADD CONSTRAINT fk_object_typeid FOREIGN KEY (typeid) REFERENCES object_type(id)
        ,ADD KEY ix_object_createddate (createddate)
        ,ADD KEY ix_object_description (description)
        ,ADD KEY ix_object_isdeleted (isdeleted)
        ,ADD KEY ix_object_modifieddate (modifieddate)
        ,ADD KEY ix_object_name (name);
    -- object_permission
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on object_permission';
    ALTER TABLE object_permission 
        ADD CONSTRAINT fk_object_permission_createdby FOREIGN KEY (createdby) REFERENCES user(distinguishedname)
        ,ADD CONSTRAINT fk_object_permission_createdbyid FOREIGN KEY (createdbyid) REFERENCES acmvalue2(id)
        ,ADD CONSTRAINT fk_object_permission_grantee FOREIGN KEY (grantee) REFERENCES acmgrantee(grantee)
        ,ADD CONSTRAINT fk_object_permission_granteeid FOREIGN KEY (granteeid) REFERENCES acmvalue2(id)
        ,ADD CONSTRAINT fk_object_permission_objectid FOREIGN KEY (objectId) REFERENCES object(id)
        ,ADD KEY ix_object_permission_allowread (isdeleted,allowread);
    -- object_property
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on object_property';
    ALTER TABLE object_property
        ADD CONSTRAINT fk_object_property_objectid FOREIGN KEY (objectid) REFERENCES object(id)
        ,ADD CONSTRAINT fk_object_property_propertyid FOREIGN KEY (propertyid) REFERENCES property(id);
    -- object_type
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on object_type';
    ALTER TABLE object_type
        ADD CONSTRAINT fk_object_type_createdby FOREIGN KEY (createdby) REFERENCES user(distinguishedname)
        ,ADD CONSTRAINT fk_object_type_deletedby FOREIGN KEY (deletedby) REFERENCES user(distinguishedname)
        ,ADD CONSTRAINT fk_object_type_modifiedby FOREIGN KEY (modifiedby) REFERENCES user(distinguishedname)
        ,ADD KEY ix_object_type_isdeleted (isdeleted)
        ,ADD KEY ix_object_type_name (name);
    -- object_type_property
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on object_type_property';
    ALTER TABLE object_type_property
        ADD CONSTRAINT fk_object_type_property_propertyid FOREIGN KEY (propertyid) REFERENCES property(id)
        ,ADD CONSTRAINT fk_object_type_property_typeid FOREIGN KEY (typeid) REFERENCES object_type(id)
        ,ADD KEY ix_object_type_property_isdeleted (isdeleted);
    -- property
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on property';
    ALTER TABLE property
        ADD CONSTRAINT fk_property_createdby FOREIGN KEY (createdby) REFERENCES user(distinguishedname)
        ,ADD CONSTRAINT fk_property_deletedby FOREIGN KEY (deletedby) REFERENCES user(distinguishedname)
        ,ADD CONSTRAINT fk_property_modifiedby FOREIGN KEY (modifiedby) REFERENCES user(distinguishedname)
        ,ADD KEY ix_property_isdeleted (isdeleted)
        ,ADD KEY ix_property_name (name);       
    -- useracm
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on useracm';
    ALTER TABLE useracm
        ADD CONSTRAINT fk_useracm_acmid FOREIGN KEY (acmid) REFERENCES acm2(id)
        ,ADD CONSTRAINT fk_useracm_userid FOREIGN KEY (userid) REFERENCES user(id);        
    -- useraocache
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on useraocache';
    ALTER TABLE useraocache
        ADD CONSTRAINT fk_useraocache_userid FOREIGN KEY (userid) REFERENCES user(id);
    -- useraocachepart
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on useraocachepart';
    ALTER TABLE useraocachepart
        ADD CONSTRAINT fk_useraocachepart_userid FOREIGN KEY (userid) REFERENCES user(id)
        ,ADD CONSTRAINT fk_useraocachepart_userkeyid FOREIGN KEY (userkeyid) REFERENCES acmkey2(id)
        ,ADD CONSTRAINT fk_useraocachepart_uservalueid FOREIGN KEY (uservalueid) REFERENCES acmvalue2(id);        
    -- a_object
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on a_object';
    ALTER TABLE a_object
        ADD KEY ix_a_object_changecount (changecount)
        ,ADD KEY ix_a_object_id (id)
        ,ADD KEY ix_a_object_modifieddate (modifieddate);
    -- a_object_permission
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on a_object_permission';
    ALTER TABLE a_object_permission
        ADD KEY ix_a_object_permission_changecount (changecount)
        ,ADD KEY ix_a_object_permission_grantee (grantee)
        ,ADD KEY ix_a_object_permission_id (id)
        ,ADD KEY ix_a_object_permission_modifieddate (modifieddate)
        ,ADD KEY ix_a_object_permission_objectid (objectid);
    -- a_object_type
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on a_object_type';
    ALTER TABLE a_object_type
        ADD KEY ix_a_object_type_changecount (changecount)
        ,ADD KEY ix_a_object_type_id (id)
        ,ADD KEY ix_a_object_type_isdeleted (isdeleted)
        ,ADD KEY ix_a_object_type_modifieddate (modifieddate);
    -- a_property
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on a_property';
    ALTER TABLE a_property
        ADD KEY ix_property_changecount (changecount)
        ,ADD KEY ix_property_id (id)
        ,ADD KEY ix_property_isdeleted (isdeleted)
        ,ADD KEY ix_property_modifieddate (modifieddate)
        ,ADD KEY ix_property_name (name);
    -- a_user
    INSERT INTO migration_status SET description = '20170630_combined creating constraints and indexes on a_user';
    ALTER TABLE a_user
        ADD KEY ix_user_distinguishedname (distinguishedname)
        ,ADD KEY ix_user_modifieddate (modifieddate);
END;
-- +migrate StatementEnd
CALL sp_Patch_20170630_create_constraints();
SET FOREIGN_KEY_CHECKS=1;
DROP PROCEDURE IF EXISTS sp_Patch_20170630_gorp_migrations; 
DROP PROCEDURE IF EXISTS sp_Patch_20170630_drop_keys_and_indexes_raw;
DROP PROCEDURE IF EXISTS sp_Patch_20170630_tablecolumns;   
DROP PROCEDURE IF EXISTS sp_Patch_20170630_transform_acmid;    
DROP PROCEDURE IF EXISTS sp_Patch_20170630_update_acmgrantee;    
DROP PROCEDURE IF EXISTS sp_Patch_20170630_grantee_flattening;
DROP PROCEDURE IF EXISTS sp_Patch_20170630_update_object_table;    
DROP PROCEDURE IF EXISTS sp_Patch_20170630_drop_tables;
DROP PROCEDURE IF EXISTS sp_Patch_20170630_create_constraints;

update dbstate set schemaVersion = '20170630' where schemaVersion <> '20170630';

-- +migrate Down

-- A downgrade from this schema version will not be supported.

INSERT INTO migration_status SET description = '20170630_combined injecting placeholders for prior migrations';
DROP PROCEDURE IF EXISTS sp_Patch_20170630_gorp_migrations;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170630_gorp_migrations()
BEGIN
    IF NOT EXISTS ( select null from gorp_migrations where id = '1_idx_object_description.sql' ) THEN
        insert gorp_migrations set applied_at = current_timestamp(), id = '1_idx_object_description.sql';
    END IF;
    IF NOT EXISTS ( select null from gorp_migrations where id = '2_idx_object_permission.sql' ) THEN
        insert gorp_migrations set applied_at = current_timestamp(), id = '2_idx_object_permission.sql';
    END IF;
    IF NOT EXISTS ( select null from gorp_migrations where id = '3_ownedby_fk_and_triggers.sql' ) THEN
        insert gorp_migrations set applied_at = current_timestamp(), id = '3_ownedby_fk_and_triggers.sql';
    END IF;
    IF NOT EXISTS ( select null from gorp_migrations where id = '4_indexes_on_object.sql' ) THEN
        insert gorp_migrations set applied_at = current_timestamp(), id = '4_indexes_on_object.sql';
    END IF;
    IF NOT EXISTS ( select null from gorp_migrations where id = '20161223_355_triggers_timestamp.sql' ) THEN
        insert gorp_migrations set applied_at = current_timestamp(), id = '20161223_355_triggers_timestamp.sql';
    END IF;
    IF NOT EXISTS ( select null from gorp_migrations where id = '20161230_140_object_pathing.sql' ) THEN
        insert gorp_migrations set applied_at = current_timestamp(), id = '20161230_140_object_pathing.sql';
    END IF;
    IF NOT EXISTS ( select null from gorp_migrations where id = '20170301_576_path_delimiter.sql' ) THEN
        insert gorp_migrations set applied_at = current_timestamp(), id = '20170301_576_path_delimiter.sql';
    END IF;
    IF NOT EXISTS ( select null from gorp_migrations where id = '20170331_409_ao_acm_performance.sql' ) THEN
        insert gorp_migrations set applied_at = current_timestamp(), id = '20170331_409_ao_acm_performance.sql';
    END IF;
    IF NOT EXISTS ( select null from gorp_migrations where id = '20170421_409_aacflatten.sql' ) THEN
        insert gorp_migrations set applied_at = current_timestamp(), id = '20170421_409_aacflatten.sql';
    END IF;
    IF NOT EXISTS ( select null from gorp_migrations where id = '20170505_fix_calcResourceString.sql' ) THEN
        insert gorp_migrations set applied_at = current_timestamp(), id = '20170505_fix_calcResourceString.sql';
    END IF;
    IF NOT EXISTS ( select null from gorp_migrations where id = '20170508_409_permissiongrantee.sql' ) THEN
        insert gorp_migrations set applied_at = current_timestamp(), id = '20170508_409_permissiongrantee.sql';
    END IF;
    IF NOT EXISTS ( select null from gorp_migrations where id = '20170630_combined.sql' ) THEN
        insert gorp_migrations set applied_at = current_timestamp(), id = '20170630_combined.sql';
    END IF;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170630_gorp_migrations();
DROP PROCEDURE IF EXISTS sp_Patch_20170630_gorp_migrations; 




update dbstate set schemaVersion = '20170630' where schemaVersion <> '20170630';