// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

// Package qan contains business logic of working with QAN and qan-agent on PMM Server node.
package qan

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/percona/kardianos-service"
	"github.com/percona/pmm/proto"
	"github.com/percona/pmm/proto/config"
	"github.com/pkg/errors"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/supervisor"
	"github.com/percona/pmm-managed/utils/logger"
)

const (
	// bypass nginx's HTTP Basic auth
	qanAPI = "http://127.0.0.1:9001"
)

type Service struct {
	baseDir    string
	supervisor *supervisor.Supervisor
	qanAPI     *http.Client
}

func NewService(baseDir string, supervisor *supervisor.Supervisor) *Service {
	return &Service{
		baseDir:    baseDir,
		supervisor: supervisor,
		qanAPI:     new(http.Client),
	}
}

// EnsureAgentIsRegistered is registers qan-agent running on PMM Server node in QAN.
// It does nothing if agent is already registered.
func (svc *Service) EnsureAgentIsRegistered(ctx context.Context) error {
	// do nothing is qan-agent already registered
	path := filepath.Join(svc.baseDir, "config", "agent.conf")
	if _, err := os.Stat(path); err == nil {
		logger.Get(ctx).Debugf("qan-agent already registered (%s exists).", path)
		return nil
	}

	path = filepath.Join(svc.baseDir, "bin", "percona-qan-agent-installer")
	cmd := exec.Command(path, "-debug", "-hostname=pmm-server", qanAPI)
	logger.Get(ctx).Debug(strings.Join(cmd.Args, " "))
	b, err := cmd.CombinedOutput()
	if err == nil {
		logger.Get(ctx).Debugf("%s", b)
		return nil
	}
	logger.Get(ctx).Infof("%s", b)
	return errors.Wrap(err, "failed to register qan-agent")
}

// getAgentUUID returns agent UUID from the qan-agent configuration file.
func (svc *Service) getAgentUUID() (string, error) {
	path := filepath.Join(svc.baseDir, "config", "agent.conf")
	f, err := os.Open(path)
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer f.Close()

	var cfg config.Agent
	if err = json.NewDecoder(f).Decode(&cfg); err != nil {
		return "", errors.WithStack(err)
	}
	if cfg.UUID == "" {
		err = errors.Errorf("missing agent UUID in configuration file %s", path)
	}
	return cfg.UUID, err
}

// getOSUUID returns OS UUID from the QAN API.
func (svc *Service) getOSUUID(ctx context.Context, agentUUID string) (string, error) {
	url := qanAPI + "/instances/" + agentUUID
	resp, err := svc.qanAPI.Get(url)
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := httputil.DumpResponse(resp, true)
		logger.Get(ctx).Errorf("GET %s:\n%s", url, b)
		return "", errors.Errorf("unexpected QAN response status code %d", resp.StatusCode)
	}

	var instance proto.Instance
	if err = json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		return "", errors.WithStack(err)
	}
	return instance.ParentUUID, nil
}

func (svc *Service) addInstance(ctx context.Context, instance *proto.Instance) (string, error) {
	b, err := json.Marshal(instance)
	if err != nil {
		return "", errors.WithStack(err)
	}

	url := qanAPI + "/instances"
	resp, err := svc.qanAPI.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer resp.Body.Close()

	rb, _ := httputil.DumpResponse(resp, true)
	if resp.StatusCode != 201 {
		logger.Get(ctx).Errorf("POST %s\n%s\n%s", url, b, rb)
		return "", errors.Errorf("unexpected QAN response status code %d", resp.StatusCode)
	}
	logger.Get(ctx).Debugf("POST %s\n%s\n%s", url, b, rb)

	// Response Location header looks like this: http://127.0.0.1:9001/qan-api/instances/6cea8824082d4ade682b94109664e6a9
	// It is wrong - it should not have /qan-api/ part.
	// Extract UUID directly from it.
	parts := strings.Split(resp.Header.Get("Location"), "/")
	return parts[len(parts)-1], nil
}

func (svc *Service) ensureAgentRuns(ctx context.Context, qanAgent *models.QanAgent) error {
	name := qanAgent.NameForSupervisor()
	err := svc.supervisor.Status(ctx, name)
	if err != nil {
		err = svc.supervisor.Stop(ctx, name)
		if err != nil {
			logger.Get(ctx).Warn(err)
		}

		config := &service.Config{
			Name:        name,
			DisplayName: name,
			Description: name,
			Executable:  filepath.Join(svc.baseDir, "bin", "percona-qan-agent"),
			Arguments: []string{
				fmt.Sprintf("-listen=127.0.0.1:%d", *qanAgent.ListenPort),
			},
		}
		err = svc.supervisor.Start(ctx, config)
	}
	return err
}

func (svc *Service) AddMySQL(ctx context.Context, rdsNode *models.RDSNode, rdsService *models.RDSService, qanAgent *models.QanAgent) error {
	agentUUID, err := svc.getAgentUUID()
	if err != nil {
		return err
	}
	osUUID, err := svc.getOSUUID(ctx, agentUUID)
	if err != nil {
		return err
	}

	instance := &proto.Instance{
		Subsystem:  "mysql",
		ParentUUID: osUUID,
		Name:       rdsNode.Name,
		DSN:        sanitizeDSN(qanAgent.DSN(rdsService)),
		Version:    *rdsService.EngineVersion,
	}
	uuid, err := svc.addInstance(ctx, instance)
	if err != nil {
		return err
	}

	// we need real DSN (with password) for qan-agent to work, and it seems to be the only way to pass it
	path := filepath.Join(svc.baseDir, "instance", fmt.Sprintf("%s.json", uuid))
	instance.DSN = qanAgent.DSN(rdsService)
	b, err := json.MarshalIndent(instance, "", "    ")
	if err != nil {
		return errors.WithStack(err)
	}
	if err = ioutil.WriteFile(path, b, 0666); err != nil {
		return errors.WithStack(err)
	}

	if err = svc.ensureAgentRuns(ctx, qanAgent); err != nil {
		return err
	}

	return nil
}
