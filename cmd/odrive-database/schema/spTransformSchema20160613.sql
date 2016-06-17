delimiter //

# This stored procedure can be used to examine existing objects and their 
# object_acm data and generate appropriate objectacm, acm, and acmpart
# records

DROP PROCEDURE IF EXISTS sp_TransformSchema20160613
//
SELECT 'Creating procedure sp_TransformSchema20160613' as Action
//
CREATE PROCEDURE sp_TransformSchema20160613(
)
BEGIN
    # Process objects that have object_acm but no corresponding objectacm
    DECLARE cursor1_id binary(16);
    DECLARE cursor1_rawacm text;
    DECLARE cursor1_modifiedby varchar(255);
    DECLARE cursor1done boolean default false;
    DECLARE cursor1 cursor for 
        select id, rawacm, modifiedby from object where id not in (select objectid from objectacm);
    DECLARE continue handler for not found set cursor1done := true;
    OPEN cursor1;
    LOOP1: LOOP
        FETCH cursor1 INTO cursor1_id, cursor1_rawacm, cursor1_modifiedby;
        IF cursor1done THEN
            LEAVE LOOP1;
        END IF;
        # Determine ACM name, a simplified flattened format serialized from acmkey and acmvalue
        BLOCK2: begin
            DECLARE acmcount int;
            DECLARE newacmid binary(16);
            DECLARE keynameold varchar(255);
            DECLARE acmname varchar(8000);
            DECLARE cursor2_acmkeyname varchar(255);
            DECLARE cursor2_acmvaluename varchar(255);
            DECLARE cursor2done boolean default false;
            DECLARE cursor2 cursor for 
                select ak.name, av.name from object_acm 
                inner join acmkey ak on object_acm.acmkeyid = ak.id 
                inner join acmvalue av on object_acm.acmvalueid = av.id 
                where object_acm.objectid = cursor1_id and object_acm.isdeleted = 0 order by ak.name, av.name;
            DECLARE continue handler for not found set cursor2done := true;
            SET acmname := '';
            SET keynameold := '';
            OPEN cursor2;
            LOOP2: LOOP
                FETCH cursor2 INTO cursor2_acmkeyname, cursor2_acmvaluename;
                IF cursor2done THEN
                    LEAVE LOOP2;
                END IF;
                IF keynameold <> cursor2_acmkeyname THEN
                    IF LENGTH(acmname) > 0 THEN
                        SET acmname := CONCAT(acmname, ';');
                    END IF;
                    SET acmname := CONCAT(acmname, cursor2_acmkeyname, '=', cursor2_acmvaluename);
                    SET keynameold := cursor2_acmkeyname;
                ELSE
                    SET acmname := CONCAT(acmname, ',', cursor2_acmvaluename);
                END IF;                
            END LOOP LOOP2;
            CLOSE cursor2;
            IF LENGTH(acmname) > 0 THEN
                SELECT count(0) FROM acm WHERE name = acmname INTO acmcount;
                IF acmcount = 0 THEN
                    # make it
                    INSERT INTO acm SET createdBy = cursor1_modifiedby, name = acmname;
                    SELECT id FROM acm WHERE name = acmname INTO newacmid;
                    BLOCK3: begin
                        DECLARE cursor3_acmkeyid binary(16);
                        DECLARE cursor3_acmkeyname varchar(255);
                        DECLARE cursor3_acmvalueid binary(16);
                        DECLARE cursor3_acmvaluename varchar(255);
                        DECLARE cursor3done boolean default false;
                        DECLARE cursor3 cursor for 
                            select ak.id, ak.name, av.id, av.name from object_acm
                            inner join acmkey ak on object_acm.acmkeyid = ak.id 
                            inner join acmvalue av on object_acm.acmvalueid = av.id 
                            where object_acm.objectid = cursor1_id and object_acm.isdeleted = 0 order by ak.name, av.name;
                        DECLARE continue handler for not found set cursor3done := true;
                        OPEN cursor3;
                        LOOP3: LOOP
                            FETCH cursor3 INTO cursor3_acmkeyid, cursor3_acmkeyname, cursor3_acmvalueid, cursor3_acmvaluename;
                            IF cursor3done THEN
                                LEAVE LOOP3;
                            END IF;
                            INSERT INTO acmpart SET createdBy = cursor1_modifiedby, acmid = newacmid, acmkeyid = cursor3_acmkeyid, acmvalueid = cursor3_acmvalueid;
                        END LOOP LOOP3;
                        CLOSE cursor3;
                    END BLOCK3;
                ELSE
                    SELECT id FROM acm WHERE name = acmname INTO newacmid;
                END IF;
                # associate acm to object
                INSERT INTO objectacm SET createdby = cursor1_modifiedby, objectid = cursor1_id, acmid = newacmid;
            ELSE
                SELECT CONCAT('Object ',hex(cursor1_id),' does not have an acm') as Action;
            END IF;
        end BLOCK2;    
    END LOOP LOOP1;
    CLOSE cursor1;
END  
//
delimiter ;
