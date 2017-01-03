package main

import (
	"fmt"

	"github.com/rubenv/sql-migrate"
	"github.com/urfave/cli"
)

func listMigrations(clictx *cli.Context) error {
	db, err := connect(clictx)
	if err != nil {
		return err
	}
	m := &migrate.AssetMigrationSource{
		Asset:    Asset,
		AssetDir: AssetDir,
		Dir:      "migrations",
	}

	allMigrations, err := m.FindMigrations()
	if err != nil {
		return err
	}

	notInstalledMigrations, _, err := migrate.PlanMigration(db.DB, "mysql", m, migrate.Up, 0)
	if err != nil {
		return err
	}

	fmt.Printf("Migrations Available:\n")
	fmt.Printf("------------------------------------------------------------\n")
	fmt.Printf("%5s   %-40s   %-10s\n", "Order", "Script Filename", "Applied")
	for idx, migration := range allMigrations {
		isNotInstalled := false
		for _, notInstalledMigration := range notInstalledMigrations {
			if notInstalledMigration.Id == migration.Id {
				isNotInstalled = true
				break
			}
		}
		fmt.Printf("%5d   %-40s   %t\n", idx, migration.Id, !isNotInstalled)
	}
	return nil
}

func migrateUp(clictx *cli.Context) error {
	db, err := connect(clictx)
	if err != nil {
		return err
	}
	m := &migrate.AssetMigrationSource{
		Asset:    AssetWithEnv,
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
		Asset:    AssetWithEnv,
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
