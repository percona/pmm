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
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/agent/agents/supervisor"
	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/versioner"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents"
	"github.com/percona/pmm/managed/services/victoriametrics"
	"github.com/percona/pmm/version"
)

// offlineConfig is the file `pmm-agent offline-run` reads instead of asking PMM Server what to run.
type offlineConfig struct {
	NodeName     string           `yaml:"node_name"`      // default: hostname
	QueueDir     string           `yaml:"queue_dir"`      // default: <paths_base>/offline-queue
	MaxDiskUsage string           `yaml:"max_disk_usage"` // e.g. 5GiB; default: PMM's 1 GiB, oldest data dropped beyond it
	MySQL        []offlineService `yaml:"mysql"`
	MongoDB      []offlineService `yaml:"mongodb"`
	PostgreSQL   []offlineService `yaml:"postgresql"`
}

// offlineService is a database to monitor, with pmm-admin's defaults for anything left out.
type offlineService struct {
	ServiceName         string `yaml:"service_name"`
	Address             string `yaml:"address"` // default: 127.0.0.1
	Port                uint16 `yaml:"port"`    // default: the database's standard port
	Username            string `yaml:"username"`
	Password            string `yaml:"password"`
	Database            string `yaml:"database"`              // PostgreSQL only; default: postgres
	DisableTablestats   bool   `yaml:"disable_tablestats"`    // MySQL only
	EnableAllCollectors bool   `yaml:"enable_all_collectors"` // MongoDB only
}

// offlineAgentStorage loads pmm-agent's own configuration; offline-run needs no configuration file.
func offlineAgentStorage(l *logrus.Entry) *config.Storage {
	storage := config.NewStorage(nil)
	var notExist config.ConfigFileDoesNotExistError
	if _, err := storage.Reload(l); err != nil && !errors.As(err, &notExist) {
		l.Fatalf("Failed to load configuration: %s.", err)
	}
	config.ConfigureLogger(storage.Get())
	return storage
}

// OfflineRun runs node_exporter, the exporters listed in path and vmagent the way PMM Server would configure them,
// but without PMM Server: vmagent keeps every sample in its on-disk queue for `pmm-agent offline-push`.
func OfflineRun(path string) {
	l := logrus.WithField("component", "offline-run")
	ctx, stop := signal.NotifyContext(context.Background(), unix.SIGTERM, unix.SIGINT)
	defer stop()

	storage := offlineAgentStorage(l)
	b, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		l.Fatal(err)
	}
	var oc offlineConfig
	if err = yaml.Unmarshal(b, &oc); err != nil {
		l.Fatalf("%s: %s", path, err)
	}
	if oc.NodeName == "" {
		oc.NodeName, _ = os.Hostname()
	}
	if oc.QueueDir == "" {
		oc.QueueDir = filepath.Join(storage.Get().Paths.PathsBase, "offline-queue")
	}

	s := supervisor.NewSupervisor(ctx, versioner.New(&versioner.RealExecFunctions{}), storage)
	stopped := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(stopped)
	}()
	go func() {
		for c := range s.Changes() { // nobody else drains it without PMM Server
			l.Infof("Agent %s: %s.", c.AgentId, c.Status)
		}
	}()

	if err = startOffline(ctx, s, &oc); err != nil {
		l.Fatal(err)
	}
	l.Infof("Collecting into %s. Stop pmm-agent before copying that directory.", oc.QueueDir)
	<-stopped
}

// startOffline does what PMM Server does for a new pmm-agent: start the exporters, then vmagent scraping them
// on the ports pmm-agent gave them.
func startOffline(ctx context.Context, s *supervisor.Supervisor, oc *offlineConfig) error {
	v, err := version.Parse(version.Version)
	if err != nil {
		return err
	}
	// Stable IDs keep the same series across pmm-agent restarts.
	id := func(kind, name string) string { return uuid.NewSHA1(uuid.NameSpaceOID, []byte(kind+"/"+name)).String() }
	exporter := func(t models.AgentType, name string) *models.Agent {
		return &models.Agent{AgentID: id(string(t), name), AgentType: t, ExporterOptions: models.ExporterOptions{PushMetrics: true}}
	}
	node := &models.Node{NodeID: id("node", oc.NodeName), NodeName: oc.NodeName, NodeType: models.GenericNodeType, Address: models.LocalhostAddr, Distro: "linux"}

	targets := []victoriametrics.OfflineTarget{{Agent: exporter(models.NodeExporterType, oc.NodeName)}}
	kinds := []struct {
		services    []offlineService
		serviceType models.ServiceType
		agentType   models.AgentType
		port        uint16
	}{
		{oc.MySQL, models.MySQLServiceType, models.MySQLdExporterType, 3306},
		{oc.MongoDB, models.MongoDBServiceType, models.MongoDBExporterType, 27017},
		{oc.PostgreSQL, models.PostgreSQLServiceType, models.PostgresExporterType, 5432},
	}
	for _, k := range kinds {
		for _, m := range k.services {
			if m.Address == "" {
				m.Address = models.LocalhostAddr
			}
			if m.Port == 0 {
				m.Port = k.port
			}
			if m.Database == "" && k.serviceType == models.PostgreSQLServiceType {
				m.Database = "postgres"
			}
			svc := &models.Service{
				ServiceID: id("service", m.ServiceName), ServiceName: m.ServiceName, ServiceType: k.serviceType,
				NodeID: node.NodeID, Address: &m.Address, Port: &m.Port, DatabaseName: m.Database,
			}
			a := exporter(k.agentType, m.ServiceName)
			a.ServiceID, a.Username, a.Password = &svc.ServiceID, &m.Username, &m.Password
			// Each builder reads only its own options; -1 is pmm-admin's default collections limit.
			a.MongoDBOptions = models.MongoDBOptions{CollectionsLimit: -1, EnableAllCollectors: m.EnableAllCollectors}
			if m.DisableTablestats {
				a.MySQLOptions.TableCountTablestatsGroupLimit = -1
			}
			targets = append(targets, victoriametrics.OfflineTarget{Service: svc, Agent: a})
		}
	}

	procs := make(map[string]*agentv1.SetStateRequest_AgentProcess)
	for _, t := range targets {
		if procs[t.Agent.AgentID], err = agents.ExporterProcess(node, t.Service, t.Agent, v); err != nil {
			return err
		}
	}

	s.SetState(ctx, &agentv1.SetStateRequest{AgentProcesses: procs})
	ports := make(map[string]uint16)
	for _, info := range s.AgentsList() {
		ports[info.AgentId] = uint16(info.ListenPort) //nolint:gosec
	}
	for _, t := range targets {
		t.Agent.ListenPort = new(ports[t.Agent.AgentID])
	}

	scrapeCfg, err := victoriametrics.OfflineScrapeConfig(node, targets, v)
	if err != nil {
		return err
	}
	procs[id("vmagent", oc.NodeName)] = agents.OfflineVMAgentProcess(string(scrapeCfg), oc.QueueDir, oc.MaxDiskUsage)
	s.SetState(ctx, &agentv1.SetStateRequest{AgentProcesses: procs})
	return nil
}

// OfflinePush sends what `pmm-agent offline-run` queued in dir to the PMM Server this pmm-agent is set up for,
// and returns once vmagent has nothing left to send.
func OfflinePush(dir string) {
	l := logrus.WithField("component", "offline-push")
	cfg := offlineAgentStorage(l).Get()
	if cfg.Server.Address == "" {
		l.Fatal("PMM Server address is not set: run `pmm-agent setup` first or pass --server-address.")
	}
	u := url.URL{Scheme: "https", Host: cfg.Server.Address, Path: "/victoriametrics/api/v1/write"}
	if cfg.Server.WithoutTLS {
		u.Scheme = "http"
	}

	// vmagent names a queue after the xxhash of its URL and drops queues of other URLs, so rename it to the real one.
	queues, _ := filepath.Glob(filepath.Join(dir, "persistent-queue", "1_*"))
	if len(queues) != 1 {
		l.Fatalf("Expected one queue in %s, found %d.", filepath.Join(dir, "persistent-queue"), len(queues))
	}
	queue := filepath.Join(dir, "persistent-queue", fmt.Sprintf("1_%016X", xxhash.Sum64String(u.String())))
	if err := os.Rename(queues[0], queue); err != nil {
		l.Fatal(err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		l.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	cmd := exec.Command(cfg.Paths.VMAgent, //nolint:gosec
		"-envflag.enable=true", "-envflag.prefix=VMAGENT_",
		"-remoteWrite.url="+u.String(), "-remoteWrite.tmpDataPath="+dir, "-httpListenAddr="+addr,
		"-remoteWrite.tlsInsecureSkipVerify="+strconv.FormatBool(cfg.Server.InsecureTLS))
	cmd.Env = append(os.Environ(), // not on the command line, where ps would show them
		"VMAGENT_remoteWrite_basicAuth_username="+cfg.Server.Username,
		"VMAGENT_remoteWrite_basicAuth_password="+cfg.Server.Password)
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err = cmd.Start(); err != nil {
		l.Fatal(err)
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	l.Infof("Pushing %s to %s.", dir, u.Host)
	for {
		select {
		case err = <-exited:
			l.Fatalf("vmagent exited before the queue was empty: %v.", err)
		case <-time.After(10 * time.Second):
		}
		bytes, blocks, err := vmagentPending(addr)
		if err != nil {
			l.Warn(err)
			continue
		}
		l.Infof("Left to push: %.0f bytes on disk, %.0f blocks in memory.", bytes, blocks)
		if bytes == 0 && blocks == 0 {
			break
		}
	}
	_ = cmd.Process.Signal(unix.SIGINT)
	<-exited
	l.Info("Done: the queue is empty.")
}

// vmagentPending returns what vmagent at addr still has to send.
func vmagentPending(addr string) (float64, float64, error) {
	resp, err := http.Get("http://" + addr + "/metrics") //nolint:noctx
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close() //nolint:errcheck
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}

	sums := make(map[string]float64)
	for line := range strings.SplitSeq(string(b), "\n") {
		name, _, ok := strings.Cut(line, "{")
		if ok && (name == "vmagent_remotewrite_pending_data_bytes" || name == "vmagent_remotewrite_pending_inmemory_blocks") {
			v, err := strconv.ParseFloat(line[strings.LastIndexByte(line, ' ')+1:], 64)
			if err != nil {
				return 0, 0, err
			}
			sums[name] += v
		}
	}
	if len(sums) != 2 {
		return 0, 0, errors.New("vmagent queue metrics are not there yet")
	}
	return sums["vmagent_remotewrite_pending_data_bytes"], sums["vmagent_remotewrite_pending_inmemory_blocks"], nil
}
