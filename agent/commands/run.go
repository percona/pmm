// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/percona/pmm/agent/agentlocal"
	"github.com/percona/pmm/agent/agents/supervisor"
	"github.com/percona/pmm/agent/client"
	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/connectionchecker"
	"github.com/percona/pmm/agent/connectionuptime"
	"github.com/percona/pmm/agent/runner"
	"github.com/percona/pmm/agent/serviceinfobroker"
	"github.com/percona/pmm/agent/tailog"
	"github.com/percona/pmm/agent/utils/secrets"
	"github.com/percona/pmm/agent/versioner"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/utils/vault"
)

// Run implements `pmm-agent run` default command.
func Run() {
	const initServerLogsMaxLength = 32 // store logs before load configuration
	logStore := tailog.NewStore(initServerLogsMaxLength)
	logrus.SetOutput(io.MultiWriter(os.Stderr, logStore))
	l := logrus.WithField("component", "main")
	rootCtx, rootCancel := context.WithCancel(context.Background())

	defer l.Info("Done.")

	// handle termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		l.Warnf("Got %s, shutting down...", unix.SignalName(s.(unix.Signal))) //nolint:forcetypeassert
		rootCancel()
	}()

	v := versioner.New(&versioner.RealExecFunctions{})
	configStorage, configFilepath := prepareConfig(l)

	for {
		ctx, cancel := context.WithCancel(rootCtx)
		cfg := configStorage.Get()

		prepareLogger(cfg, logStore, l)
		configureVault(cfg, l)

		supervisor := supervisor.NewSupervisor(ctx, v, configStorage)
		connectionChecker := connectionchecker.New(configStorage)
		serviceInfoBroker := serviceinfobroker.New(configStorage)
		r := runner.New(cfg.RunnerCapacity, cfg.RunnerMaxConnectionsPerService)
		client := client.New(configStorage, supervisor, r, connectionChecker, v, serviceInfoBroker, prepareConnectionService(ctx, cfg), logStore)
		localServer := agentlocal.NewServer(configStorage, supervisor, client, configFilepath, logStore)

		logrus.Infof("Window check connection time is %.2f hour(s)", cfg.WindowConnectedTime.Hours())

		var wg sync.WaitGroup
		wg.Add(3) //nolint:mnd
		reloadCh := make(chan bool, 1)
		go func() {
			defer wg.Done()
			supervisor.Run(ctx)
			cancel()
		}()
		go func() {
			defer wg.Done()
			r.Run(ctx)
			cancel()
		}()
		go func() {
			defer wg.Done()
			localServer.Run(ctx, reloadCh)
			cancel()
		}()

		processClientUntilCancel(ctx, client, reloadCh)

		cleanupTmp(cfg.Paths.TempDir, l)
		wg.Wait()
		select {
		case <-rootCtx.Done():
			return
		default:
		}
	}
}

func processClientUntilCancel(ctx context.Context, client *client.Client, reloadCh chan bool) {
	for {
		clientCtx, cancelClientCtx := context.WithCancel(ctx)
		err := client.Run(clientCtx)
		if err != nil {
			logrus.Errorf("Client error: %s", err)
		}

		cancelClientCtx()
		<-client.Done()

		select {
		case <-reloadCh:
			return
		case <-ctx.Done():
			return
		default:
		}
	}
}

func prepareConfig(l *logrus.Entry) (*config.Storage, string) {
	configStorage := config.NewStorage(nil)
	configFilepath, err := configStorage.Reload(l)
	if err != nil {
		l.Fatalf("Failed to load configuration: %s.", err)
	}

	if configFilepath != "" {
		cfg := configStorage.Get()
		err := config.SaveToFile(configFilepath, withStoredVaultCredentials(cfg, configFilepath), "")
		if err != nil {
			l.Warnf("Failed to save updated config: %s", err)
		}
	}

	return configStorage, configFilepath
}

// withStoredVaultCredentials returns cfg with the Vault token and secret ID the configuration file holds,
// rather than those given by flags or the environment: an operator who passes them that way keeps them
// off the disk, and saving the configuration is not to undo that.
func withStoredVaultCredentials(cfg *config.Config, configFilepath string) *config.Config {
	stored := *cfg
	stored.Vault.Token = ""
	stored.Vault.SecretID = ""

	fileCfg, err := config.LoadFromFile(configFilepath, &cfg.Encryption)
	if err == nil {
		stored.Vault.Token = fileCfg.Vault.Token
		stored.Vault.SecretID = fileCfg.Vault.SecretID
	}
	return &stored
}

// configureVault sets up reading the credentials stored in Vault. An invalid configuration is reported
// when a credential is read, along with what reads it, rather than stopping pmm-agent here.
func configureVault(cfg *config.Config, l *logrus.Entry) {
	if !cfg.Vault.Enabled() {
		secrets.Configure(nil)
		return
	}

	client, err := vault.New(cfg.Vault)
	if err != nil {
		l.Errorf("Invalid Vault configuration: %s.", err)
		secrets.Configure(invalidVault{err: fmt.Errorf("invalid Vault configuration: %w", err)})
		return
	}

	client.SetLogger(l)

	l.Infof("Reading credentials stored in Vault from %s.", cfg.Vault.Address)
	secrets.Configure(client)
}

type invalidVault struct {
	err error
}

func (v invalidVault) Get(context.Context, string, string) (string, error) {
	return "", v.err
}

func (v invalidVault) Write(context.Context, string, map[string]string) error {
	return v.err
}

func prepareLogger(cfg *config.Config, logStore *tailog.Store, l *logrus.Entry) {
	config.ConfigureLogger(cfg)
	logStore.Resize(cfg.LogLinesCount)
	l.Debugf("Loaded configuration: %+v", cfg)
}

func cleanupTmp(tmpRoot string, log *logrus.Entry) {
	for k := range inventoryv1.AgentType_name {
		agentType := inventoryv1.AgentType(k)
		if agentType == inventoryv1.AgentType_AGENT_TYPE_UNSPECIFIED {
			continue
		}

		agentTmp := filepath.Join(tmpRoot, strings.TrimPrefix(strings.ToLower(agentType.String()), "agent_type_"))
		err := os.RemoveAll(agentTmp)
		if err != nil {
			log.Warnf("Failed to cleanup directory '%s': %s", agentTmp, err.Error())
		}
	}
	// Remove all other directories found in tmpRoot. They shouldn't be necessary once the agent is stopped.
	dirs, err := os.ReadDir(tmpRoot)
	if err != nil {
		log.Warnf("Failed to read directory '%s': %s", tmpRoot, err.Error())
		return
	}
	for _, dir := range dirs {
		if dir.IsDir() {
			dirPath := filepath.Join(tmpRoot, dir.Name())
			err := os.RemoveAll(dirPath)
			if err != nil {
				log.Warnf("Failed to cleanup directory '%s': %s", dirPath, err.Error())
			}
		}
	}
}

func prepareConnectionService(ctx context.Context, cfg *config.Config) *connectionuptime.Service {
	connectionUptimeService := connectionuptime.NewService(cfg.WindowConnectedTime)
	connectionUptimeService.RunCleanupGoroutine(ctx)

	return connectionUptimeService
}
