-- +migrate Up
CREATE INDEX idx_object_description ON object (description);

-- +migrate Down
DROP INDEX `idx_object_description` ON object;
