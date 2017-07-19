-- +migrate Up

-- Fixes duplicate entries in acmkey2 and acmvalue2 and references in useraocachepart and acmpart2

SET FOREIGN_KEY_CHECKS=0;

INSERT INTO migration_status SET description = '20170718_fix_duplicates';
DROP PROCEDURE IF EXISTS sp_Patch_20170718_fix_duplicates;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170718_fix_duplicates()
proc_label: BEGIN
    -- only do this if not yet 20170718
    IF EXISTS( select null from dbstate where schemaversion = '20170718') THEN
        LEAVE proc_label;
    END IF;

    INSERT INTO migration_status SET description = '20170718_fix_duplicates acmkey2';
    ACMKEYMIGRATE: BEGIN
        DECLARE vNamePrev text default '';
        DECLARE vNameCur text default '';
        DECLARE vIDGood int default 0;
        DECLARE vIDBad int default 0;

        DECLARE c_acmkey_finished int default 0;
        DECLARE c_acmkey cursor for 
            select name, id 
            from acmkey2 where name in (
                select name from (select name, count(name) from acmkey2 group by name having count(name) > 1) t1
            ) order by name asc, id asc;
        DECLARE continue handler for not found set c_acmkey_finished = 1;
        OPEN c_acmkey;
        get_acmkey: LOOP
            FETCH c_acmkey INTO vNameCur, vIDBad;
            IF c_acmkey_finished = 1 THEN
                CLOSE c_acmkey;
                LEAVE get_acmkey;
            END IF;
            IF vNamePrev <> vNameCur THEN
                SET vIDGood := vIDBad;
            ELSE
                update useraocachepart set userkeyid = vIDGood where userkeyid = vIDBad;
                update acmpart2 set acmkeyid = vIDGood where acmkeyid = vIDBad;
                delete from acmkey2 where id = vIDBad;
            END IF;
            SET vNamePrev := vNameCur;
        END LOOP get_acmkey;
    END ACMKEYMIGRATE;

    INSERT INTO migration_status SET description = '20170718_fix_duplicates acmvalue2';
    ACMVALUEMIGRATE: BEGIN
        DECLARE vNamePrev text default '';
        DECLARE vNameCur text default '';
        DECLARE vIDGood int default 0;
        DECLARE vIDBad int default 0;

        DECLARE c_acmvalue_finished int default 0;
        DECLARE c_acmvalue cursor for 
            select name, id 
            from acmvalue2 where name in (
                select name from (select name, count(name) from acmvalue2 group by name having count(name) > 1) t1
            ) order by name asc, id asc;
        DECLARE continue handler for not found set c_acmvalue_finished = 1;
        OPEN c_acmvalue;
        get_acmvalue: LOOP
            FETCH c_acmvalue INTO vNameCur, vIDBad;
            IF c_acmvalue_finished = 1 THEN
                CLOSE c_acmvalue;
                LEAVE get_acmvalue;
            END IF;
            IF vNamePrev <> vNameCur THEN
                SET vIDGood := vIDBad;
            ELSE
                update useraocachepart set uservalueid = vIDGood where uservalueid = vIDBad;
                update acmpart2 set acmvalueid = vIDGood where acmvalueid = vIDBad;
                delete from acmvalue2 where id = vIDBad;
            END IF;
            SET vNamePrev := vNameCur;
        END LOOP get_acmvalue;
    END ACMVALUEMIGRATE;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170718_fix_duplicates();
DROP PROCEDURE IF EXISTS sp_Patch_20170718_fix_duplicates;

INSERT INTO migration_status SET description = '20170718_create_unique_constraints';
DROP PROCEDURE IF EXISTS sp_Patch_20170718_create_unique_constraints;
-- +migrate StatementBegin
CREATE PROCEDURE sp_Patch_20170718_create_unique_constraints()
BEGIN
    -- acmkey2
    IF NOT EXISTS (select null from information_schema.table_constraints where table_name = 'acmkey2' and binary constraint_name = 'uc_acmkey2_name') THEN
        insert into migration_status set description = '20170718 creating unique constraint for acmkey2';
        alter table acmkey2 add constraint uc_acmkey2_name unique(name);
    END IF;
    -- acmvalue2
    IF NOT EXISTS (select null from information_schema.table_constraints where table_name = 'acmvalue2' and binary constraint_name = 'uc_acmvalue2_name') THEN
        insert into migration_status set description = '20170718 creating unique constraint for acmvalue2';
        alter table acmvalue2 add constraint uc_acmvalue2_name unique(name);
    END IF;
END;
-- +migrate StatementEnd
CALL sp_Patch_20170718_create_unique_constraints();
SET FOREIGN_KEY_CHECKS=1;
DROP PROCEDURE IF EXISTS sp_Patch_20170718_create_unique_constraints; 

update dbstate set schemaVersion = '20170718' where schemaVersion <> '20170718';

-- +migrate Down

-- A downgrade from this schema version will not be supported.

update dbstate set schemaVersion = '20170718' where schemaVersion <> '20170718';