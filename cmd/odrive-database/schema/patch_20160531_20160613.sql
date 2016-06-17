# Remvoe logic for functions, triggers
source triggers.drop.sql
source functions.drop.sql


# New tables
source table.acm.create.sql
source table.acmpart.create.sql
source table.objectacm.create.sql

# New indexes
CREATE INDEX `ix_ownedBy` ON `object` (`ownedBy`);
CREATE INDEX `ix_ownedBy` ON `a_object` (`ownedBy`);

# New constraints
ALTER TABLE acm
	ADD CONSTRAINT fk_acm_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_acm_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_acm_modifiedBy FOREIGN KEY (modifiedBy) REFERENCES user(distinguishedName)
;

ALTER TABLE acmpart
	ADD CONSTRAINT fk_acmpart_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_acmpart_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_acmpart_modifiedBy FOREIGN KEY (modifiedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_acmpart_acmId FOREIGN KEY (acmId) REFERENCES acm(id)
	,ADD CONSTRAINT fk_acmpart_acmKeyId FOREIGN KEY (acmKeyId) REFERENCES acmkey(id)
	,ADD CONSTRAINT fk_acmpart_acmValueId FOREIGN KEY (acmValueId) REFERENCES acmvalue(id)
;

ALTER TABLE objectacm
	ADD CONSTRAINT fk_objectacm_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_objectacm_deletedBy FOREIGN KEY (deletedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_objectacm_modifiedBy FOREIGN KEY (modifiedBy) REFERENCES user(distinguishedName)
	,ADD CONSTRAINT fk_objectacm_objectId FOREIGN KEY (objectId) REFERENCES object(id)
	,ADD CONSTRAINT fk_objectacm_acmId FOREIGN KEY (acmId) REFERENCES acm(id)
;

# Functions and triggers recreated
source functions.create.sql
source triggers.create.sql

# Transformation
source spTransformSchema20160613.sql
call sp_TransformSchema20160613();
drop procedure sp_TransformSchema20160613;

# Update schema version
update dbstate set schemaVersion = '20160613';