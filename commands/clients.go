// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package commands

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/AlekSi/pointer"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	agentlocalpb "github.com/percona/pmm/api/agentlocalpb/json/client"
	managementpb "github.com/percona/pmm/api/managementpb/json/client"
	"github.com/percona/pmm/api/managementpb/json/client/node"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm-agent/config"
)

// setLocalTransport configures transport for accessing local pmm-agent API.
//
// This method is not thread-safe.
func setLocalTransport(port uint16, l *logrus.Entry) {
	// use JSON APIs over HTTP/1.1
	transport := httptransport.New(fmt.Sprintf("127.0.0.1:%d", port), "/", []string{"http"})
	transport.SetLogger(l)
	transport.SetDebug(l.Logger.GetLevel() >= logrus.DebugLevel)
	transport.Context = context.Background()

	// disable HTTP/2
	httpTransport := transport.Transport.(*http.Transport)
	httpTransport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}

	agentlocalpb.Default.SetTransport(transport)
}

type statusResult struct {
	ConfigFilePath string
}

// localStatus returns locally running pmm-agent status.
// Error is returned if pmm-agent is not running.
//
// This method is not thread-safe.
func localStatus() (*statusResult, error) {
	res, err := agentlocalpb.Default.AgentLocal.Status(nil)
	if err != nil {
		return nil, err
	}

	return &statusResult{
		ConfigFilePath: res.Payload.ConfigFilePath,
	}, nil
}

// localReload reloads locally running pmm-agent.
//
// This method is not thread-safe.
func localReload() error {
	_, err := agentlocalpb.Default.AgentLocal.Reload(nil)
	return err
}

type errFromNginx string

func (e errFromNginx) Error() string {
	return "response from nginx: " + string(e)
}

func (e errFromNginx) GoString() string {
	return fmt.Sprintf("errFromNginx(%q)", string(e))
}

// setServerTransport configures transport for accessing PMM Server API.
//
// This method is not thread-safe.
func setServerTransport(u *url.URL, insecureTLS bool, l *logrus.Entry) {
	// use JSON APIs over HTTP/1.1
	transport := httptransport.New(u.Host, u.Path, []string{u.Scheme})
	// FIXME https://jira.percona.com/browse/PMM-3867
	if u.User != nil {
		logrus.Panic("PMM Server authentication is not implemented yet.")
	}
	transport.SetLogger(l)
	transport.SetDebug(l.Logger.GetLevel() >= logrus.DebugLevel)
	transport.Context = context.Background()

	// set error handlers for nginx responses if pmm-managed is down
	errorConsumer := runtime.ConsumerFunc(func(reader io.Reader, data interface{}) error {
		b, _ := ioutil.ReadAll(reader)
		return errFromNginx(string(b))
	})
	transport.Consumers = map[string]runtime.Consumer{
		runtime.JSONMime:    runtime.JSONConsumer(),
		runtime.HTMLMime:    errorConsumer,
		runtime.TextMime:    errorConsumer,
		runtime.DefaultMime: errorConsumer,
	}

	// disable HTTP/2, set TLS config
	httpTransport := transport.Transport.(*http.Transport)
	httpTransport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	if u.Scheme == "https" {
		if httpTransport.TLSClientConfig == nil {
			httpTransport.TLSClientConfig = new(tls.Config)
		}
		if insecureTLS {
			httpTransport.TLSClientConfig.InsecureSkipVerify = true
		} else {
			httpTransport.TLSClientConfig.ServerName = u.Hostname()
		}
	}

	managementpb.Default.SetTransport(transport)
}

// serverRegister registers Node on PMM Server.
//
// This method is not thread-safe.
func serverRegister(cfgSetup *config.Setup) (string, error) {
	nodeTypes := map[string]string{
		"generic":   node.RegisterBodyNodeTypeGENERICNODE,
		"container": node.RegisterBodyNodeTypeCONTAINERNODE,
	}

	res, err := managementpb.Default.Node.Register(&node.RegisterParams{
		Body: node.RegisterBody{
			NodeType:      pointer.ToString(nodeTypes[cfgSetup.NodeType]),
			NodeName:      cfgSetup.NodeName,
			MachineID:     cfgSetup.MachineID,
			Distro:        cfgSetup.Distro,
			ContainerID:   cfgSetup.ContainerID,
			ContainerName: cfgSetup.ContainerName,
			NodeModel:     cfgSetup.NodeModel,
			Region:        cfgSetup.Region,
			Az:            cfgSetup.Az,
			// TODO CustomLabels:  customLabels,
			Address: cfgSetup.Address,

			Reregister: cfgSetup.Force,
		},
		Context: context.Background(),
	})
	if err != nil {
		return "", err
	}
	return res.Payload.PMMAgent.AgentID, nil
}

// check interfaces
var (
	_ error          = errFromNginx("")
	_ fmt.GoStringer = errFromNginx("")
)
