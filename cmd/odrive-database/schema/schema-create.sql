use metadatadb;

# Drop logic objects
source triggers.drop.sql
source functions.drop.sql
source constraints.drop.sql

# Create objects
source tables.create.sql
source constraints.create.sql
source functions.create.sql
source triggers.create.sql

# Initialize DB state
source dbstate-init.sql

