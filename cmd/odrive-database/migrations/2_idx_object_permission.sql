-- +migrate Up
CREATE INDEX ix_object_permission_createdby ON object_permission (createdBy);
CREATE INDEX ix_object_permission_allowread ON object_permission (allowRead);

-- +migrate Down
ALTER TABLE object_permission DROP FOREIGN KEY fk_object_permission_createdBy;

DROP INDEX `ix_object_permission_createdby` ON object_permission;
DROP INDEX `ix_object_permission_allowread` ON object_permission;

ALTER TABLE object_permission
	ADD CONSTRAINT fk_object_permission_createdBy FOREIGN KEY (createdBy) REFERENCES user(distinguishedName);
