// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
// Package main is the main package for encryption keys rotation.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/Percona-Lab/kingpin"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/encryption"
)

func main() {
	sqlDB, dbName := openDB()
	defer sqlDB.Close() //nolint:errcheck

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	err := stopPMMServer()
	if err != nil {
		log.Printf("Failed to rotate encryption key: %+v", err)
		os.Exit(1)
	}

	err = rotateEncryptionKey(db, dbName)
	if err != nil {
		log.Printf("Failed to rotate encryption key: %+v", err)
		os.Exit(1)
	}

	err = startPMMServer()
	if err != nil {
		log.Printf("Failed to rotate encryption key: %+v", err)
		os.Exit(1)
	}
}

func startPMMServer() error {
	if isPMMServerStatus("RUNNING") {
		return nil
	}

	cmd := exec.Command("supervisorctl", "start pmm-managed")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, output)
	}

	if !isPMMServerStatus("RUNNING") {
		return errors.New("cannot start pmm-managed")
	}

	return nil
}

func stopPMMServer() error {
	if isPMMServerStatus("STOPPED") {
		return nil
	}

	cmd := exec.Command("supervisorctl", "stop pmm-managed")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, output)
	}

	if !isPMMServerStatus("STOPPED") {
		return errors.New("cannot stop pmm-managed")
	}

	return nil
}

func isPMMServerStatus(status string) bool {
	cmd := exec.Command("supervisorctl", "status pmm-managed")
	output, _ := cmd.CombinedOutput()

	return strings.Contains(string(output), strings.ToUpper(status))
}

func rotateEncryptionKey(db *reform.DB, dbName string) error {
	return db.InTransaction(func(tx *reform.TX) error {
		err := models.DecryptDB(tx, dbName, models.DefaultAgentEncryptionColumns)
		if err != nil {
			return err
		}

		err = encryption.RotateEncryptionKey()
		if err != nil {
			return err
		}

		err = models.EncryptDB(tx, dbName, models.DefaultAgentEncryptionColumns)
		if err != nil {
			if e := encryption.RestoreOldEncryptionKey(); e != nil {
				return errors.Wrap(e, e.Error())
			}
			return err
		}

		return nil
	})
}

func openDB() (*sql.DB, string) {
	postgresAddrF := kingpin.Flag("postgres-addr", "PostgreSQL address").
		Default(models.DefaultPostgreSQLAddr).
		Envar("PMM_POSTGRES_ADDR").
		String()
	postgresDBNameF := kingpin.Flag("postgres-name", "PostgreSQL database name").
		Default("pmm-managed").
		Envar("PMM_POSTGRES_DBNAME").
		String()
	postgresDBUsernameF := kingpin.Flag("postgres-username", "PostgreSQL database username").
		Default("pmm-managed").
		Envar("PMM_POSTGRES_USERNAME").
		String()
	postgresSSLModeF := kingpin.Flag("postgres-ssl-mode", "PostgreSQL SSL mode").
		Default(models.DisableSSLMode).
		Envar("PMM_POSTGRES_SSL_MODE").
		Enum(models.DisableSSLMode, models.RequireSSLMode, models.VerifyCaSSLMode, models.VerifyFullSSLMode)
	postgresSSLCAPathF := kingpin.Flag("postgres-ssl-ca-path", "PostgreSQL SSL CA root certificate path").
		Envar("PMM_POSTGRES_SSL_CA_PATH").
		String()
	postgresDBPasswordF := kingpin.Flag("postgres-password", "PostgreSQL database password").
		Default("pmm-managed").
		Envar("PMM_POSTGRES_DBPASSWORD").
		String()
	postgresSSLKeyPathF := kingpin.Flag("postgres-ssl-key-path", "PostgreSQL SSL key path").
		Envar("PMM_POSTGRES_SSL_KEY_PATH").
		String()
	postgresSSLCertPathF := kingpin.Flag("postgres-ssl-cert-path", "PostgreSQL SSL certificate path").
		Envar("PMM_POSTGRES_SSL_CERT_PATH").
		String()

	kingpin.Parse()

	setupParams := models.SetupDBParams{
		Address:     *postgresAddrF,
		Name:        *postgresDBNameF,
		Username:    *postgresDBUsernameF,
		Password:    *postgresDBPasswordF,
		SSLMode:     *postgresSSLModeF,
		SSLCAPath:   *postgresSSLCAPathF,
		SSLKeyPath:  *postgresSSLKeyPathF,
		SSLCertPath: *postgresSSLCertPathF,
	}

	sqlDB, err := models.OpenDB(setupParams)
	if err != nil {
		log.Panicf("Failed to connect to database: %+v", err)
	}

	return sqlDB, *postgresDBNameF
}
