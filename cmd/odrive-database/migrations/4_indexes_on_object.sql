-- +migrate Up

-- Add indexes on fields we commonly rely upon for list/search operations

ALTER TABLE object 
      ADD INDEX ix_ownedBy (ownedBy)
    , ADD INDEX ix_modifiedDate (modifiedDate)
    , ADD INDEX ix_createdDate (createdDate);

-- +migrate Down

-- Remove the indexes

DROP INDEX ix_ownedBy ON object;
DROP INDEX ix_modifiedDate ON object;
DROP INDEX ix_createdDate ON object;
