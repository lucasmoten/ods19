package main

import (
	"errors"
	"fmt"

	"github.com/rubenv/sql-migrate"
	"github.com/urfave/cli"
)

func listMigrations(clictx *cli.Context) error {

	return errors.New("list command not implemented")
}

func migrateUp(clictx *cli.Context) error {
	db, err := connect(clictx)
	if err != nil {
		return err
	}
	m := &migrate.AssetMigrationSource{
		Asset:    Asset,
		AssetDir: AssetDir,
		Dir:      "migrations",
	}

	n, err := migrate.Exec(db.DB, "mysql", m, migrate.Up)
	if err != nil {
		return err
	}

	fmt.Printf("applied %v migrations up\n", n)
	return nil
}

func migrateDown(clictx *cli.Context) error {
	db, err := connect(clictx)
	if err != nil {
		return err
	}
	m := &migrate.AssetMigrationSource{
		Asset:    Asset,
		AssetDir: AssetDir,
		Dir:      "migrations",
	}

	// Apply exactly one migration down.
	n, err := migrate.ExecMax(db.DB, "mysql", m, migrate.Down, 1)
	if err != nil {
		return err
	}
	fmt.Printf("applied %v migrations down\n", n)

	return nil
}
