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
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/models"
	encryptionService "github.com/percona/pmm/managed/services/encryption"
	"github.com/percona/pmm/managed/utils/encryption"
	"github.com/percona/pmm/managed/utils/secrets"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/utils/vault"
	"github.com/percona/pmm/version"
)

const codeDBConnectionFailed = 1

func main() {
	signal.Ignore(syscall.SIGINT, syscall.SIGTERM) // to prevent any interuptions during process

	logger.SetupGlobalLogger()

	var opts flags
	kong.Parse(
		&opts,
		kong.Name("encryption-rotation"),
		kong.Description("Version "+version.Version),
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

	if opts.GenerateKey {
		e := &encryption.Encryption{}
		key, err := e.GenerateKey()
		if err != nil {
			logrus.Errorf("Failed to generate key: %v", err)
			os.Exit(1)
		}
		fmt.Print(key) //nolint:forbidigo
		os.Exit(0)
	}

	// It runs with `docker exec`, which gives it the environment of the container as it was given: the
	// references that PMM Server read from Vault at startup are read here again.
	err := configureVault()
	if err != nil {
		logrus.Error(err)
		os.Exit(codeDBConnectionFailed)
	}
	opts.DBPassword, err = secrets.Resolve(context.Background(), opts.DBPassword)
	if err != nil {
		logrus.Error(err)
		os.Exit(codeDBConnectionFailed)
	}

	sqlDB, err := models.OpenDB(setupParams(opts))
	if err != nil {
		logrus.Error(err)
		os.Exit(codeDBConnectionFailed)
	}

	statusCode, err := encryptionService.RotateEncryptionKey(sqlDB, "pmm-managed")
	_ = sqlDB.Close()
	if err != nil {
		logrus.Error(err)
		os.Exit(statusCode)
	}
}

//nolint:lll
type flags struct {
	Address     string `name:"postgres-addr" default:"${address}" help:"PostgreSQL address with port"`
	DBName      string `name:"postgres-name" default:"pmm-managed" help:"PostgreSQL database name"`
	DBUsername  string `name:"postgres-username" default:"pmm-managed" help:"PostgreSQL database username name"`
	DBPassword  string `name:"postgres-password" default:"pmm-managed" env:"PMM_POSTGRES_DBPASSWORD" help:"PostgreSQL database password"`
	SSLMode     string `name:"postgres-ssl-mode" default:"${disable_sslmode}" help:"PostgreSQL SSL mode" enum:"${disable_sslmode}, ${require_sslmode},${verify_sslmode}, ${verify_full_sslmode}"`
	SSLCAPath   string `name:"postgres-ssl-ca-path" help:"PostgreSQL SSL CA root certificate path" type:"path"`
	SSLKeyPath  string `name:"postgres-ssl-key-path" help:"PostgreSQL SSL key path" type:"path"`
	SSLCertPath string `name:"postgres-ssl-cert-path" help:"PostgreSQL SSL certificate path" type:"path"`
	GenerateKey bool   `name:"generate-key" help:"Only generate a new encryption key and print to stdout"`
}

// configureVault sets up reading the encryption key and the database password from Vault, when PMM Server
// reads them from there.
func configureVault() error {
	if os.Getenv("PMM_VAULT_ADDRESS") == "" {
		return nil
	}

	cfg, err := secrets.ConfigFromEnv(os.Getenv)
	if err != nil {
		return err
	}
	client, err := vault.New(cfg)
	if err != nil {
		return err
	}
	secrets.Configure(client, cfg.EffectiveCacheTTL())
	return nil
}

func setupParams(opts flags) models.SetupDBParams {
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
