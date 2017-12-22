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
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	servicelib "github.com/percona/kardianos-service"
	"github.com/percona/pmm/proto"
	"github.com/percona/pmm/proto/config"
	"github.com/pkg/errors"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/supervisor"
	"github.com/percona/pmm-managed/utils/logger"
)

// affects only initial agent registration; after this, it should be changed manually in the config file
const (
	// qanAgentLoggingLevel = "debug"
	qanAgentLoggingLevel = "info"
)

type Service struct {
	baseDir    string
	supervisor *supervisor.Supervisor
	qanAPI     *http.Client
}

func NewService(ctx context.Context, baseDir string, supervisor *supervisor.Supervisor) (*Service, error) {
	svc := &Service{
		baseDir:    baseDir,
		supervisor: supervisor,
		qanAPI:     new(http.Client),
	}
	return svc, nil
}

// ensureAgentIsRegistered registers a single qan-agent instance on PMM Server node in QAN.
// It does not re-register or change configuration if agent is already registered.
// QAN API URL is always returned when no error is encountered.
func (svc *Service) ensureAgentIsRegistered(ctx context.Context) (*url.URL, error) {
	qanURL, err := getQanURL(ctx)
	if err != nil {
		return nil, err
	}

	// do not change anything if qan-agent is already registered
	path := filepath.Join(svc.baseDir, "config", "agent.conf")
	if _, err = os.Stat(path); err == nil {
		logger.Get(ctx).Debugf("qan-agent already registered (%s exists).", path)
		return qanURL, nil
	}

	path = filepath.Join(svc.baseDir, "bin", "percona-qan-agent-installer")
	args := []string{"-debug", "-hostname=pmm-server"}
	if qanURL.User != nil && qanURL.User.Username() != "" {
		args = append(args, "-server-user="+qanURL.User.Username())
		pass, _ := qanURL.User.Password()
		args = append(args, "-server-pass="+pass)
	}
	args = append(args, qanURL.String()) // full URL, with username and password (yes, again! that's how installer is written)
	cmd := exec.Command(path, args...)
	logger.Get(ctx).Debug(strings.Join(cmd.Args, " "))
	b, err := cmd.CombinedOutput()
	if err != nil {
		logger.Get(ctx).Infof("%s", b)
		return nil, errors.Wrap(err, "failed to register qan-agent")
	}
	logger.Get(ctx).Debugf("%s", b)

	// set logging level to the specified one, very useful for debugging
	path = filepath.Join(svc.baseDir, "config", "log.conf")
	if err = ioutil.WriteFile(path, []byte(fmt.Sprintf(`{"Level":%q,"Offline":"false"}`, qanAgentLoggingLevel)), 0666); err != nil {
		return nil, errors.Wrap(err, "failed to write log.conf")
	}
	return qanURL, nil
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
func (svc *Service) getOSUUID(ctx context.Context, qanURL *url.URL, agentUUID string) (string, error) {
	url := *qanURL
	url.Path = path.Join(url.Path, "instances", agentUUID)
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return "", errors.WithStack(err)
	}
	rb, _ := httputil.DumpRequestOut(req, true)
	logger.Get(ctx).Debugf("getOSUUID request:\n\n%s\n", rb)

	resp, err := svc.qanAPI.Do(req)
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer resp.Body.Close()

	rb, _ = httputil.DumpResponse(resp, true)
	if resp.StatusCode != 200 {
		logger.Get(ctx).Errorf("getOSUUID response:\n\n%s\n", rb)
		return "", errors.Errorf("unexpected QAN response status code %d", resp.StatusCode)
	}
	logger.Get(ctx).Debugf("getOSUUID response:\n\n%s\n", rb)

	var instance proto.Instance
	if err = json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		return "", errors.WithStack(err)
	}
	return instance.ParentUUID, nil
}

// addInstance adds instance to QAN API.
// If successful, instance UUID will be set.
func (svc *Service) addInstance(ctx context.Context, qanURL *url.URL, instance *proto.Instance) error {
	b, err := json.Marshal(instance)
	if err != nil {
		return errors.WithStack(err)
	}

	url := *qanURL
	url.Path = path.Join(url.Path, "instances")
	req, err := http.NewRequest("POST", url.String(), bytes.NewReader(b))
	if err != nil {
		return errors.WithStack(err)
	}
	req.Header.Set("Content-Type", "application/json")
	rb, _ := httputil.DumpRequestOut(req, true)
	logger.Get(ctx).Debugf("addInstance request:\n\n%s\n", rb)

	resp, err := svc.qanAPI.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()

	rb, _ = httputil.DumpResponse(resp, true)
	if resp.StatusCode != 201 {
		logger.Get(ctx).Errorf("addInstance response:\n\n%s\n", rb)
		return errors.Errorf("unexpected QAN response status code %d", resp.StatusCode)
	}
	logger.Get(ctx).Debugf("addInstance response:\n\n%s\n", rb)

	// Response Location header looks like this: http://127.0.0.1/qan-api/instances/6cea8824082d4ade682b94109664e6a9
	// Extract UUID directly from it instead of following it.
	parts := strings.Split(resp.Header.Get("Location"), "/")
	instance.UUID = parts[len(parts)-1]
	return nil
}

// removeInstance removes instance from QAN API.
func (svc *Service) removeInstance(ctx context.Context, qanURL *url.URL, uuid string) error {
	url := *qanURL
	url.Path = path.Join(url.Path, "instances", uuid)
	req, err := http.NewRequest("DELETE", url.String(), nil)
	if err != nil {
		return errors.WithStack(err)
	}
	rb, _ := httputil.DumpRequestOut(req, true)
	logger.Get(ctx).Debugf("removeInstance request:\n\n%s\n", rb)

	resp, err := svc.qanAPI.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()

	rb, _ = httputil.DumpResponse(resp, true)
	if resp.StatusCode != 204 {
		logger.Get(ctx).Errorf("removeInstance response:\n\n%s\n", rb)
		return errors.Errorf("unexpected QAN response status code %d", resp.StatusCode)
	}
	logger.Get(ctx).Debugf("removeInstance response:\n\n%s\n", rb)
	return nil
}

// ensureAgentRuns checks qan-agent process status and starts it if it is not configured or down.
func (svc *Service) ensureAgentRuns(ctx context.Context, nameForSupervisor string, port uint16) error {
	err := svc.supervisor.Status(ctx, nameForSupervisor)
	if err != nil {
		err = svc.supervisor.Stop(ctx, nameForSupervisor)
		if err != nil {
			logger.Get(ctx).Warn(err)
		}

		config := &servicelib.Config{
			Name:        nameForSupervisor,
			DisplayName: nameForSupervisor,
			Description: nameForSupervisor,
			Executable:  filepath.Join(svc.baseDir, "bin", "percona-qan-agent"),
			Arguments: []string{
				fmt.Sprintf("-listen=127.0.0.1:%d", port),
			},
		}
		err = svc.supervisor.Start(ctx, config)
	}
	return err
}

func (svc *Service) sendQANCommand(ctx context.Context, qanURL *url.URL, agentUUID string, command string, data []byte) error {
	cmd := proto.Cmd{
		User:      "pmm-managed",
		AgentUUID: agentUUID,
		Service:   "qan",
		Cmd:       command,
		Data:      data,
	}
	b, err := json.Marshal(cmd)
	if err != nil {
		return errors.WithStack(err)
	}

	// Send the command to the API which relays it to the agent, then relays the agent's reply back to here.
	// It takes a few seconds for agent to connect to QAN API once it is started via service manager.
	// QAN API fails to start/stop unconnected agent for QAN, so we retry the request when getting 404 response.
	const attempts = 10
	url := *qanURL
	url.Path = path.Join(url.Path, "agents", agentUUID, "cmd")
	for i := 0; i < attempts; i++ {
		req, err := http.NewRequest("PUT", url.String(), bytes.NewReader(b))
		if err != nil {
			return errors.WithStack(err)
		}
		req.Header.Set("Content-Type", "application/json")
		rb, _ := httputil.DumpRequestOut(req, true)
		logger.Get(ctx).Debugf("sendQANCommand request:\n\n%s\n", rb)

		resp, err := svc.qanAPI.Do(req)
		if err != nil {
			return errors.WithStack(err)
		}
		rb, _ = httputil.DumpResponse(resp, true)
		resp.Body.Close()

		if resp.StatusCode == 200 {
			logger.Get(ctx).Debugf("sendQANCommand response:\n\n%s\n", rb)
			return nil
		}
		if resp.StatusCode == 404 {
			logger.Get(ctx).Debugf("sendQANCommand response:\n\n%s\n", rb)
			time.Sleep(time.Second)
			continue
		}

		logger.Get(ctx).Errorf("sendQANCommand response:\n\n%s\n", rb)
		return errors.Errorf("%s: unexpected QAN API response status code %d", command, resp.StatusCode)
	}

	return errors.Errorf("%s: failed to send command after %d attempts", command, attempts)
}

// AddMySQL adds MySQL instance to QAN, configuring and enabling it.
// It sets MySQL instance UUID to qanAgent.QANDBInstanceUUID.
func (svc *Service) AddMySQL(ctx context.Context, rdsNode *models.RDSNode, rdsService *models.RDSService, qanAgent *models.QanAgent) error {
	qanURL, err := svc.ensureAgentIsRegistered(ctx)
	if err != nil {
		return err
	}

	agentUUID, err := svc.getAgentUUID()
	if err != nil {
		return err
	}
	osUUID, err := svc.getOSUUID(ctx, qanURL, agentUUID)
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
	if err = svc.addInstance(ctx, qanURL, instance); err != nil {
		return err
	}
	qanAgent.QANDBInstanceUUID = pointer.ToString(instance.UUID)

	// we need real DSN (with password) for qan-agent to work, and it seems to be the only way to pass it
	path := filepath.Join(svc.baseDir, "instance", fmt.Sprintf("%s.json", instance.UUID))
	instance.DSN = qanAgent.DSN(rdsService)
	b, err := json.MarshalIndent(instance, "", "    ")
	if err != nil {
		return errors.WithStack(err)
	}
	if err = ioutil.WriteFile(path, b, 0666); err != nil {
		return errors.WithStack(err)
	}

	if err = svc.ensureAgentRuns(ctx, qanAgent.NameForSupervisor(), *qanAgent.ListenPort); err != nil {
		return err
	}

	command := "StartTool"
	config := map[string]interface{}{
		"UUID":           instance.UUID,
		"CollectFrom":    "perfschema",
		"Interval":       60,
		"ExampleQueries": true,
	}
	b, err = json.Marshal(config)
	if err != nil {
		return errors.WithStack(err)
	}
	logger.Get(ctx).Debugf("%s %s %s", agentUUID, command, b)
	return svc.sendQANCommand(ctx, qanURL, agentUUID, command, b)
}

func (svc *Service) RemoveMySQL(ctx context.Context, qanAgent *models.QanAgent) error {
	qanURL, err := svc.ensureAgentIsRegistered(ctx)
	if err != nil {
		return err
	}

	// agent should be running to remove instance from it
	if err = svc.ensureAgentRuns(ctx, qanAgent.NameForSupervisor(), *qanAgent.ListenPort); err != nil {
		return err
	}

	agentUUID, err := svc.getAgentUUID()
	if err != nil {
		return err
	}

	command := "StopTool"
	b := []byte(*qanAgent.QANDBInstanceUUID)
	logger.Get(ctx).Debugf("%s %s %s", agentUUID, command, b)
	if err = svc.sendQANCommand(ctx, qanURL, agentUUID, command, b); err != nil {
		return err
	}

	return svc.removeInstance(ctx, qanURL, *qanAgent.QANDBInstanceUUID)

	// we do not stop qan-agent even if it has zero MySQL instances now - to be safe
}
