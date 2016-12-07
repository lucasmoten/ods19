delimiter //

# This stored procedure can be used to generate test data
# It is not part of the base schema and is primarily intended for development load testing


DROP PROCEDURE IF EXISTS sp_TestData
//
SELECT 'Creating procedure' as Action
//
CREATE PROCEDURE sp_TestData(
	IN max_object_type int,
	IN max_object int,
    IN assignedParentId varchar(32),
	IN prefix varchar(20),
	IN suffix varchar(20)
)
BEGIN
	DECLARE player1 varchar(255) default 'cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us';
	DECLARE player2 varchar(255) default 'cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us';
    DECLARE grantee1 varchar(255) default 'cntesttester01oupeopleoudaeouchimeraou_s_governmentcus';
    DECLARE grantee2 varchar(255) default 'cntesttester10oupeopleoudaeouchimeraou_s_governmentcus';
	DECLARE cur_object_type int default 0;
	DECLARE new_type_name varchar(60) default '';
	DECLARE new_type_id binary(16) default '';
	DECLARE new_type_id_hex varchar(32) default '';
	DECLARE cur_object int default 0;
	DECLARE new_object_name varchar(60) default '';
	DECLARE new_object_id binary(16) default '';
	DECLARE count_user int default 0;
	DECLARE count_type int default 0;
	DECLARE count_object int default 0;
	DECLARE createdBy varchar(255) default '';
	DECLARE random_parent_number int default 0;
	DECLARE parent_id binary(16) default '';
	DECLARE change_token varchar(60) default '';
	DECLARE object_type_ids_hex varchar(2000) default '';
	DECLARE parent_id_hex varchar(32) default '';
	DECLARE parent_ids_hex varchar(5000) default '';
	DECLARE parent_ids_hex_ml int default 0;
    DECLARE new_object_acm varchar(2000) default '';
    DECLARE acm1 varchar(2000) default '{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}';
    DECLARE acm2 varchar(2000) default '{"version":"2.1.0","classif":"TS","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":["si","tk"],"disponly_to":[""],"dissem_ctrls":[""],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"TS//SI/TK","banner":"TOP SECRET//SI/TK","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["ts"],"f_sci_ctrls":["si","tk"],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}';
    DECLARE acmkeyid_dissem_countries binary(16);
    DECLARE acmkeyid_f_accms binary(16);
    DECLARE acmkeyid_f_atom_energy binary(16);    
    DECLARE acmkeyid_f_clearance binary(16);
    DECLARE acmkeyid_f_macs binary(16);
    DECLARE acmkeyid_f_missions binary(16);
    DECLARE acmkeyid_f_oc_org binary(16);
    DECLARE acmkeyid_f_regions binary(16);
    DECLARE acmkeyid_f_sci_ctrls binary(16);
    DECLARE acmvalueid_si binary(16);
    DECLARE acmvalueid_tk binary(16);
    DECLARE acmvalueid_ts binary(16);
    DECLARE acmvalueid_u binary(16);
    DECLARE acmvalueid_USA binary(16);
    DECLARE acm1id binary(16);
    DECLARE acm2id binary(16);
    DECLARE acmpartid binary(16);
    DECLARE otherUser varchar(255) default '';
    DECLARE granteeForCreatedBy varchar(255) default '';
    DECLARE granteeForOtherUser varchar(255) default '';
    DECLARE acmShareForCreatedBy varchar(255) default '';
    DECLARE acmShareForOtherUser varchar(255) default '';
    DECLARE acmShareForEveryone varchar(255) default '{"projects":{"":{"disp_nm":"","groups":["-Everyone"]}}}';
    DECLARE granteeForEveryone varchar(255) default '_everyone';

	# Create player1 if not exist
	SELECT count(*) FROM user where distinguishedName = player1 INTO count_user;
	IF count_user = 0 THEN
		INSERT user SET distinguishedName = player1, createdBy = player1;
	END IF;
	# Create player2 if not exist
	SELECT count(*) FROM user where distinguishedName = player2 INTO count_user;
	IF count_user = 0 THEN
		INSERT user SET distinguishedName = player2, createdBy = player1;
	END IF;
    # Create ACM keys
    SELECT count(*) FROM acmkey where name = 'dissem_countries' and isdeleted = 0 INTO count_user;
    IF count_user = 0 THEN
        INSERT acmkey SET createdBy = player1, name = 'dissem_countries';
    END IF;
    SELECT id from acmkey where name = 'dissem_countries' and isdeleted = 0 INTO acmkeyid_dissem_countries;
    SELECT count(*) FROM acmkey where name = 'f_accms' and isdeleted = 0 INTO count_user;
    IF count_user = 0 THEN
        INSERT acmkey SET createdBy = player1, name = 'f_accms';
    END IF;
    SELECT id from acmkey where name = 'f_accms' and isdeleted = 0 INTO acmkeyid_f_accms;
    SELECT count(*) FROM acmkey where name = 'f_atom_energy' and isdeleted = 0 INTO count_user;
    IF count_user = 0 THEN
        INSERT acmkey SET createdBy = player1, name = 'f_atom_energy';
    END IF;
    SELECT id from acmkey where name = 'f_atom_energy' and isdeleted = 0 INTO acmkeyid_f_atom_energy;
    SELECT count(*) FROM acmkey where name = 'f_clearance' and isdeleted = 0 INTO count_user;
    IF count_user = 0 THEN
        INSERT acmkey SET createdBy = player1, name = 'f_clearance';
    END IF;
    SELECT id from acmkey where name = 'f_clearance' and isdeleted = 0 INTO acmkeyid_f_clearance;
    SELECT count(*) FROM acmkey where name = 'f_macs' and isdeleted = 0 INTO count_user;
    IF count_user = 0 THEN
        INSERT acmkey SET createdBy = player1, name = 'f_macs';
    END IF;
    SELECT id from acmkey where name = 'f_macs' and isdeleted = 0 INTO acmkeyid_f_macs;
    SELECT count(*) FROM acmkey where name = 'f_missions' and isdeleted = 0 INTO count_user;
    IF count_user = 0 THEN
        INSERT acmkey SET createdBy = player1, name = 'f_missions';
    END IF;
    SELECT id from acmkey where name = 'f_missions' and isdeleted = 0 INTO acmkeyid_f_missions;
    SELECT count(*) FROM acmkey where name = 'f_oc_org' and isdeleted = 0 INTO count_user;
    IF count_user = 0 THEN
        INSERT acmkey SET createdBy = player1, name = 'f_oc_org';
    END IF;
    SELECT id from acmkey where name = 'f_oc_org' and isdeleted = 0 INTO acmkeyid_f_oc_org;
    SELECT count(*) FROM acmkey where name = 'f_regions' and isdeleted = 0 INTO count_user;
    IF count_user = 0 THEN
        INSERT acmkey SET createdBy = player1, name = 'f_regions';
    END IF;
    SELECT id from acmkey where name = 'f_regions' and isdeleted = 0 INTO acmkeyid_f_regions;
    SELECT count(*) FROM acmkey where name = 'f_sci_ctrls' and isdeleted = 0 INTO count_user;
    IF count_user = 0 THEN
        INSERT acmkey SET createdBy = player1, name = 'f_sci_ctrls';
    END IF;
    SELECT id from acmkey where name = 'f_sci_ctrls' and isdeleted = 0 INTO acmkeyid_f_sci_ctrls;
    # Create ACM Values
    SELECT count(*) from acmvalue where name = 'si' and isdeleted = 0 INTO count_user;
    IF count_user = 0 THEN
        INSERT acmvalue SET createdBy = player1, name = 'si';
    END IF;
    SELECT id from acmvalue where name = 'si' and isdeleted = 0 INTO acmvalueid_si;
    SELECT count(*) from acmvalue where name = 'tk' and isdeleted = 0 INTO count_user;
    IF count_user = 0 THEN
        INSERT acmvalue SET createdBy = player1, name = 'tk';
    END IF;
    SELECT id from acmvalue where name = 'tk' and isdeleted = 0 INTO acmvalueid_tk;
    SELECT count(*) from acmvalue where name = 'ts' and isdeleted = 0 INTO count_user;
    IF count_user = 0 THEN
        INSERT acmvalue SET createdBy = player1, name = 'ts';
    END IF;
    SELECT id from acmvalue where name = 'ts' and isdeleted = 0 INTO acmvalueid_ts;
    SELECT count(*) from acmvalue where name = 'u' and isdeleted = 0 INTO count_user;
    IF count_user = 0 THEN
        INSERT acmvalue SET createdBy = player1, name = 'u';
    END IF;
    SELECT id from acmvalue where name = 'u' and isdeleted = 0 INTO acmvalueid_u;
    SELECT count(*) from acmvalue where name = 'USA' and isdeleted = 0 INTO count_user;
    IF count_user = 0 THEN
        INSERT acmvalue SET createdBy = player1, name = 'USA';
    END IF;
    SELECT id from acmvalue where name = 'USA' and isdeleted = 0 INTO acmvalueid_USA;
    SELECT count(*) from acm where name = 'dissem_countries=USA;f_clearance=u' and isdeleted = 0 INTO count_user;
    IF count_user = 0 THEN
        INSERT acm SET createdBy = player1, name = 'dissem_countries=USA;f_clearance=u';
        SELECT id from acm where name = 'dissem_countries=USA;f_clearance=u' and isdeleted = 0 INTO acm1id;
        INSERT acmpart SET createdBy = player1, acmid = acm1id, acmkeyid = acmkeyid_dissem_countries, acmvalueid = acmvalueid_USA;
        INSERT acmpart SET createdBy = player1, acmid = acm1id, acmkeyid = acmkeyid_f_clearance, acmvalueid = acmvalueid_u;
    END IF;
    SELECT id from acm where name = 'dissem_countries=USA;f_clearance=u' and isdeleted = 0 INTO acm1id;
    SELECT count(*) from acm where name = 'dissem_countries=USA;f_clearance=ts;f_sci_ctrls=si,tk' and isdeleted = 0 INTO count_user;
    IF count_user = 0 THEN
        INSERT acm SET createdBy = player1, name = 'dissem_countries=USA;f_clearance=ts;f_sci_ctrls=si,tk';
        SELECT id from acm where name = 'dissem_countries=USA;f_clearance=ts;f_sci_ctrls=si,tk' and isdeleted = 0 INTO acm2id;
        INSERT acmpart SET createdBy = player1, acmid = acm2id, acmkeyid = acmkeyid_dissem_countries, acmvalueid = acmvalueid_USA;
        INSERT acmpart SET createdBy = player1, acmid = acm2id, acmkeyid = acmkeyid_f_clearance, acmvalueid = acmvalueid_ts;
        INSERT acmpart SET createdBy = player1, acmid = acm2id, acmkeyid = acmkeyid_f_sci_ctrls, acmvalueid = acmvalueid_si;
        INSERT acmpart SET createdBy = player1, acmid = acm2id, acmkeyid = acmkeyid_f_sci_ctrls, acmvalueid = acmvalueid_tk;
    END IF;
    SELECT id from acm where name = 'dissem_countries=USA;f_clearance=ts;f_sci_ctrls=si,tk' and isdeleted = 0 INTO acm2id;

    IF LENGTH(assignedParentId) > 0 THEN
        SET parent_id := unhex(assignedParentId);
    ELSE
        SET parent_id := null;
    END IF;

	WHILE (cur_object < max_object) DO

		# Increment object counter
		SET cur_object := cur_object + 1;

		# Increment type counter
		SET cur_object_type := cur_object_type + 1;
		IF cur_object_type > max_object_type THEN
			SET cur_object_type := 1;
		END IF;

		# Type name
		SET new_type_name = concat('Test Type ', cur_object_type);
		# Make type if not exits
		IF cur_object_type = cur_object THEN
			SET new_type_id_hex := '';
			SELECT hex(id) FROM object_type WHERE name = new_type_name AND isDeleted = 0 INTO new_type_id_hex;
			IF new_type_id_hex IS NULL OR new_type_id_hex = '' THEN
				INSERT object_type SET createdBy = player1, name = new_type_name;
				SELECT hex(id) FROM object_type WHERE name = new_type_name AND isDeleted = 0 INTO new_type_id_hex;
			END IF;
			SET object_type_ids_hex := concat(object_type_ids_hex, ',', new_type_id_hex);
		END IF;

		# Get Type ID
		# Direct Position using cache
		SET new_type_id := unhex(substring(object_type_ids_hex, (2 + ((cur_object_type-1) * 33)), 32));

		# Object name
		SET new_object_name = concat(prefix, 'Object #', cur_object, suffix);
        
        # Determine user to be createdBy and acm
        SET createdBy := player1;
        SET otherUser := player2;
        SET granteeForCreatedBy := grantee1;
        SET granteeForOtherUser := grantee2;
        SET new_object_acm = acm2;
        IF ((cur_object mod 2) > 0) THEN
            SET new_object_acm = acm1;
            SET createdBy := player2;
            SET otherUser := player1;
            SET granteeForCreatedBy := grantee2;
            SET granteeForOtherUser := grantee1;
        END IF;

        SET acmShareForCreatedBy := CONCAT('{"users":["', createdBy, '"]}');
        SET acmShareForOtherUser := CONCAT('{"users":["', otherUser, '"]}');


        INSERT object SET createdBy = createdBy, name = new_object_name, rawacm = new_object_acm, typeId = new_type_id, parentId = parent_id;
        SELECT id FROM object WHERE name = new_object_name AND isDeleted = 0 LIMIT 1 INTO new_object_id;
        # Associate related acmkey and acmvalue
        IF ((cur_object mod 2) > 0) THEN
            INSERT INTO objectacm SET createdBy = createdBy, objectId = new_object_id, acmid = acm1id;
            # dissem_countries = USA, f_clearance = u
            #INSERT INTO object_acm SET createdBy = createdBy, objectId = new_object_id, acmkeyid = acmkeyid_dissem_countries, acmvalueid = acmvalueid_USA;
            #INSERT INTO object_acm SET createdBy = createdBy, objectId = new_object_id, acmkeyid = acmkeyid_f_clearance, acmvalueid = acmvalueid_u;
        ELSE
            INSERT INTO objectacm SET createdBy = createdBy, objectId = new_object_id, acmid = acm2id;
            # dissem_countries = USA, f_clearance = ts, f_sci_ctrls = si, tk
            #INSERT INTO object_acm SET createdBy = createdBy, objectId = new_object_id, acmkeyid = acmkeyid_dissem_countries, acmvalueid = acmvalueid_USA;
            #INSERT INTO object_acm SET createdBy = createdBy, objectId = new_object_id, acmkeyid = acmkeyid_f_clearance, acmvalueid = acmvalueid_ts;
            #INSERT INTO object_acm SET createdBy = createdBy, objectId = new_object_id, acmkeyid = acmkeyid_f_sci_ctrls, acmvalueid = acmvalueid_si;
            #INSERT INTO object_acm SET createdBy = createdBy, objectId = new_object_id, acmkeyid = acmkeyid_f_sci_ctrls, acmvalueid = acmvalueid_tk;
        END IF;                
        # Create permission records for access (grant full crud to creator, and explicit share read to other)
        INSERT INTO object_permission SET createdBy = createdBy, objectId = new_object_id, grantee = granteeForCreatedBy, acmShare = acmShareForCreatedBy, allowCreate = 1, allowRead = 1, allowUpdate = 1, allowDelete = 1, allowShare = 1, explicitShare = 0;
        INSERT INTO object_permission SET createdBy = createdBy, objectId = new_object_id, grantee = granteeForOtherUser, acmShare = acmShareForOtherUser, allowCreate = 0, allowRead = 1, allowUpdate = 0, allowDelete = 0, allowShare = 0, explicitShare = 1;
        # Every third object we'll share to everyone'
        IF ((cur_object mod 3) = 0) THEN
            INSERT INTO object_permission SET createdBy = createdBy, objectId = new_object_id, grantee = granteeForEveryone, acmShare = acmShareForEveryone, allowCreate = 0, allowRead = 1, allowUpdate = 0, allowDelete = 0, allowShare = 0, explicitShare = 1;
        END IF;

	END WHILE;
END
//
delimiter ;
