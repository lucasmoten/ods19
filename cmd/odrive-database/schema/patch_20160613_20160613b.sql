# Recreate triggers
source triggers.drop.sql
source triggers.create.sql

# Update schema version
update dbstate set schemaVersion = '20160613';
