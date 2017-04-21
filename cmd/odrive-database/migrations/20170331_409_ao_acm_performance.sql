-- +migrate Up

-- remove existing triggers on object for insert/update so migration doesnt create erroneous versions
DROP TRIGGER IF EXISTS ti_object;
DROP TRIGGER IF EXISTS tu_object;
DROP TRIGGER IF EXISTS ti_object_permission;
DROP TRIGGER IF EXISTS tu_object_permission;

-- new tables
CREATE TABLE IF NOT EXISTS migration_status
(
    id int unsigned not null auto_increment
    ,description varchar(255)
    ,CONSTRAINT pk_migration_status PRIMARY KEY (id)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;

INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql creating tables';

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
    ,INDEX ix_acmvalue2_name (name)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;

CREATE TABLE IF NOT EXISTS acmvalue2
(
    id int unsigned not null auto_increment
    ,name varchar(255) not null
    ,CONSTRAINT pk_acmvalue2 PRIMARY KEY (id)
    ,INDEX ix_acmvalue2_name (name)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;

CREATE TABLE IF NOT EXISTS acmpart2
(
    id int unsigned not null auto_increment
    ,acmid int unsigned not null
    ,acmkeyid int unsigned not null
    ,acmvalueid int unsigned null
    ,CONSTRAINT pk_acmpart2 PRIMARY KEY (id)
    ,CONSTRAINT fk_acmpart2_acmid FOREIGN KEY (acmid) REFERENCES acm2(id)
    ,CONSTRAINT fk_acmpart2_acmkeyid FOREIGN KEY (acmkeyid) REFERENCES acmkey2(id)
    ,CONSTRAINT fk_acmpart2_acmvalueid FOREIGN KEY (acmvalueid) REFERENCES acmvalue2(id)
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
    ,CONSTRAINT fk_useraocachepart_userid FOREIGN KEY (userid) REFERENCES user(id)
    ,CONSTRAINT fk_useraocachepart_userkeyid FOREIGN KEY (userkeyid) REFERENCES acmkey2(id)
    ,CONSTRAINT fk_useraocachepart_uservalueid FOREIGN KEY (uservalueid) REFERENCES acmvalue2(id)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;

CREATE TABLE IF NOT EXISTS useraocache
(
    id int unsigned not null auto_increment
    ,userid binary(16) not null
    ,isCaching tinyint not null default 1
    ,cacheDate timestamp(6) not null default current_timestamp
    ,sha256hash char(64) not null
    ,CONSTRAINT pk_useraocache PRIMARY KEY (id)
    ,CONSTRAINT fk_useraocache_userid FOREIGN KEY (userid) REFERENCES user(id)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;

CREATE TABLE IF NOT EXISTS useracm
(
    id int unsigned not null auto_increment
    ,userid binary(16) not null
    ,acmid int unsigned not null
    ,CONSTRAINT pk_useracm PRIMARY KEY (id)
    ,CONSTRAINT fk_useracm_userid FOREIGN KEY (userid) REFERENCES user(id)
    ,CONSTRAINT fk_useracm_acmid FOREIGN KEY (acmid) REFERENCES acm2(id)
) DEFAULT CHARACTER SET utf8 COLLATE utf8_unicode_ci
;

-- add column acmid to object and a_object tables, 
-- along with foreign key constraint, done in a procedure to check if exists
INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql adding acmid to object table';
DROP PROCEDURE IF EXISTS sp_Patch_20170331_tables_acmid;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170331_tables_acmid()
BEGIN
    IF NOT EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'object' and column_name = 'acmid') THEN
        ALTER TABLE object ADD COLUMN acmid int unsigned null;
        ALTER TABLE object ADD CONSTRAINT fk_object_acmid FOREIGN KEY (acmid) REFERENCES acm2(id);
    END IF;
    IF NOT EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'a_object' and column_name = 'acmid') THEN
        ALTER TABLE a_object ADD COLUMN acmid int unsigned null;
    END IF;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170331_tables_acmid();
DROP PROCEDURE IF EXISTS sp_Patch_20170331_tables_acmid;    

-- Migrate existing objects, tranforming acm/part/key/value into simpler non archived structs
-- identify association needed for revisions and object to do by acmid isntead of via objectacm
-- table
INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql creating procedure to transform acmid';
DROP PROCEDURE IF EXISTS sp_Patch_20170331_transform_acmid;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170331_transform_acmid()
BEGIN

    INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql migrate acm to populate acm2 tables';
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
                INSERT INTO migration_status SET description = concat('20170331_409_ao_acm_performance.sql migrate acm to populate acm2 tables (', ACCMIGRATECOUNT, ' of ', ACMMIGRATETOTAL, ')');
            END IF;
            -- value
            IF (SELECT 1=1 FROM acmvalue2 WHERE name = vValueName) IS NULL THEN
                INSERT INTO acmvalue2 (name) VALUES (vValueName);
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

    INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql assign acmid from acm';
    ASSIGNACMID: BEGIN
        DECLARE ASSIGNACMIDCOUNT int default 0;
        DECLARE ASSIGNACMIDTOTAL int default 0;
        DECLARE vObjectID binary(16) default 0;
        DECLARE vACMID int default 0;
        DECLARE c_object_finished int default 0;
        DECLARE c_object cursor for SELECT id FROM object WHERE acmid is null or acmid = 0;
        DECLARE continue handler for not found set c_object_finished = 1;
        OPEN c_object;
        SELECT COUNT(0) INTO ASSIGNACMIDTOTAL 
            FROM object where acmid is null or acmid = 0;
        get_object: LOOP
            FETCH c_object INTO vObjectID;
            IF c_object_finished = 1 THEN
                CLOSE c_object;
                LEAVE get_object;
            END IF;
            SET ASSIGNACMIDCOUNT := ASSIGNACMIDCOUNT + 1;
            IF floor(ASSIGNACMIDCOUNT/5000) = ceiling(ASSIGNACMIDCOUNT/5000) THEN
                INSERT INTO migration_status SET description = concat('20170331_409_ao_acm_performance.sql assign acmid from acm (', ASSIGNACMIDCOUNT, ' of ', ASSIGNACMIDTOTAL, ')');
            END IF;            
            REVISIONS: BEGIN
                DECLARE vAID int default 0;
                DECLARE vPrevAID int default -1;
                DECLARE vChangeCount int default 0;
                DECLARE vPrevChangeCount int default -1;
                DECLARE vFlattenedACM text default '';
                DECLARE c_revision_finished int default 0;
                DECLARE c_revision cursor FOR 
                    SELECT d.id, d.changecount, d.acm FROM (
                        SELECT a_id id, changecount, modifieddate, '' acm
                        FROM a_object 
                        WHERE id = vObjectID
                        UNION ALL
                        SELECT -1 a_id, -1 id, oa.modifieddate, a.name acm
                        FROM a_objectacm oa INNER JOIN acm a on oa.acmid = a.id
                        WHERE oa.objectid = vObjectID
                    ) AS d ORDER BY d.modifieddate;
                DECLARE continue handler for not found set c_revision_finished = 1;
                SET vACMID := 0;
                OPEN c_revision;
                get_revision: LOOP
                    FETCH c_revision INTO vAID, vChangeCount, vFlattenedACM;
                    IF c_revision_finished = 1 THEN
                        CLOSE c_revision;
                        LEAVE get_revision;
                    END IF;
                    -- Row represents Object Revision
                    IF vAID <> -1 AND vChangeCount <> -1 THEN
                        SET vPrevAID := vAID;
                        SET vPrevChangeCount := vChangeCount;
                        IF vACMID > 0 THEN
                            UPDATE a_object SET acmid = vACMID WHERE a_id = vPrevAID AND changecount = vPrevChangeCount;
                        END IF;
                    END IF;
                    -- Row represents ACM changed
                    IF length(vFlattenedACM) > 0 THEN
                        SELECT id INTO vACMID FROM acm2 WHERE flattenedacm = vFlattenedACM LIMIT 1;
                        UPDATE a_object SET acmid = vACMID WHERE a_id = vPrevAID AND changecount = vPrevChangeCount;
                    END IF;
                END LOOP get_revision;
            END REVISIONS;
            UPDATE object SET acmid = vACMID WHERE id = vObjectID;
        END LOOP get_object;
    END ASSIGNACMID;    
END
-- +migrate StatementEnd
INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql running procedure to transform acmid';
CALL sp_Patch_20170331_transform_acmid();
DROP PROCEDURE IF EXISTS sp_Patch_20170331_transform_acmid;    

-- add column ownedbyid to object and a_object tables, 
-- along with foreign key constraint, done in a procedure to check if exists
INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql adding ownedbyid to object table';
DROP PROCEDURE IF EXISTS sp_Patch_20170331_tables_ownedbyid;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170331_tables_ownedbyid()
BEGIN
    IF NOT EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'object' and column_name = 'ownedbyid') THEN
        ALTER TABLE object ADD COLUMN ownedbyid int unsigned null;
        ALTER TABLE object ADD CONSTRAINT fk_object_ownedbyid FOREIGN KEY (ownedbyid) REFERENCES acmvalue2(id);
    END IF;
    IF NOT EXISTS ( select null from information_schema.columns where table_schema = database() and table_name = 'a_object' and column_name = 'ownedbyid') THEN
        ALTER TABLE a_object ADD COLUMN ownedbyid int unsigned null;
    END IF;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170331_tables_ownedbyid();
DROP PROCEDURE IF EXISTS sp_Patch_20170331_tables_ownedbyid;    

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

-- If a value needs altered in object_permission, what do we need to do to fix the hash? and does it impact the file ?
INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql fixing invalid grantee in acmgrantee';
DROP PROCEDURE IF EXISTS sp_Patch_20170331_grantee_flattening;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170331_grantee_flattening()
BEGIN
    IF EXISTS ( select null from information_schema.table_constraints where constraint_schema = database() and table_name = 'object_permission' and constraint_name = 'fk_object_permission_grantee') THEN
        INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql fixing invalid grantee in acmgrantee. removing fk_object_permission_grantee constraint';
        ALTER TABLE `object_permission` DROP FOREIGN KEY `fk_object_permission_grantee`;
    END IF;
    IF EXISTS ( select null from information_schema.statistics where table_schema = database() and table_name = 'object_permission' and index_name = 'ix_grantee') THEN
        INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql fixing invalid grantee in acmgrantee. removing ix_grantee index';
        ALTER TABLE `object_permission` DROP INDEX `ix_grantee`;
    END IF;
    INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql fixing invalid grantee in acmgrantee. updating object_permission';
    UPDATE object_permission SET grantee = lower(aacflatten(grantee)), permissionMac = new_keymac('${OD_ENCRYPT_MASTERKEY}',lower(aacflatten(grantee)),allowCreate,allowRead,allowUpdate,allowDelete,allowShare,hex(encryptKey)) WHERE grantee <> lower(aacflatten(grantee));
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
        INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql fixing invalid grantee in acmgrantee. checking for duplicate acmgrantees as identified by lower(aacflatten(grantee))';        
        OPEN c_acmgrantee;
        get_acmgrantee: LOOP
            FETCH c_acmgrantee INTO vFlattenedGrantee, vGrantee, vResourceString;
            IF c_acmgrantee_finished = 1 THEN
                CLOSE c_acmgrantee;
                LEAVE get_acmgrantee;
            END IF;
            IF vPrevFlattenedGrantee = vFlattenedGrantee THEN
                -- seen before, lets get rid of it.  We're indescriminate about its quality
                INSERT INTO migration_status SET description = concat('20170331_409_ao_acm_performance.sql removing duplicate record in acmgrantee, grantee=', vGrantee, ', resourceString=', vResourceString);                
                DELETE FROM acmgrantee WHERE grantee = vGrantee AND resourceString = vResourceString;
            END IF;
            SET vPrevFlattenedGrantee := vFlattenedGrantee;
        END LOOP get_acmgrantee;
    END DEDUPEACMGRANTEE;
    INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql fixing invalid grantee in acmgrantee. updating acmgrantee';    
    UPDATE acmgrantee SET grantee = lower(aacflatten(grantee)) WHERE grantee <> lower(aacflatten(grantee));
    INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql fixing invalid grantee in acmgrantee. adding foreign key and index back to object_permission';    
    ALTER TABLE object_permission
        ADD CONSTRAINT fk_object_permission_grantee FOREIGN KEY (grantee) REFERENCES acmgrantee(grantee)
        ,ADD INDEX ix_grantee (grantee)
        ;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170331_grantee_flattening();
DROP PROCEDURE IF EXISTS sp_Patch_20170331_grantee_flattening;

-- a function to add resource string to acmgrantee if not yet present
-- this function will break up the parts of a resource string and check if its in acmgrantee
-- and get normalized resource string from acmgrantee if parsed grantee value is already
-- present but resourcestring value differs.
INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql creating function calcResourceString';
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
    DECLARE vResourceString varchar(300) default '';
    DECLARE vGrantee varchar(255) default '';
    DECLARE vDisplayName varchar(255) default '';

    SELECT (length(vOriginalString)-length(replace(vOriginalString,'/',''))) + 1 INTO vParts;
    SELECT substring_index(vOriginalString,'/',1) INTO vPart1;
    SELECT substring_index(substring_index(vOriginalString,'/',2),'/',-1) INTO vPart2;
    SELECT substring_index(substring_index(vOriginalString,'/',3),'/',-1) INTO vPart3;
    SELECT substring_index(substring_index(vOriginalString,'/',4),'/',-1) INTO vPart4;
    SELECT substring_index(substring_index(vOriginalString,'/',5),'/',-1) INTO vPart5;

    IF vParts > 1 AND (vPart1 = 'user' or vPart1 = 'group') THEN
        -- Calculate resource string and grantee, check if exists in acmgrantee, inserting as needed
        IF vPart1 = 'user' THEN
            SET vResourceString := CONCAT(vPart1, '/', vPart2);
            SET vGrantee := aacflatten(vPart2);
            IF (SELECT 1=1 FROM acmgrantee WHERE resourcestring = vResourceString) IS NULL THEN
                IF (select 1=1 FROM acmgrantee WHERE grantee = vGrantee) IS NULL THEN
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
                    SELECT resourcestring INTO vResourceString FROM acmgrantee WHERE grantee = vGrantee LIMIT 1;
                END IF;
            END IF;
        END IF;
        IF vPart1 = 'group' THEN
            IF vParts <= 3 THEN
                -- Pseudo group (i.e., Everyone)
                SET vResourceString := CONCAT(vPart1, '/', vPart2);
                SET vGrantee := aacflatten(vPart2);
                IF (SELECT 1=1 FROM acmgrantee WHERE resourcestring = vResourceString) IS NULL THEN
                    IF (select 1=1 FROM acmgrantee WHERE grantee = vGrantee) IS NULL THEN
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
                        SELECT resourcestring INTO vResourceString FROM acmgrantee WHERE grantee = vGrantee LIMIT 1;
                    END IF;
                END IF;                                
            END IF;
            IF vParts > 3 THEN
                -- Typical groups
                SET vResourceString := CONCAT(vPart1, '/', vPart2, '/', vPart3, '/', vPart4);
                SET vGrantee := aacflatten(CONCAT(vPart2,'_',vPart4));
                IF (SELECT 1=1 FROM acmgrantee WHERE resourcestring = vResourceString) IS NULL THEN
                    IF (select 1=1 FROM acmgrantee WHERE grantee = vGrantee) IS NULL THEN
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
                        SELECT resourcestring INTO vResourceString FROM acmgrantee WHERE grantee = vGrantee LIMIT 1;
                    END IF;
                END IF;   
            END IF;
        END IF;
        -- See if grantee exists in acmvalue2
        IF (SELECT 1=1 FROM acmvalue2 WHERE name = vGrantee) IS NULL THEN
            INSERT INTO acmvalue2 (name) VALUES (vGrantee);
        END IF;        
    ELSE
        SET vResourceString := '';
    END IF;
	RETURN vResourceString;
END;
-- +migrate StatementEnd

-- a function to lookup the grantee from acmgrantee corresponding to the provided resourcestring
-- and from that lookup the acmvalue2.id that matches
INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql creating function calcGranteeIDFromResourceString';
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

-- Migrate existing objects, determining an identifier based upon ownedby, populating a respective
-- acmgrantee and acmvalue2 records as appropriate to support joining tables on user ownership
INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql creating procedure to populate ownedbyid';
DROP PROCEDURE IF EXISTS sp_Patch_20170331_transform_ownedbyid;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170331_transform_ownedbyid()
BEGIN
    INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql assign ownedbyid from ownedby';
    ASSIGNOWNEDBYID: BEGIN
        DECLARE ASSIGNOWNEDBYIDCOUNT int default 0;
        DECLARE ASSIGNOWNEDBYIDTOTAL int default 0;
        DECLARE vObjectID binary(16) default 0;
        DECLARE vOwnedByID int unsigned default 0;
        DECLARE vOwnedBy varchar(255) default '';
        DECLARE c_object_finished int default 0;
        DECLARE c_object cursor for SELECT id FROM object WHERE ownedbyid is null or ownedbyid = 0;
        DECLARE continue handler for not found set c_object_finished = 1;
        OPEN c_object;
        SELECT COUNT(0) INTO ASSIGNOWNEDBYIDTOTAL 
            FROM object where ownedbyid is null or ownedbyid = 0;
        get_object: LOOP
            FETCH c_object INTO vObjectID;
            IF c_object_finished = 1 THEN
                CLOSE c_object;
                LEAVE get_object;
            END IF;
            SET ASSIGNOWNEDBYIDCOUNT := ASSIGNOWNEDBYIDCOUNT + 1;
            IF floor(ASSIGNOWNEDBYIDCOUNT/5000) = ceiling(ASSIGNOWNEDBYIDCOUNT/5000) THEN
                INSERT INTO migration_status SET description = concat('20170331_409_ao_acm_performance.sql assign ownedbyid from ownedby (', ASSIGNOWNEDBYIDCOUNT, ' of ', ASSIGNOWNEDBYIDTOTAL, ')');
            END IF;            
            REVISIONS: BEGIN
                DECLARE vAID int default 0;
                DECLARE c_revision_finished int default 0;
                DECLARE c_revision cursor FOR 
                    SELECT a_id, ownedby FROM a_object WHERE id = vObjectID ORDER BY changecount asc;
                DECLARE continue handler for not found set c_revision_finished = 1;
                OPEN c_revision;
                get_revision: LOOP
                    FETCH c_revision INTO vAID, vOwnedBy;
                    IF c_revision_finished = 1 THEN
                        CLOSE c_revision;
                        LEAVE get_revision;
                    END IF;
                    SET vOwnedBy := calcResourceString(vOwnedBy);
                    IF LENGTH(vOwnedBy) > 0 THEN
                        SET vOwnedByID := calcGranteeIDFromResourceString(vOwnedBy);
                        UPDATE a_object SET ownedbyid = vOwnedByID, ownedby = vOwnedBy WHERE a_id = vAID;
                    END IF;
                END LOOP get_revision;
            END REVISIONS;
            IF LENGTH(vOwnedBy) > 0 THEN
                -- update the current object
                UPDATE object SET ownedbyid = vOwnedByID, ownedBy = vOwnedBy WHERE id = vObjectID;
            END IF;
        END LOOP get_object;
    END ASSIGNOWNEDBYID;   
END;
-- +migrate StatementEnd
CALL sp_Patch_20170331_transform_ownedbyid();
DROP PROCEDURE IF EXISTS sp_Patch_20170331_transform_ownedbyid;    

-- create triggers on object for insert/update taking into account the new acmid field
INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql recreating triggers';
DROP TRIGGER IF EXISTS ti_object;
DROP TRIGGER IF EXISTS tu_object;
DROP TRIGGER IF EXISTS ti_object_permission;
DROP TRIGGER IF EXISTS tu_object_permission;

-- +migrate StatementBegin
CREATE TRIGGER ti_object
BEFORE INSERT ON object FOR EACH ROW
BEGIN
    DECLARE error_msg varchar(128) default '';
    DECLARE thisTableName varchar(128) default 'object';
    DECLARE count_parent int default 0;
    DECLARE count_type int default 0;
    DECLARE type_contentConnector varchar(2000) default '';

    # Rules
    # Type must be specified
    SELECT COUNT(*) FROM object_type WHERE isDeleted = 0 AND id = NEW.typeId INTO count_type;
    IF count_type = 0 THEN
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
        SELECT COUNT(*) FROM object WHERE isDeleted = 0 AND id = NEW.parentId INTO count_parent;
        IF count_parent = 0 THEN
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
    SET NEW.ownedByNew := NULL;
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
    # Assign PDF availability if not set
    IF NEW.isPDFAvailable IS NULL THEN
        SET NEW.isPDFAvailable := 0;
    END IF;
    # Assign US Persons Data if not set
    IF NEW.containsUSPersonsData IS NULL THEN
        SET NEW.containsUSPersonsData := 'Unknown';
    END IF;
    # Assign FOIA Exempt status if not set
    IF NEW.exemptFromFOIA IS NULL THEN
        SET NEW.exemptFromFOIA := 'Unknown';
    END IF;
    # Assign stream storage state if not set
    IF NEW.isStreamStored IS NULL THEN
        SET NEW.isStreamStored := 0;
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
        ,ownedByNew
        ,isPDFAvailable
        ,isStreamStored
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
        ,NEW.ownedByNew
        ,NEW.isPDFAvailable
        ,NEW.isStreamStored
        ,NEW.containsUSPersonsData
        ,NEW.exemptFromFOIA
        ,NEW.acmId
        ,NEW.ownedById
    );

    # Specific field level changes
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedBy', newValue = NEW.ownedBy;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'typeId', newValue = hex(NEW.typeId);
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'name', newValue = NEW.name;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'description', newValue = NEW.description;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'parentId', newValue = hex(NEW.parentId);
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentConnector', newValue = NEW.contentConnector;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'rawAcm', newTextValue = NEW.rawAcm;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentType', newValue = NEW.contentType;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentSize', newValue = NEW.contentSize;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentHash', newValue = hex(NEW.contentHash);
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'encryptIV', newValue = hex(NEW.encryptIV);
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedByNew', newValue = NEW.ownedByNew;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isPDFAvailable', newValue = NEW.isPDFAvailable;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isStreamStored', newValue = NEW.isStreamStored;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'containsUSPersonsData', newValue = NEW.containsUSPersonsData;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'exemptFromFOIA', newValue = NEW.exemptFromFOIA;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmId', newValue = NEW.acmId;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedById', newValue = NEW.ownedById;

    # Pathing
    PATHING: BEGIN
        DECLARE vUniqueName varchar(300) default '';
        DECLARE vUniquePathName varchar(10240) default '';
        DECLARE vPathName varchar(10240) default '';
        DECLARE vParentPathName varchar(10240) default '';
        DECLARE vParentUniquePathName varchar(10240) default '';
        DECLARE vRootOwnedByGrantee varchar(255) default '';
        DECLARE vPathDelimiter varchar(5) default char(30);
        DECLARE vResourceDelimiter varchar(5) default '/';

        -- Get the unique name and initialize path as rooted
        SET vUniqueName := calcUniqueName(NEW.id, NEW.name, NEW.parentId, NEW.ownedBy);
        SET vUniquePathName := CONCAT(vPathDelimiter, vUniqueName);
        SET vPathName := CONCAT(vPathDelimiter, NEW.name);

        -- Get pathing info from parents
        IF NEW.parentId IS NULL THEN
            SELECT grantee INTO vRootOwnedByGrantee FROM acmgrantee WHERE resourceString = NEW.ownedBy or LOCATE(concat(resourceString,vResourceDelimiter),NEW.ownedBy) > 0 LIMIT 1;
            SET vUniquePathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vUniquePathName);
            SET vPathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vPathName);
        ELSE
            SELECT uniquePathName, pathName INTO vParentUniquePathName, vParentPathName FROM object_pathing WHERE id = NEW.parentId LIMIT 1;
            SET vPathName := CONCAT(vParentPathName, vPathName);
            SET vUniquePathName := CONCAT(vParentUniquePathName, vUniquePathName);
        END IF;        
        INSERT INTO object_pathing (id, uniqueName, uniquePathName, pathName) VALUES (NEW.id, vUniqueName, vUniquePathName, vPathName);
    END PATHING;

END;
-- +migrate StatementEnd
-- +migrate StatementBegin
CREATE TRIGGER tu_object
BEFORE UPDATE ON object FOR EACH ROW
BEGIN
    DECLARE error_msg varchar(128) default '';
    DECLARE thisTableName varchar(128) default 'object';
    DECLARE count_parent int default 0;
    DECLARE count_type int default 0;
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
    SELECT COUNT(*) FROM object_type WHERE isDeleted = 0 AND id = NEW.typeId INTO count_type;
    IF (count_type = 0) and length(error_msg) < 78 THEN
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
        SELECT COUNT(*) FROM object WHERE (isDeleted = 0 or (NEW.IsDeleted <> OLD.IsDeleted)) AND id = NEW.parentId INTO count_parent;
        IF count_parent = 0 THEN
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
        SET NEW.ownedByNew := NULL;
    END IF;
    SET NEW.ownedBy := calcResourceString(NEW.ownedBy);
    # ownedByID derived from ownedBy
    SET NEW.ownedByID := calcGranteeIDFromResourceString(NEW.ownedBy);
    # Assign PDF availability if not set
    IF NEW.isPDFAvailable IS NULL THEN
        SET NEW.isPDFAvailable = OLD.isPDFAvailable;
    END IF;
    # Assign US Persons Data if not set
    IF NEW.containsUSPersonsData IS NULL THEN
        SET NEW.containsUSPersonsData = 'Unknown';
    END IF;
    # Assign FOIA Exempt status if not set
    IF NEW.exemptFromFOIA IS NULL THEN
        SET NEW.exemptFromFOIA = 'Unknown';
    END IF;
    # Assign stream storage state if not set
    IF NEW.isStreamStored IS NULL THEN
        SET NEW.isStreamStored = OLD.isStreamStored;
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
        ,ownedByNew
        ,isPDFAvailable
        ,isStreamStored
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
        ,NEW.ownedByNew
        ,NEW.isPDFAvailable
        ,NEW.isStreamStored
        ,NEW.containsUSPersonsData
        ,NEW.exemptFromFOIA
        ,NEW.acmId
        ,NEW.ownedById
    );

    # Specific field level changes
    IF NEW.ownedBy <> OLD.ownedBy THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedBy', newValue = NEW.ownedBy;
    END IF;
    IF NEW.typeId <> OLD.typeId THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'typeId', newValue = hex(NEW.typeId);
    END IF;
    IF NEW.name <> OLD.name THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'name', newValue = NEW.name;
    END IF;
    IF NEW.description <> OLD.description THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'description', newValue = NEW.description;
    END IF;
    IF NEW.parentId <> OLD.parentId THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'parentId', newValue = hex(NEW.parentId);
    END IF;
    IF NEW.contentConnector <> OLD.contentConnector THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentConnector', newValue = NEW.contentConnector;
    END IF;
    IF NEW.rawAcm <> OLD.rawAcm THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'rawAcm', newTextValue = NEW.rawAcm;
    END IF;
    IF NEW.contentType <> OLD.contentType THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentType', newValue = NEW.contentType;
    END IF;
    IF NEW.contentSize <> OLD.contentSize THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentSize', newValue = NEW.contentSize;
    END IF;
    IF NEW.contentHash <> OLD.contentHash THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentHash', newValue = hex(NEW.contentHash);
    END IF;
    IF NEW.encryptIV <> OLD.encryptIV THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'encryptIV', newValue = hex(NEW.encryptIV);
    END IF;
    IF NEW.ownedByNew <> OLD.ownedByNew THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedByNew', newValue = NEW.ownedByNew;
    END IF;
    IF NEW.isPDFAvailable <> OLD.isPDFAvailable THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isPDFAvailable', newValue = NEW.isPDFAvailable;
    END IF;
    IF NEW.isStreamStored <> OLD.isStreamStored THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isStreamStored', newValue = NEW.isStreamStored;
    END IF;
    IF NEW.containsUSPersonsData <> OLD.containsUSPersonsData THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'containsUSPersonsData', newValue = NEW.containsUSPersonsData;
    END IF;
    IF NEW.exemptFromFOIA <> OLD.exemptFromFOIA THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'exemptFromFOIA', newValue = NEW.exemptFromFOIA;
    END IF;
    IF NEW.acmId <> OLD.acmId THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'acmId', newValue = NEW.acmId;
    END IF;
    IF NEW.ownedById <> OLD.ownedById THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedById', newValue = NEW.ownedById;
    END IF;

    # Pathing
    IF (NEW.name <> OLD.name) or (NEW.parentId <> OLD.parentId) or (NEW.ownedBy <> OLD.ownedBy) THEN
        PATHING: BEGIN
            DECLARE vUniqueName varchar(300) default '';
            DECLARE vUniquePathName varchar(10240) default '';
            DECLARE vPathName varchar(10240) default '';
            DECLARE vParentPathName varchar(10240) default '';
            DECLARE vParentUniquePathName varchar(10240) default '';
            DECLARE vRootOwnedByGrantee varchar(255) default '';
            DECLARE vOldPathName varchar(10240) default '';
            DECLARE vOldUniquePathName varchar(10240) default '';
            DECLARE vPathDelimiter varchar(5) default char(30);
            DECLARE vResourceDelimiter varchar(5) default '/';

            -- Get the unique name and initialize path as rooted
            SET vUniqueName := calcUniqueName(NEW.id, NEW.name, NEW.parentID, NEW.ownedBy);
            SET vUniquePathName := CONCAT(vPathDelimiter, vUniqueName);
            SET vPathName := CONCAT(vPathDelimiter, NEW.name);

            -- Get new pathing info from parents
            IF NEW.parentId IS NULL THEN
                SELECT grantee INTO vRootOwnedByGrantee FROM acmgrantee WHERE resourceString = NEW.ownedBy or LOCATE(concat(resourceString,vResourceDelimiter),NEW.ownedBy) > 0 LIMIT 1;
                SET vUniquePathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vUniquePathName);
                SET vPathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vPathName);
            ELSE
                SELECT uniquePathName, pathName INTO vParentUniquePathName, vParentPathName FROM object_pathing WHERE id = NEW.parentId LIMIT 1;
                SET vPathName := CONCAT(vParentPathName, vPathName);
                SET vUniquePathName := CONCAT(vParentUniquePathName, vUniquePathName);
            END IF;        

            -- Get old pathing info
            SELECT uniquePathName, pathName INTO vOldUniquePathName, vOldPathName FROM object_pathing WHERE id = NEW.id LIMIT 1;

            -- Update this record
            UPDATE object_pathing SET 
                uniqueName = vUniqueName, 
                uniquePathName = vUniquePathName, 
                pathName = vPathName 
            WHERE id = NEW.id;

            -- Update descendents
            UPDATE object_pathing SET 
                uniquePathName = REPLACE(uniquePathName, vOldUniquePathName, vUniquePathName),
                pathName = REPLACE(pathName, vOldPathName, vPathName)
            WHERE
                uniquePathName LIKE CONCAT(vOldUniquePathName,vPathDelimiter,'%')
                AND
                pathName LIKE CONCAT(vOldPathName,vPathDelimiter,'%');
        END PATHING;    
    END IF;

END;
-- +migrate StatementEnd
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
INSERT INTO migration_status SET description = '20170331_409_ao_acm_performance.sql setting schema version';
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
	SET NEW.schemaversion := '20170331'; 
	# Identifier is randomized as a GUID
	SET NEW.identifier := concat(@@hostname, '-', left(uuid(),8));
END;
-- +migrate StatementEnd
update dbstate set schemaVersion = '20170331';

-- +migrate Down

-- remove triggers on object for create/update
DROP TRIGGER IF EXISTS ti_object;
DROP TRIGGER IF EXISTS tu_object;

-- remove constraints
ALTER TABLE `object` DROP FOREIGN KEY `fk_object_acmid`;
ALTER TABLE `object` DROP INDEX `fk_object_acmid`;
ALTER TABLE `object` DROP FOREIGN KEY `fk_object_ownedbyid`;
ALTER TABLE `object` DROP INDEX `fk_object_ownedbyid`;
ALTER TABLE `useracm` DROP FOREIGN KEY `fk_useracm_userid`;
ALTER TABLE `useracm` DROP INDEX `fk_useracm_userid`;
ALTER TABLE `useracm` DROP FOREIGN KEY `fk_useracm_acmid`;
ALTER TABLE `useracm` DROP INDEX `fk_useracm_acmid`;
ALTER TABLE `useraocache` DROP FOREIGN KEY `fk_useraocache_userid`;
ALTER TABLE `useraocache` DROP INDEX `fk_useraocache_userid`;
ALTER TABLE `useraocachepart` DROP FOREIGN KEY `fk_useraocachepart_userid`;
ALTER TABLE `useraocachepart` DROP INDEX `fk_useraocachepart_userid`;
ALTER TABLE `useraocachepart` DROP FOREIGN KEY `fk_useraocachepart_userkeyid`;
ALTER TABLE `useraocachepart` DROP INDEX `fk_useraocachepart_userkeyid`;
ALTER TABLE `useraocachepart` DROP FOREIGN KEY `fk_useraocachepart_uservalueid`;
ALTER TABLE `useraocachepart` DROP INDEX `fk_useraocachepart_uservalueid`;
ALTER TABLE `acmpart2` DROP FOREIGN KEY `fk_acmpart2_acmid`;
ALTER TABLE `acmpart2` DROP INDEX `fk_acmpart2_acmid`;
ALTER TABLE `acmpart2` DROP FOREIGN KEY `fk_acmpart2_acmkeyid`;
ALTER TABLE `acmpart2` DROP INDEX `fk_acmpart2_acmkeyid`;
ALTER TABLE `acmpart2` DROP FOREIGN KEY `fk_acmpart2_acmvalueid`;
ALTER TABLE `acmpart2` DROP INDEX `fk_acmpart2_acmvalueid`;

-- remove column acmid from object, a_object tables
ALTER TABLE object DROP COLUMN acmid;
ALTER TABLE a_object DROP COLUMN acmid;

-- remove column ownedbyid from object, a_object tables
ALTER TABLE object DROP COLUMN ownedbyid;
ALTER TABLE a_object DROP COLUMN ownedbyid;

-- remove tables if exist
DROP TABLE IF EXISTS migration_status;
DROP TABLE IF EXISTS useracm;
DROP TABLE IF EXISTS useraocache;
DROP TABLE IF EXISTS useraocachepart;
DROP TABLE IF EXISTS acmpart2;
DROP TABLE IF EXISTS acmvalue2;
DROP TABLE IF EXISTS acmkey2;
DROP TABLE IF ExISTS acm2;

-- recreate triggers for insert/update as in 20170301
-- +migrate StatementBegin
CREATE TRIGGER ti_object
BEFORE INSERT ON object FOR EACH ROW
BEGIN
    DECLARE error_msg varchar(128) default '';
    DECLARE thisTableName varchar(128) default 'object';
    DECLARE count_parent int default 0;
    DECLARE count_type int default 0;
    DECLARE type_contentConnector varchar(2000) default '';

    # Rules
    # Type must be specified
    SELECT COUNT(*) FROM object_type WHERE isDeleted = 0 AND id = NEW.typeId INTO count_type;
    IF count_type = 0 THEN
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
        SELECT COUNT(*) FROM object WHERE isDeleted = 0 AND id = NEW.parentId INTO count_parent;
        IF count_parent = 0 THEN
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
    SET NEW.ownedByNew := NULL;
    # Assign Owner if not set
    IF NEW.ownedBy IS NULL OR NEW.ownedBy = '' THEN
        SET NEW.ownedBy := concat('user/', NEW.createdBy);
    END IF;
    SET NEW.changeCount := 0;
    # Standard change token formula
    SET NEW.changeToken := md5(CONCAT(CAST(NEW.id AS CHAR),':',CAST(NEW.changeCount AS CHAR),':',CAST(NEW.modifiedDate AS CHAR)));
    # Assign contentConnector if not set
    IF NEW.contentConnector IS NULL OR NEW.contentConnector = '' THEN
        SELECT contentConnector FROM object_type WHERE isDeleted = 0 AND id = NEW.typeId INTO type_contentConnector;
        SET NEW.contentConnector := type_contentConnector;
    END IF;
    # Assign PDF availability if not set
    IF NEW.isPDFAvailable IS NULL THEN
        SET NEW.isPDFAvailable := 0;
    END IF;
    # Assign US Persons Data if not set
    IF NEW.containsUSPersonsData IS NULL THEN
        SET NEW.containsUSPersonsData := 'Unknown';
    END IF;
    # Assign FOIA Exempt status if not set
    IF NEW.exemptFromFOIA IS NULL THEN
        SET NEW.exemptFromFOIA := 'Unknown';
    END IF;
    # Assign stream storage state if not set
    IF NEW.isStreamStored IS NULL THEN
        SET NEW.isStreamStored := 0;
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
        ,ownedByNew
        ,isPDFAvailable
        ,isStreamStored
        ,containsUSPersonsData
        ,exemptFromFOIA
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
        ,NEW.ownedByNew
        ,NEW.isPDFAvailable
        ,NEW.isStreamStored
        ,NEW.containsUSPersonsData
        ,NEW.exemptFromFOIA
    );

    # Specific field level changes
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedBy', newValue = NEW.ownedBy;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'typeId', newValue = hex(NEW.typeId);
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'name', newValue = NEW.name;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'description', newValue = NEW.description;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'parentId', newValue = hex(NEW.parentId);
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentConnector', newValue = NEW.contentConnector;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'rawAcm', newTextValue = NEW.rawAcm;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentType', newValue = NEW.contentType;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentSize', newValue = NEW.contentSize;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentHash', newValue = hex(NEW.contentHash);
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'encryptIV', newValue = hex(NEW.encryptIV);
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedByNew', newValue = NEW.ownedByNew;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isPDFAvailable', newValue = NEW.isPDFAvailable;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isStreamStored', newValue = NEW.isStreamStored;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'containsUSPersonsData', newValue = NEW.containsUSPersonsData;
    INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'exemptFromFOIA', newValue = NEW.exemptFromFOIA;

    # Pathing
    PATHING: BEGIN
        DECLARE vUniqueName varchar(300) default '';
        DECLARE vUniquePathName varchar(10240) default '';
        DECLARE vPathName varchar(10240) default '';
        DECLARE vParentPathName varchar(10240) default '';
        DECLARE vParentUniquePathName varchar(10240) default '';
        DECLARE vRootOwnedByGrantee varchar(255) default '';
        DECLARE vPathDelimiter varchar(5) default char(30);
        DECLARE vResourceDelimiter varchar(5) default '/';

        -- Get the unique name and initialize path as rooted
        SET vUniqueName := calcUniqueName(NEW.id, NEW.name, NEW.parentId, NEW.ownedBy);
        SET vUniquePathName := CONCAT(vPathDelimiter, vUniqueName);
        SET vPathName := CONCAT(vPathDelimiter, NEW.name);

        -- Get pathing info from parents
        IF NEW.parentId IS NULL THEN
            SELECT grantee INTO vRootOwnedByGrantee FROM acmgrantee WHERE resourceString = NEW.ownedBy or LOCATE(concat(resourceString,vResourceDelimiter),NEW.ownedBy) > 0;
            SET vUniquePathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vUniquePathName);
            SET vPathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vPathName);
        ELSE
            SELECT uniquePathName, pathName INTO vParentUniquePathName, vParentPathName FROM object_pathing WHERE id = NEW.parentId;
            SET vPathName := CONCAT(vParentPathName, vPathName);
            SET vUniquePathName := CONCAT(vParentUniquePathName, vUniquePathName);
        END IF;        
        INSERT INTO object_pathing (id, uniqueName, uniquePathName, pathName) VALUES (NEW.id, vUniqueName, vUniquePathName, vPathName);
    END PATHING;

END;
-- +migrate StatementEnd
-- +migrate StatementBegin
CREATE TRIGGER tu_object
BEFORE UPDATE ON object FOR EACH ROW
BEGIN
    DECLARE error_msg varchar(128) default '';
    DECLARE thisTableName varchar(128) default 'object';
    DECLARE count_parent int default 0;
    DECLARE count_type int default 0;
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
    SELECT COUNT(*) FROM object_type WHERE isDeleted = 0 AND id = NEW.typeId INTO count_type;
    IF (count_type = 0) and length(error_msg) < 78 THEN
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
        SELECT COUNT(*) FROM object WHERE (isDeleted = 0 or (NEW.IsDeleted <> OLD.IsDeleted)) AND id = NEW.parentId INTO count_parent;
        IF count_parent = 0 THEN
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
        SET NEW.ownedByNew := NULL;
    END IF;
    # Assign PDF availability if not set
    IF NEW.isPDFAvailable IS NULL THEN
        SET NEW.isPDFAvailable = OLD.isPDFAvailable;
    END IF;
    # Assign US Persons Data if not set
    IF NEW.containsUSPersonsData IS NULL THEN
        SET NEW.containsUSPersonsData = 'Unknown';
    END IF;
    # Assign FOIA Exempt status if not set
    IF NEW.exemptFromFOIA IS NULL THEN
        SET NEW.exemptFromFOIA = 'Unknown';
    END IF;
    # Assign stream storage state if not set
    IF NEW.isStreamStored IS NULL THEN
        SET NEW.isStreamStored = OLD.isStreamStored;
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
        ,ownedByNew
        ,isPDFAvailable
        ,isStreamStored
        ,containsUSPersonsData
        ,exemptFromFOIA
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
        ,NEW.ownedByNew
        ,NEW.isPDFAvailable
        ,NEW.isStreamStored
        ,NEW.containsUSPersonsData
        ,NEW.exemptFromFOIA
    );

    # Specific field level changes
    IF NEW.ownedBy <> OLD.ownedBy THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedBy', newValue = NEW.ownedBy;
    END IF;
    IF NEW.typeId <> OLD.typeId THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'typeId', newValue = hex(NEW.typeId);
    END IF;
    IF NEW.name <> OLD.name THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'name', newValue = NEW.name;
    END IF;
    IF NEW.description <> OLD.description THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'description', newValue = NEW.description;
    END IF;
    IF NEW.parentId <> OLD.parentId THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'parentId', newValue = hex(NEW.parentId);
    END IF;
    IF NEW.contentConnector <> OLD.contentConnector THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentConnector', newValue = NEW.contentConnector;
    END IF;
    IF NEW.rawAcm <> OLD.rawAcm THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'rawAcm', newTextValue = NEW.rawAcm;
    END IF;
    IF NEW.contentType <> OLD.contentType THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentType', newValue = NEW.contentType;
    END IF;
    IF NEW.contentSize <> OLD.contentSize THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentSize', newValue = NEW.contentSize;
    END IF;
    IF NEW.contentHash <> OLD.contentHash THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'contentHash', newValue = hex(NEW.contentHash);
    END IF;
    IF NEW.encryptIV <> OLD.encryptIV THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'encryptIV', newValue = hex(NEW.encryptIV);
    END IF;
    IF NEW.ownedByNew <> OLD.ownedByNew THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'ownedByNew', newValue = NEW.ownedByNew;
    END IF;
    IF NEW.isPDFAvailable <> OLD.isPDFAvailable THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isPDFAvailable', newValue = NEW.isPDFAvailable;
    END IF;
    IF NEW.isStreamStored <> OLD.isStreamStored THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'isStreamStored', newValue = NEW.isStreamStored;
    END IF;
    IF NEW.containsUSPersonsData <> OLD.containsUSPersonsData THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'containsUSPersonsData', newValue = NEW.containsUSPersonsData;
    END IF;
    IF NEW.exemptFromFOIA <> OLD.exemptFromFOIA THEN
        INSERT field_changes SET modifiedDate = NEW.modifiedDate, modifiedBy = NEW.modifiedBy, recordId = NEW.id, tableName = thisTableName, columnName = 'exemptFromFOIA', newValue = NEW.exemptFromFOIA;
    END IF;

    # Pathing
    IF (NEW.name <> OLD.name) or (NEW.parentId <> OLD.parentId) or (NEW.ownedBy <> OLD.ownedBy) THEN
        PATHING: BEGIN
            DECLARE vUniqueName varchar(300) default '';
            DECLARE vUniquePathName varchar(10240) default '';
            DECLARE vPathName varchar(10240) default '';
            DECLARE vParentPathName varchar(10240) default '';
            DECLARE vParentUniquePathName varchar(10240) default '';
            DECLARE vRootOwnedByGrantee varchar(255) default '';
            DECLARE vOldPathName varchar(10240) default '';
            DECLARE vOldUniquePathName varchar(10240) default '';
            DECLARE vPathDelimiter varchar(5) default char(30);
            DECLARE vResourceDelimiter varchar(5) default '/';

            -- Get the unique name and initialize path as rooted
            SET vUniqueName := calcUniqueName(NEW.id, NEW.name, NEW.parentID, NEW.ownedBy);
            SET vUniquePathName := CONCAT(vPathDelimiter, vUniqueName);
            SET vPathName := CONCAT(vPathDelimiter, NEW.name);

            -- Get new pathing info from parents
            IF NEW.parentId IS NULL THEN
                SELECT grantee INTO vRootOwnedByGrantee FROM acmgrantee WHERE resourceString = NEW.ownedBy or LOCATE(concat(resourceString,vResourceDelimiter),NEW.ownedBy) > 0;
                SET vUniquePathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vUniquePathName);
                SET vPathName := CONCAT(vPathDelimiter, vRootOwnedByGrantee, vPathName);
            ELSE
                SELECT uniquePathName, pathName INTO vParentUniquePathName, vParentPathName FROM object_pathing WHERE id = NEW.parentId;
                SET vPathName := CONCAT(vParentPathName, vPathName);
                SET vUniquePathName := CONCAT(vParentUniquePathName, vUniquePathName);
            END IF;        

            -- Get old pathing info
            SELECT uniquePathName, pathName INTO vOldUniquePathName, vOldPathName FROM object_pathing WHERE id = NEW.id;

            -- Update this record
            UPDATE object_pathing SET 
                uniqueName = vUniqueName, 
                uniquePathName = vUniquePathName, 
                pathName = vPathName 
            WHERE id = NEW.id;

            -- Update descendents
            UPDATE object_pathing SET 
                uniquePathName = REPLACE(uniquePathName, vOldUniquePathName, vUniquePathName),
                pathName = REPLACE(pathName, vOldPathName, vPathName)
            WHERE
                uniquePathName LIKE CONCAT(vOldUniquePathName,vPathDelimiter,'%')
                AND
                pathName LIKE CONCAT(vOldPathName,vPathDelimiter,'%');
        END PATHING;    
    END IF;

END;
-- +migrate StatementEnd

DROP TRIGGER IF EXISTS ti_dbstate;

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
	SET NEW.schemaversion := '20170301'; 
	# Identifier is randomized as a GUID
	SET NEW.identifier := concat(@@hostname, '-', left(uuid(),8));
END;
-- +migrate StatementEnd
update dbstate set schemaVersion = '20170301';