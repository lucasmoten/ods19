-- +migrate Up
CREATE INDEX ix_object_permission_createdby ON object_permission (createdBy);
CREATE INDEX ix_object_permission_allowread ON object_permission (allowRead);

-- +migrate Down
DROP INDEX `ix_object_permission_createdby` ON object_permission;
DROP INDEX `ix_object_permission_allowread` ON object_permission;
