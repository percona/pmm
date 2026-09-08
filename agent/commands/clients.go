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

// Package commands contains CLI commands implementations.
package commands

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"

	"github.com/percona/pmm/agent/config"
	agentlocalClient "github.com/percona/pmm/api/agentlocal/v1/json/client"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	aservice "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	nservice "github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
	managementClient "github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
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

	// disable HTTP/2
	httpTransport := transport.Transport.(*http.Transport) //nolint:forcetypeassert
	httpTransport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)

	agentlocalClient.Default.SetTransport(transport)
}

type statusResult struct {
	ConfigFilepath string
}

// localStatus returns locally running pmm-agent status.
// Error is returned if pmm-agent is not running.
//
// This method is not thread-safe.
func localStatus() (*statusResult, error) {
	res, err := agentlocalClient.Default.AgentLocalService.Status(nil)
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
	_, err := agentlocalClient.Default.AgentLocalService.Reload(nil)
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
		user := u.User.Username()
		password, _ := u.User.Password()
		if user == "service_token" || user == "api_key" {
			transport.DefaultAuthentication = httptransport.BearerToken(password)
		} else {
			transport.DefaultAuthentication = httptransport.BasicAuth(user, password)
		}
	}
	transport.SetLogger(l)
	transport.SetDebug(l.Logger.GetLevel() >= logrus.DebugLevel)

	// set error handlers for nginx responses if pmm-managed is down
	errorConsumer := runtime.ConsumerFunc(func(reader io.Reader, _ any) error {
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
	httpTransport := transport.Transport.(*http.Transport) //nolint:forcetypeassert
	httpTransport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
	if u.Scheme == "https" {
		httpTransport.TLSClientConfig = tlsconfig.Get()
		httpTransport.TLSClientConfig.ServerName = u.Hostname()
		httpTransport.TLSClientConfig.InsecureSkipVerify = insecureTLS
	}

	managementClient.Default.SetTransport(transport)
	inventoryClient.Default.SetTransport(transport)
}

// errAgentNotFound reports that PMM Server has no Agent with the given ID.
var errAgentNotFound = errors.New("agent not found")

// errCredentialsRejected reports that PMM Server did not accept the credentials of the request.
var errCredentialsRejected = errors.New("credentials rejected")

// serverNode describes the Node which PMM Server has an Agent registered on.
type serverNode struct {
	Name    string
	Address string
}

// serverNodeOfAgent returns the Node which PMM Server has the Agent registered on.
// The errors errAgentNotFound and errCredentialsRejected mean that the Node has to be registered again. Any other
// error means that PMM Server could not be asked, so that the caller can tell "the registration is gone"
// apart from "the answer is unknown".
//
// This method is not thread-safe.
func serverNodeOfAgent(agentID string) (serverNode, error) {
	// The constructors bound the requests with the default timeout, so a hung PMM Server cannot stall the setup.
	agent, err := inventoryClient.Default.AgentsService.GetAgent(aservice.NewGetAgentParams().WithAgentID(agentID))
	if err != nil {
		return serverNode{}, lookupError(err)
	}
	if agent.Payload.PMMAgent == nil {
		// The ID belongs to another kind of Agent, so it is not a registration of this pmm-agent.
		return serverNode{}, errAgentNotFound
	}

	node, err := inventoryClient.Default.NodesService.GetNode(nservice.NewGetNodeParams().WithNodeID(agent.Payload.PMMAgent.RunsOnNodeID))
	if err != nil {
		return serverNode{}, lookupError(err)
	}

	return nodeOf(node.Payload)
}

// serverCode returns the gRPC code PMM Server put in the body of a failed inventory request, or codes.OK
// when the answer carries none. It is the only thing which identifies the answer as PMM Server's own.
func serverCode(err error) codes.Code {
	var code int32
	switch e := err.(type) { //nolint:errorlint
	case *aservice.GetAgentDefault:
		if e.Payload != nil {
			code = e.Payload.Code
		}
	case *nservice.GetNodeDefault:
		if e.Payload != nil {
			code = e.Payload.Code
		}
	}
	if code < 0 {
		return codes.OK
	}

	return codes.Code(code)
}

// lookupError maps a failed inventory lookup to what it says about the registration. Only PMM Server's
// own answer says anything: a proxy whose path rules predate this call answers 404 just the same, and
// PMM Server maps a failure of its own to 401 exactly as it does a credential it rejected. The gRPC code
// in the body is what tells those apart, so an answer carrying none is no answer at all.
func lookupError(err error) error {
	switch serverCode(err) {
	// An ID which PMM Server does not know, or rejects as invalid, cannot be registered there either.
	case codes.NotFound, codes.InvalidArgument:
		return errAgentNotFound
	// The credentials the Agent runs with are gone, so registering either succeeds with the ones given to
	// setup or reports a credentials problem with an actionable message. codes.PermissionDenied is
	// deliberately not here: a service account below the admin role still holds valid credentials, it
	// just cannot read the inventory, and registering the Node again over that would only add a second.
	case codes.Unauthenticated:
		return errCredentialsRejected
	default:
		return err
	}
}

// nodeOf returns the Node in the GetNode response. A Node type this pmm-agent does not know is not one
// it can compare a name with, and a newer PMM Server may well answer with one.
func nodeOf(node *nservice.GetNodeOKBody) (serverNode, error) {
	switch {
	case node.Generic != nil:
		return serverNode{Name: node.Generic.NodeName, Address: node.Generic.Address}, nil
	case node.Container != nil:
		return serverNode{Name: node.Container.NodeName, Address: node.Container.Address}, nil
	case node.Remote != nil:
		return serverNode{Name: node.Remote.NodeName, Address: node.Remote.Address}, nil
	case node.RemoteRDS != nil:
		return serverNode{Name: node.RemoteRDS.NodeName, Address: node.RemoteRDS.Address}, nil
	case node.RemoteAzureDatabase != nil:
		return serverNode{Name: node.RemoteAzureDatabase.NodeName, Address: node.RemoteAzureDatabase.Address}, nil
	default:
		return serverNode{}, errors.New("PMM Server answered with a Node type this pmm-agent does not know")
	}
}

// ParseKeyValuePair parses --custom-labels flag value.
//
// Note that quotes around value are parsed and removed by shell before this function is called.
// For example, the value of [[--custom-labels='region=us-east1, mylabel=mylab-22']] will be received by this function
// as [[region=us-east1, mylabel=mylab-22]].
func ParseKeyValuePair(labels string) (map[string]string, error) {
	result := make(map[string]string)
	parts := strings.SplitSeq(labels, ",")
	for part := range parts {
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
func serverRegister(cfgSetup *config.Setup) (agentID, token string, _ error) { //nolint:nonamedreturns
	nodeTypes := map[string]string{
		"generic":   mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE,
		"container": mservice.RegisterNodeBodyNodeTypeNODETYPECONTAINERNODE,
	}

	var disableCollectors []string
	for v := range strings.SplitSeq(cfgSetup.DisableCollectors, ",") {
		disableCollector := strings.TrimSpace(v)
		if disableCollector != "" {
			disableCollectors = append(disableCollectors, disableCollector)
		}
	}

	customLabels, err := ParseKeyValuePair(cfgSetup.CustomLabels)
	if err != nil {
		return "", "", err
	}

	res, err := managementClient.Default.ManagementService.RegisterNode(&mservice.RegisterNodeParams{
		Body: mservice.RegisterNodeBody{
			NodeType:      new(nodeTypes[cfgSetup.NodeType]),
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
			MetricsMode:       new(strings.ToUpper("METRICS_MODE_" + cfgSetup.MetricsMode)),
			DisableCollectors: disableCollectors,
			ExposeExporter:    cfgSetup.ExposeExporter,
		},
		Context: context.Background(),
	})
	if err != nil {
		return "", "", err
	}
	// TODO: Investigate what can lead to PMMAgent being nil in the response
	if res.Payload == nil || res.Payload.PMMAgent == nil {
		return "", "", errors.New("unexpected empty response from PMM Server (missing pmm_agent)")
	}
	return res.Payload.PMMAgent.AgentID, res.Payload.Token, nil
}

// check interfaces.
var (
	_ error          = nginxError("")
	_ fmt.GoStringer = nginxError("")
)
