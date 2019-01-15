package main

import (
	"log"

	_ "github.com/kshvakov/clickhouse"

	"github.com/Percona-Lab/qan-api/migrations"
	// TODO: research alternatives. Ex.: https://github.com/go-reform/reform
	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/database/clickhouse"
	"github.com/golang-migrate/migrate/source/go_bindata"
	"github.com/jmoiron/sqlx"
)

// NewDB return updated db.
func NewDB(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("clickhouse", dsn)
	if err != nil {
		log.Fatal("Connection: ", err)
	}

	if err := runMigrations(dsn); err != nil {
		log.Fatal("Migrations: ", err)
	}
	log.Println("Migrations applied.")
	return db, nil
}

func runMigrations(dsn string) error {
	s := bindata.Resource(migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		})

	d, err := bindata.WithInstance(s)
	if err != nil {
		return err
	}
	log.Println("dsn: ", dsn)
	m, err := migrate.NewWithSourceInstance("go-bindata", d, dsn)
	if err != nil {
		return err
	}

	// run up to the latest migration
	err = m.Up()
	if err == migrate.ErrNoChange {
		return nil
	}
	return err
}
