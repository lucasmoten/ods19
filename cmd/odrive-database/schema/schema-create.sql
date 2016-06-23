# Drop logic objects
source triggers.drop.sql
source functions.drop.sql
source constraints.drop.sql

# Set collation
ALTER DATABASE CHARACTER SET utf8 COLLATE utf8_unicode_ci;
SET character_set_client = utf8;
SET collation_connection = @@collation_database;

# Create objects
source tables.create.sql
source constraints.create.sql
source functions.create.sql
source triggers.create.sql

# Initialize DB state
source dbstate-init.sql

