package migrations

import (
	"embed"
	"io/fs"
)

// Embed MySQL migrations
//
//go:embed mysql/*.sql
var mysqlMigrations embed.FS

// Embed PostgreSQL migrations
//
//go:embed postgres/*.sql
var postgresMigrations embed.FS

// GetMySQLMigrations returns the embedded MySQL migration filesystem
func GetMySQLMigrations() fs.FS {
	mysqlFS, err := fs.Sub(mysqlMigrations, "mysql")
	if err != nil {
		panic("failed to create MySQL migrations sub-filesystem: " + err.Error())
	}
	return mysqlFS
}

// GetPostgresMigrations returns the embedded PostgreSQL migration filesystem
func GetPostgresMigrations() fs.FS {
	postgresFS, err := fs.Sub(postgresMigrations, "postgres")
	if err != nil {
		panic("failed to create PostgreSQL migrations sub-filesystem: " + err.Error())
	}
	return postgresFS
}
