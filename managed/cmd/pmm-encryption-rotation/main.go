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
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
	encryptionService "github.com/percona/pmm/managed/services/encryption"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/version"
)

const codeDBConnectionFailed = 1

func main() {
	signal.Ignore(syscall.SIGINT, syscall.SIGTERM) // to prevent any interuptions during process

	logger.SetupGlobalLogger()

	logrus.Infof("PMM Encryption Rotation Tools version: %s", version.Version)

	sqlDB, err := models.OpenDB(setupParams())
	if err != nil {
		logrus.Error(err)
		os.Exit(codeDBConnectionFailed)
	}

	statusCode := encryptionService.RotateEncryptionKey(sqlDB, "pmm-managed")
	sqlDB.Close() //nolint:errcheck

	os.Exit(statusCode)
}

type flags struct {
	Address     string `name:"postgres-addr" default:"${address}" help:"PostgreSQL address with port"`
	DBName      string `name:"postgres-name" default:"pmm-managed" help:"PostgreSQL database name"`
	DBUsername  string `name:"postgres-username" default:"pmm-managed" help:"PostgreSQL database username name"`
	DBPassword  string `name:"postgres-password" default:"pmm-managed" help:"PostgreSQL database password"`
	SSLMode     string `name:"postgres-ssl-mode" default:"${disable_sslmode}" help:"PostgreSQL SSL mode" enum:"${disable_sslmode}, ${require_sslmode},${verify_sslmode}, ${verify_full_sslmode}"` //nolint:lll
	SSLCAPath   string `name:"postgres-ssl-ca-path" help:"PostgreSQL SSL CA root certificate path" type:"path"`
	SSLKeyPath  string `name:"postgres-ssl-key-path" help:"PostgreSQL SSL key path" type:"path"`
	SSLCertPath string `name:"postgres-ssl-cert-path" help:"PostgreSQL SSL certificate path" type:"path"`
}

func setupParams() models.SetupDBParams {
	var opts flags
	kong.Parse(
		&opts,
		kong.Name("encryption-rotation"),
		kong.Description(fmt.Sprintf("Version %s", version.Version)),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:             true,
			NoExpandSubcommands: true,
		}),
		kong.Vars{
			"address":             models.DefaultPostgreSQLAddr,
			"disable_sslmode":     models.DisableSSLMode,
			"require_sslmode":     models.RequireSSLMode,
			"verify_sslmode":      models.VerifyCaSSLMode,
			"verify_full_sslmode": models.VerifyFullSSLMode,
		},
	)

	return models.SetupDBParams{
		Address:     opts.Address,
		Name:        opts.DBName,
		Username:    opts.DBUsername,
		Password:    opts.DBPassword,
		SSLMode:     opts.SSLMode,
		SSLCAPath:   opts.SSLCAPath,
		SSLKeyPath:  opts.SSLKeyPath,
		SSLCertPath: opts.SSLCertPath,
	}
}
