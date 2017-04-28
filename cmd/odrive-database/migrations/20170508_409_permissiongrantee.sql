-- +migrate Up

-- remove existing triggers on object for insert/update so migration doesnt create erroneous versions
DROP TRIGGER IF EXISTS ti_object_permission;
DROP TRIGGER IF EXISTS tu_object_permission;

-- add column createdbyid and granteeid to object_permission and a_object_permission tables, 
-- along with foreign key constraint, done in a procedure to check if exists
INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql adding granteeid and createdbyid to object_permission table';
DROP PROCEDURE IF EXISTS sp_Patch_20170508_tables_permission;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170508_tables_permission()
BEGIN
    IF NOT EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'object_permission' and column_name = 'createdbyid') THEN
        ALTER TABLE object_permission ADD COLUMN createdbyid int unsigned null;
        ALTER TABLE object_permission ADD CONSTRAINT fk_object_permission_createdbyid FOREIGN KEY (createdbyid) REFERENCES acmvalue2(id);
    END IF;
    IF NOT EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'a_object_permission' and column_name = 'createdbyid') THEN
        ALTER TABLE a_object_permission ADD COLUMN createdbyid int unsigned null;
    END IF;
    IF NOT EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'object_permission' and column_name = 'granteeid') THEN
        ALTER TABLE object_permission ADD COLUMN granteeid int unsigned null;
        ALTER TABLE object_permission ADD CONSTRAINT fk_object_permission_granteeid FOREIGN KEY (granteeid) REFERENCES acmvalue2(id);
    END IF;
    IF NOT EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'a_object_permission' and column_name = 'granteeid') THEN
        ALTER TABLE a_object_permission ADD COLUMN granteeid int unsigned null;
    END IF;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170508_tables_permission();
DROP PROCEDURE IF EXISTS sp_Patch_20170508_tables_permission; 

-- Migrate existing permissions, determining acmvalue2 identifiers based upon createdby
INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql creating procedure to populate createdbyid';
DROP PROCEDURE IF EXISTS sp_Patch_20170508_transform_permissions;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170508_transform_permissions()
BEGIN
    INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql assign createdbyid from createdby';
    ASSIGNCREATEDBYID: BEGIN
        DECLARE vCreatedByID int unsigned default 0;
        DECLARE vCreatedBy varchar(255) default '';
        DECLARE c_createdby_finished int default 0;
        DECLARE c_createdby cursor for SELECT distinct createdby FROM object_permission;
        DECLARE continue handler for not found set c_createdby_finished = 1;
        OPEN c_createdby;
        get_createdby: LOOP
            FETCH c_createdby INTO vCreatedBy;
            IF c_createdby_finished = 1 THEN
                CLOSE c_createdby;
                LEAVE get_createdby;
            END IF;
			SELECT id FROM acmvalue2 WHERE name = aacflatten(vCreatedBy) INTO vCreatedByID;
			UPDATE a_object_permission SET createdbyid = vCreatedByID WHERE createdby = vCreatedBy;
			UPDATE object_permission SET createdbyid = vCreatedByID WHERE createdby = vCreatedBy;
        END LOOP get_createdby;
    END ASSIGNCREATEDBYID;   
END;
-- +migrate StatementEnd
CALL sp_Patch_20170508_transform_permissions();
DROP PROCEDURE IF EXISTS sp_Patch_20170508_transform_permissions;   

-- Migrate existing permissions, determining acmvalue2 identifiers based upon grantee
INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql creating procedure to populate granteeid';
DROP PROCEDURE IF EXISTS sp_Patch_20170508_transform_permissions;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170508_transform_permissions()
BEGIN
    INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql assign granteeid from grantee';
    ASSIGNGRANTEEID: BEGIN
        DECLARE vGranteeID int unsigned default 0;
        DECLARE vGrantee varchar(255) default '';
        DECLARE c_grantee_finished int default 0;
        DECLARE c_grantee cursor for SELECT distinct grantee FROM object_permission;
        DECLARE continue handler for not found set c_grantee_finished = 1;
        OPEN c_grantee;
        get_grantee: LOOP
            FETCH c_grantee INTO vGrantee;
            IF c_grantee_finished = 1 THEN
                CLOSE c_grantee;
                LEAVE get_grantee;
            END IF;
			SELECT id FROM acmvalue2 WHERE name = aacflatten(vGrantee) INTO vGranteeID;
			UPDATE a_object_permission SET granteeid = vGranteeID WHERE grantee = vGrantee;
			UPDATE object_permission SET granteeid = vGranteeID WHERE grantee = vGrantee;
        END LOOP get_grantee;
    END ASSIGNGRANTEEID;   
END;
-- +migrate StatementEnd
CALL sp_Patch_20170508_transform_permissions();
DROP PROCEDURE IF EXISTS sp_Patch_20170508_transform_permissions; 

INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql reindexing object_permission';
INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql removing constraints and indexes from object_permission';
DROP PROCEDURE IF EXISTS sp_Patch_20170508_constraints_permission;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170508_constraints_permission()
BEGIN
	IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_deletedBy') THEN
		ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_deletedBy`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_deletedBy') THEN
		ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_deletedBy`;
	END IF;
	IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_modifiedBy') THEN
		ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_modifiedBy`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_modifiedBy') THEN
		ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_modifiedBy`;
	END IF;
	IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_granteeid') THEN
		ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_granteeid`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_granteeid') THEN
		ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_granteeid`;
	END IF;
	IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_createdBy') THEN
		ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_createdBy`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_createdBy') THEN
		ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_createdBy`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'ix_object_permission_createdby') THEN
		ALTER TABLE `object_permission` DROP INDEX `ix_object_permission_createdby`;
	END IF;
	IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_createdbyid') THEN
		ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_createdbyid`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_createdbyid') THEN	
		ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_createdbyid`;
	END IF;
	IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_grantee') THEN
		ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_grantee`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'ix_grantee') THEN
		ALTER TABLE `object_permission` DROP INDEX `ix_grantee`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_grantee') THEN
		ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_grantee`;
	END IF;
	IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_objectId') THEN
		ALTER TABLE `object_permission` DROP FOREIGN KEY fk_object_permission_objectId;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_objectId') THEN
		ALTER TABLE `object_permission` DROP INDEX fk_object_permission_objectId;
	END IF;
	IF EXISTS (select null from information_schema.table_constraints where table_name = 'object_permission' and binary constraint_name = 'fk_object_permission_objectid') THEN
		ALTER TABLE `object_permission` DROP FOREIGN KEY fk_object_permission_objectid;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'fk_object_permission_objectid') THEN
		ALTER TABLE `object_permission` DROP INDEX fk_object_permission_objectid;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'ix_objectId') THEN
		ALTER TABLE `object_permission` DROP INDEX `ix_objectId`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and binary index_name = 'ix_isDeleted') THEN
		ALTER TABLE `object_permission` DROP INDEX `ix_isDeleted`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object_permission' and index_name like 'ix_object_permission_allowread') THEN
		ALTER TABLE `object_permission` DROP INDEX `ix_object_permission_allowread`;
	END IF;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170508_constraints_permission();
DROP PROCEDURE IF EXISTS sp_Patch_20170508_constraints_permission;
INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql creating constraints and indexes on object_permission';
ALTER TABLE object_permission 
	ADD CONSTRAINT fk_object_permission_granteeid FOREIGN KEY (granteeid) REFERENCES acmvalue2(id)
	,ADD CONSTRAINT fk_object_permission_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_object_permission_createdbyid FOREIGN KEY (createdbyid) REFERENCES acmvalue2(id)
	,ADD CONSTRAINT fk_object_permission_grantee FOREIGN KEY (grantee) REFERENCES acmgrantee(grantee)
	,ADD CONSTRAINT fk_object_permission_objectId FOREIGN KEY (objectId) REFERENCES object(id)
	,ADD KEY ix_object_permission_allowread (isdeleted,allowread);

INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql reindexing object';
INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql removing constraints and indexes from object';
DROP PROCEDURE IF EXISTS sp_Patch_20170508_constraints_object;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170508_constraints_object()
BEGIN
	IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_acmid') THEN
		ALTER TABLE `object` DROP FOREIGN KEY `fk_object_acmid`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_acmid') THEN
		ALTER TABLE `object` DROP INDEX `fk_object_acmid`;
	END IF;
	IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_createdBy') THEN
		ALTER TABLE `object` DROP FOREIGN KEY `fk_object_createdBy`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_createdBy') THEN
		ALTER TABLE `object` DROP INDEX `fk_object_createdBy`;
	END IF;
	IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_deletedBy') THEN
		ALTER TABLE `object` DROP FOREIGN KEY `fk_object_deletedBy`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_deletedBy') THEN
		ALTER TABLE `object` DROP INDEX `fk_object_deletedBy`;
	END IF;
	IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_expungedBy') THEN
		ALTER TABLE `object` DROP FOREIGN KEY `fk_object_expungedBy`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_expungedBy') THEN
		ALTER TABLE `object` DROP INDEX `fk_object_expungedBy`;
	END IF;
	IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_modifiedBy') THEN
		ALTER TABLE `object` DROP FOREIGN KEY `fk_object_modifiedBy`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_modifiedBy') THEN
		ALTER TABLE `object` DROP INDEX `fk_object_modifiedBy`;
	END IF;
	IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_ownedbyid') THEN
		ALTER TABLE `object` DROP FOREIGN KEY `fk_object_ownedbyid`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_ownedbyid') THEN
		ALTER TABLE `object` DROP INDEX `fk_object_ownedbyid`;
	END IF;
	IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_parentId') THEN
		ALTER TABLE `object` DROP FOREIGN KEY `fk_object_parentId`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_parentId') THEN
		ALTER TABLE `object` DROP INDEX `fk_object_parentId`;
	END IF;
	IF EXISTS (select null from information_schema.table_constraints where table_name = 'object' and binary constraint_name = 'fk_object_typeId') THEN
		ALTER TABLE `object` DROP FOREIGN KEY `fk_object_typeId`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_typeId') THEN
		ALTER TABLE `object` DROP INDEX `fk_object_typeId`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_createdDate') THEN
		ALTER TABLE `object` DROP INDEX `ix_createdDate`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_modifiedDate') THEN
		ALTER TABLE `object` DROP INDEX `ix_modifiedDate`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_ownedBy') THEN
		ALTER TABLE `object` DROP INDEX `ix_ownedBy`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'idx_object_description') THEN
		ALTER TABLE `object` DROP INDEX `idx_object_description`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'fk_object_ownedByNew') THEN
		ALTER TABLE `object` DROP INDEX `fk_object_ownedByNew`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_name') THEN
		ALTER TABLE `object` DROP INDEX `ix_name`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_isDeleted') THEN
		ALTER TABLE `object` DROP INDEX `ix_isDeleted`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_object_createdDate') THEN
		ALTER TABLE `object` DROP INDEX `ix_object_createdDate`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_object_modifiedDate') THEN
		ALTER TABLE `object` DROP INDEX `ix_object_modifiedDate`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_object_name') THEN
		ALTER TABLE `object` DROP INDEX `ix_object_name`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_object_isdeleted') THEN
		ALTER TABLE `object` DROP INDEX `ix_object_isdeleted`;
	END IF;
	IF EXISTS (select null from information_schema.statistics where table_name = 'object' and binary index_name = 'ix_object_description') THEN
		ALTER TABLE `object` DROP INDEX `ix_object_description`;
	END IF;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170508_constraints_object();
DROP PROCEDURE IF EXISTS sp_Patch_20170508_constraints_object;
INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql creating constraints and indexes on object';
ALTER TABLE object
	ADD CONSTRAINT fk_object_acmid FOREIGN KEY (acmid) REFERENCES acm2(id)
	,ADD CONSTRAINT fk_object_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)	-- may not be needed, filters for user root rely on ownership
	,ADD CONSTRAINT fk_object_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)	-- may not be needed, filters for trash rely on ownership
	,ADD CONSTRAINT fk_object_expungedBy FOREIGN KEY (expungedBy) REFERENCES user(distinguishedName)	-- may not be needed, no reference
	,ADD CONSTRAINT fk_object_modifiedBy FOREIGN KEY (modifiedBy) REFERENCES user(distinguishedName)	-- may not be needed, no reference
	,ADD CONSTRAINT fk_object_ownedbyid FOREIGN KEY (ownedbyid) REFERENCES acmvalue2(id)
	,ADD CONSTRAINT fk_object_parentId FOREIGN KEY (parentid) REFERENCES object(id)
	,ADD CONSTRAINT fk_object_typeId FOREIGN KEY (typeId) REFERENCES object_type(id)
	,ADD KEY ix_object_createdDate (createddate)
	,ADD KEY ix_object_modifiedDate (modifieddate)
	,ADD KEY ix_object_name (name)
	,ADD KEY ix_object_isdeleted (isdeleted)
	,ADD KEY ix_object_description (description);

INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql recreating ti_object_permission';
-- +migrate StatementBegin
CREATE TRIGGER ti_object_permission
BEFORE INSERT ON object_permission FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'object_permission';
	DECLARE count_object int default 0;
	DECLARE vCreatedByID int unsigned default 0;
	DECLARE vGranteeID int unsigned default 0;

	# Rules
	# ObjectId must be specified
	SELECT COUNT(*) FROM object WHERE id = NEW.objectId INTO count_object;
	IF count_object = 0 THEN
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

	# Specific field level changes
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'objectId', newValue = hex(NEW.objectId);
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'grantee', newValue = NEW.grantee;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmShare', newTextValue = NEW.acmShare;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowCreate', newValue = NEW.allowCreate;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowRead', newValue = NEW.allowRead;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowUpdate', newValue = NEW.allowUpdate;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowDelete', newValue = NEW.allowDelete;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowShare', newValue = NEW.allowShare;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'explicitShare', newValue = NEW.explicitShare;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'encryptKey', newValue = hex(NEW.encryptKey);
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'permissionIV', newValue = hex(NEW.permissionIV);
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'permissionMAC', newValue = hex(NEW.permissionMAC);

END;
-- +migrate StatementEnd

INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql recreating tu_object_permission';
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

	# Specific field level changes
	IF NEW.objectId <> OLD.objectId THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'objectId', newValue = hex(NEW.objectId);
	END IF;
	IF NEW.grantee <> OLD.grantee THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'grantee', newValue = NEW.grantee;
	END IF;
	IF NEW.acmShare <> OLD.acmShare THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmShare', newTextValue = NEW.acmShare;
	END IF;
	IF NEW.allowCreate <> OLD.allowCreate THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowCreate', newValue = NEW.allowCreate;
	END IF;
	IF NEW.allowRead <> OLD.allowRead THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowRead', newValue = NEW.allowRead;
	END IF;
	IF NEW.allowUpdate <> OLD.allowUpdate THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowUpdate', newValue = NEW.allowUpdate;
	END IF;
	IF NEW.allowDelete <> OLD.allowDelete THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowDelete', newValue = NEW.allowDelete;
	END IF;
	IF NEW.allowShare <> OLD.allowShare THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowShare', newValue = NEW.allowShare;
	END IF;
	IF NEW.explicitShare <> OLD.explicitShare THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'explicitShare', newValue = NEW.explicitShare;
	END IF;
	IF NEW.encryptKey <> OLD.encryptKey THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'encryptKey', newValue = hex(NEW.encryptKey);
	END IF;
	IF NEW.permissionIV <> OLD.permissionIV THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'permissionIV', newValue = hex(NEW.permissionIV);
	END IF;
	IF NEW.permissionMAC <> OLD.permissionMAC THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'permissionMAC', newValue = hex(NEW.permissionMAC);
	END IF;

END;
-- +migrate StatementEnd

-- dbstate
DROP TRIGGER IF EXISTS ti_dbstate;
INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql setting schema version';
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
	SET NEW.schemaversion := '20170508'; 
	# Identifier is randomized as a GUID
	SET NEW.identifier := concat(@@hostname, '-', left(uuid(),8));
END;
-- +migrate StatementEnd
update dbstate set schemaVersion = '20170508';

-- +migrate Down

-- remove triggers on object for create/update
DROP TRIGGER IF EXISTS ti_object_permission;
DROP TRIGGER IF EXISTS tu_object_permission;

-- remove column createdbyid and granteeid to object_permission and a_object_permission tables, 
-- along with foreign key constraint, done in a procedure to check if exists
INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql removing createdbyid and granteeid from object_permission table';
DROP PROCEDURE IF EXISTS sp_Patch_20170508_tables_permission;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170508_tables_permission()
BEGIN
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'object_permission' and column_name = 'createdbyid') THEN
        ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_createdbyid`;
        ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_createdbyid`;
        ALTER TABLE object_permission DROP COLUMN createdbyid;
    END IF;
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'object_permission' and column_name = 'granteeid') THEN
        ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_granteeid`;
        ALTER TABLE `object_permission` DROP INDEX `fk_object_permission_granteeid`;
        ALTER TABLE object_permission DROP COLUMN granteeid;
    END IF;
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'a_object_permission' and column_name = 'createdbyid') THEN
        ALTER TABLE a_object_permission DROP COLUMN createdbyid;
    END IF;
    IF EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'a_object_permission' and column_name = 'granteeid') THEN
        ALTER TABLE a_object_permission DROP COLUMN granteeid;
    END IF;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170508_tables_permission();
DROP PROCEDURE IF EXISTS sp_Patch_20170508_tables_permission;

INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql restoring ti_object_permission';
-- +migrate StatementBegin
CREATE TRIGGER ti_object_permission
BEFORE INSERT ON object_permission FOR EACH ROW
BEGIN
	DECLARE error_msg varchar(128) default '';
	DECLARE thisTableName varchar(128) default 'object_permission';
	DECLARE count_object int default 0;

	# Rules
	# ObjectId must be specified
	SELECT COUNT(*) FROM object WHERE id = NEW.objectId INTO count_object;
	IF count_object = 0 THEN
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
	);

	# Specific field level changes
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'objectId', newValue = hex(NEW.objectId);
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'grantee', newValue = NEW.grantee;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmShare', newTextValue = NEW.acmShare;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowCreate', newValue = NEW.allowCreate;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowRead', newValue = NEW.allowRead;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowUpdate', newValue = NEW.allowUpdate;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowDelete', newValue = NEW.allowDelete;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowShare', newValue = NEW.allowShare;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'explicitShare', newValue = NEW.explicitShare;
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'encryptKey', newValue = hex(NEW.encryptKey);
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'permissionIV', newValue = hex(NEW.permissionIV);
	INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'permissionMAC', newValue = hex(NEW.permissionMAC);

END;
-- +migrate StatementEnd
INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql restoring tu_object_permission';
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
	);

	# Specific field level changes
	IF NEW.objectId <> OLD.objectId THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'objectId', newValue = hex(NEW.objectId);
	END IF;
	IF NEW.grantee <> OLD.grantee THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'grantee', newValue = NEW.grantee;
	END IF;
	IF NEW.acmShare <> OLD.acmShare THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmShare', newTextValue = NEW.acmShare;
	END IF;
	IF NEW.allowCreate <> OLD.allowCreate THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowCreate', newValue = NEW.allowCreate;
	END IF;
	IF NEW.allowRead <> OLD.allowRead THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowRead', newValue = NEW.allowRead;
	END IF;
	IF NEW.allowUpdate <> OLD.allowUpdate THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowUpdate', newValue = NEW.allowUpdate;
	END IF;
	IF NEW.allowDelete <> OLD.allowDelete THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowDelete', newValue = NEW.allowDelete;
	END IF;
	IF NEW.allowShare <> OLD.allowShare THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'allowShare', newValue = NEW.allowShare;
	END IF;
	IF NEW.explicitShare <> OLD.explicitShare THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'explicitShare', newValue = NEW.explicitShare;
	END IF;
	IF NEW.encryptKey <> OLD.encryptKey THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'encryptKey', newValue = hex(NEW.encryptKey);
	END IF;
	IF NEW.permissionIV <> OLD.permissionIV THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'permissionIV', newValue = hex(NEW.permissionIV);
	END IF;
	IF NEW.permissionMAC <> OLD.permissionMAC THEN
		INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'permissionMAC', newValue = hex(NEW.permissionMAC);
	END IF;

END;
-- +migrate StatementEnd

-- dbstate
DROP TRIGGER IF EXISTS ti_dbstate;
INSERT INTO migration_status SET description = '20170508_409_permissiongrantee.sql setting schema version';
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
	SET NEW.schemaversion := '20170505'; 
	# Identifier is randomized as a GUID
	SET NEW.identifier := concat(@@hostname, '-', left(uuid(),8));
END;
-- +migrate StatementEnd
update dbstate set schemaVersion = '20170505';