package main

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// createSchema executes all necessary DDL. Any error is immediately returned.
func createSchema(db *sqlx.DB) error {

	// Drop triggers, functions, constraints
	if err := execFile(db, "schema/triggers.drop.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/functions.drop.sql"); err != nil {
		return err
	}
	if err := dropConstraints(db); err != nil {
		// TODO(cm) find a nicer way to do this on first run
		fmt.Println("ignoring constraint drop failure")
		fmt.Printf("err: %v", err)
	}
	if err := dropTables(db); err != nil {
		fmt.Println("ignoring table drop failure")
		fmt.Printf("err: %v", err)
	}

	// Set collation
	if err := execStmt(db, "ALTER DATABASE CHARACTER SET utf8 COLLATE utf8_unicode_ci"); err != nil {
		return err
	}
	if err := execStmt(db, "SET character_set_client = utf8"); err != nil {
		return err
	}
	if err := execStmt(db, "SET collation_connection = @@collation_database"); err != nil {
		return err
	}

	// Create tables
	if err := createTables(db); err != nil {
		return err
	}

	// Create constraints
	if err := createConstraints(db); err != nil {
		return err
	}

	// Create functions
	if err := createFunctions(db); err != nil {
		return err
	}

	// Create triggers
	if err := createTriggers(db); err != nil {
		return err
	}

	// Data for state
	if err := execStmt(db, "insert dbstate set modifiedDate = createdDate;"); err != nil {
		return err
	}

	return nil
}

// createConstraints invokes every required create constraint statement.
func createConstraints(db *sqlx.DB) error {

	// All our constraints run from a single, semicolon delimited file.
	if err := execFile(db, "schema/constraints.create.sql"); err != nil {
		return err
	}

	return nil
}

func dropConstraints(db *sqlx.DB) error {
	if err := execFileIgnoreError(db, "schema/constraints.drop.sql"); err != nil {
		return err
	}
	return nil
}

// createFunctions explicitly invokes every required create trigger statement.
func createFunctions(db *sqlx.DB) error {

	if err := execFile(db, "schema/function.ordered_uuid.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/function.bitwise256_xor.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/function.new_keydecrypt.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/function.old_keydecrypt.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/function.int2boolStr.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/function.pseudorand256.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/function.new_keymac.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/function.new_keymacdata.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/procedure.migrate_keys.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/procedure.rotate_keys.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/function.aacflatten.sql"); err != nil {
		return err
	}

	return nil

}

// createTables explicitly invokes every required create table statement.
func createTables(db *sqlx.DB) error {
	if err := execFile(db, "schema/table.acm.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.acmgrantee.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.acmkey.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.acmpart.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.acmvalue.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.dbstate.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.field_changes.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.object.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.objectacm.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.object_permission.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.object_property.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.object_tag.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.object_type.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.object_type_property.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.property.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.relationship.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.user.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.user_object_favorite.create.sql"); err != nil {
		return err
	}
	if err := execFile(db, "schema/table.user_object_subscription.create.sql"); err != nil {
		return err
	}
	return nil
}

func dropTables(db *sqlx.DB) error {
	if err := execFileIgnoreError(db, "schema/tables.drop.sql"); err != nil {
		return err
	}

	if err := execStmt(db, "drop table if exists gorp_migrations;"); err != nil {
		return err
	}

	return nil
}

// createTriggers explicitly invokes every required create trigger statement.
func createTriggers(db *sqlx.DB) error {

	if err := declareProc(db, "schema/triggers.acm.ti_acm.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.acm.tu_acm.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.acm.td_acm.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.acm.td_a_acm.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.acmkey.ti_acmkey.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.acmkey.tu_acmkey.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.acmkey.td_acmkey.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.acmkey.td_a_acmkey.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.acmpart.ti_acmpart.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.acmpart.tu_acmpart.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.acmpart.td_acmpart.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.acmpart.td_a_acmpart.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.acmvalue.ti_acmvalue.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.acmvalue.tu_acmvalue.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.acmvalue.td_acmvalue.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.acmvalue.td_a_acmvalue.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.dbstate.ti_dbstate.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.dbstate.tu_dbstate.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.dbstate.td_dbstate.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.field_changes.td_field_changes.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object.ti_object.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object.tu_object.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object.td_object.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object.td_a_object.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.objectacm.ti_objectacm.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.objectacm.tu_objectacm.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.objectacm.td_objectacm.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.objectacm.td_a_objectacm.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_permission.ti_object_permission.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_permission.tu_object_permission.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_permission.td_object_permission.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_permission.td_a_object_permission.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_property.ti_object_property.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_property.tu_object_property.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_property.td_object_property.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_tag.ti_object_tag.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_tag.tu_object_tag.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_tag.td_object_tag.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_tag.td_a_object_tag.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_type.ti_object_type.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_type.tu_object_type.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_type.td_object_type.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_type.td_a_object_type.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_type_property.ti_object_type_property.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_type_property.tu_object_type_property.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.object_type_property.td_object_type_property.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.property.ti_property.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.property.tu_property.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.property.td_property.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.property.td_a_property.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.relationship.ti_relationship.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.relationship.tu_relationship.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.relationship.td_relationship.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.relationship.td_a_relationship.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.user.ti_user.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.user.tu_user.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.user.td_user.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.user.td_a_user.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.user_object_favorite.ti_user_object_favorite.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.user_object_favorite.tu_user_object_favorite.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.user_object_favorite.td_user_object_favorite.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.user_object_favorite.td_a_user_object_favorite.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.user_object_subscription.ti_user_object_subscription.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.user_object_subscription.tu_user_object_subscription.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.user_object_subscription.td_user_object_subscription.create.sql"); err != nil {
		return err
	}
	if err := declareProc(db, "schema/triggers.user_object_subscription.td_a_user_object_subscription.create.sql"); err != nil {
		return err
	}

	return nil
}
