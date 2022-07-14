// Copyright 2019 Percona LLC
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
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/agent/config"
	agentlocalpb "github.com/percona/pmm/api/agentlocalpb/json/client"
	managementpb "github.com/percona/pmm/api/managementpb/json/client"
	"github.com/percona/pmm/api/managementpb/json/client/node"
	"github.com/percona/pmm/utils/tlsconfig"
)

var customLabelRE = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)=([^='", ]+)$`)

// setLocalTransport configures transport for accessing local pmm-agent API.
//
// This method is not thread-safe.
func setLocalTransport(host string, port uint16, l *logrus.Entry) {
	// use JSON APIs over HTTP/1.1
	address := net.JoinHostPort(host, strconv.Itoa(int(port)))
	transport := httptransport.New(address, "/", []string{"http"})
	transport.SetLogger(l)
	transport.SetDebug(l.Logger.GetLevel() >= logrus.DebugLevel)
	transport.Context = context.Background()

	// disable HTTP/2
	httpTransport := transport.Transport.(*http.Transport)
	httpTransport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)

	agentlocalpb.Default.SetTransport(transport)
}

type statusResult struct {
	ConfigFilepath string
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
		ConfigFilepath: res.Payload.ConfigFilepath,
	}, nil
}

// localReload reloads locally running pmm-agent.
//
// This method is not thread-safe.
func localReload() error {
	_, err := agentlocalpb.Default.AgentLocal.Reload(nil)
	return err
}

type nginxError string

func (e nginxError) Error() string {
	return "response from nginx: " + string(e)
}

func (e nginxError) GoString() string {
	return fmt.Sprintf("nginxError(%q)", string(e))
}

// setServerTransport configures transport for accessing PMM Server API.
//
// This method is not thread-safe.
func setServerTransport(u *url.URL, insecureTLS bool, l *logrus.Entry) {
	// use JSON APIs over HTTP/1.1
	transport := httptransport.New(u.Host, u.Path, []string{u.Scheme})
	if u.User != nil {
		password, _ := u.User.Password()
		transport.DefaultAuthentication = httptransport.BasicAuth(u.User.Username(), password)
	}
	transport.SetLogger(l)
	transport.SetDebug(l.Logger.GetLevel() >= logrus.DebugLevel)
	transport.Context = context.Background()

	// set error handlers for nginx responses if pmm-managed is down
	errorConsumer := runtime.ConsumerFunc(func(reader io.Reader, data interface{}) error {
		b, _ := io.ReadAll(reader)
		return nginxError(string(b))
	})
	transport.Consumers = map[string]runtime.Consumer{
		runtime.JSONMime:    runtime.JSONConsumer(),
		runtime.HTMLMime:    errorConsumer,
		runtime.TextMime:    errorConsumer,
		runtime.DefaultMime: errorConsumer,
	}

	// disable HTTP/2, set TLS config
	httpTransport := transport.Transport.(*http.Transport)
	httpTransport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
	if u.Scheme == "https" {
		httpTransport.TLSClientConfig = tlsconfig.Get()
		httpTransport.TLSClientConfig.ServerName = u.Hostname()
		httpTransport.TLSClientConfig.InsecureSkipVerify = insecureTLS
	}

	managementpb.Default.SetTransport(transport)
}

// ParseCustomLabels parses --custom-labels flag value.
//
// Note that quotes around value are parsed and removed by shell before this function is called.
// E.g. the value of [[--custom-labels='region=us-east1, mylabel=mylab-22']] will be received by this function
// as [[region=us-east1, mylabel=mylab-22]].
func ParseCustomLabels(labels string) (map[string]string, error) {
	result := make(map[string]string)
	parts := strings.Split(labels, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		submatches := customLabelRE.FindStringSubmatch(part)
		if submatches == nil {
			return nil, errors.New("wrong custom label format")
		}
		result[submatches[1]] = submatches[2]
	}
	return result, nil
}

// serverRegister registers Node on PMM Server.
//
// This method is not thread-safe.
func serverRegister(cfgSetup *config.Setup) (string, error) {
	nodeTypes := map[string]string{
		"generic":   node.RegisterNodeBodyNodeTypeGENERICNODE,
		"container": node.RegisterNodeBodyNodeTypeCONTAINERNODE,
	}

	var disableCollectors []string
	for _, v := range strings.Split(cfgSetup.DisableCollectors, ",") {
		disableCollector := strings.TrimSpace(v)
		if disableCollector != "" {
			disableCollectors = append(disableCollectors, disableCollector)
		}
	}

	customLabels, err := ParseCustomLabels(cfgSetup.CustomLabels)
	if err != nil {
		return "", err
	}

	res, err := managementpb.Default.Node.RegisterNode(&node.RegisterNodeParams{
		Body: node.RegisterNodeBody{
			NodeType:      pointer.ToString(nodeTypes[cfgSetup.NodeType]),
			NodeName:      cfgSetup.NodeName,
			MachineID:     cfgSetup.MachineID,
			Distro:        cfgSetup.Distro,
			ContainerID:   cfgSetup.ContainerID,
			ContainerName: cfgSetup.ContainerName,
			NodeModel:     cfgSetup.NodeModel,
			Region:        cfgSetup.Region,
			Az:            cfgSetup.Az,
			Address:       cfgSetup.Address,
			CustomLabels:  customLabels,
			AgentPassword: cfgSetup.AgentPassword,

			Reregister:        cfgSetup.Force,
			MetricsMode:       pointer.ToString(strings.ToUpper(cfgSetup.MetricsMode)),
			DisableCollectors: disableCollectors,
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
	_ error          = nginxError("")
	_ fmt.GoStringer = nginxError("")
)
