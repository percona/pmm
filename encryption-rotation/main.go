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
	"os"
	"os/signal"
	"syscall"

	"github.com/Percona-Lab/kingpin"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/logger"
	"github.com/sirupsen/logrus"
)

func main() {
	signal.Ignore(syscall.SIGINT, syscall.SIGTERM) // to prevent any interuptions during process

	logger.SetupGlobalLogger()

	sqlDB, err := models.OpenDB(setupParams())
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	statusCode := models.RotateEncryptionKey(sqlDB, "pmm-managed")
	sqlDB.Close() //nolint:errcheck

	os.Exit(statusCode)
}

func setupParams() models.SetupDBParams {
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

	return models.SetupDBParams{
		Address:     *postgresAddrF,
		Name:        *postgresDBNameF,
		Username:    *postgresDBUsernameF,
		Password:    *postgresDBPasswordF,
		SSLMode:     *postgresSSLModeF,
		SSLCAPath:   *postgresSSLCAPathF,
		SSLKeyPath:  *postgresSSLKeyPathF,
		SSLCertPath: *postgresSSLCertPathF,
	}
}
