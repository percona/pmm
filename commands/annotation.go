// pmm-admin
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
	"strings"

	"github.com/percona/pmm/api/inventorypb/json/client"
	"github.com/percona/pmm/api/inventorypb/json/client/nodes"
	"github.com/percona/pmm/api/inventorypb/json/client/services"
	managementClient "github.com/percona/pmm/api/managementpb/json/client"
	"github.com/percona/pmm/api/managementpb/json/client/annotation"
	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm-admin/agentlocal"
)

var annotationResultT = ParseTemplate(`
Annotation added.
`)

var errNoNode = errors.New("no node available")

// annotationResult is a result of annotation command.
type annotationResult struct{}

// Result is a command run result.
func (res *annotationResult) Result() {}

// String stringifies command result.
func (res *annotationResult) String() string {
	return RenderTemplate(annotationResultT, res)
}

type annotationCommand struct {
	Text        string
	Tags        string
	Node        bool
	NodeName    string
	Service     bool
	ServiceName string
}

func (cmd *annotationCommand) nodeName() (string, error) {
	if cmd.NodeName != "" {
		return cmd.NodeName, nil
	}

	if !cmd.Node {
		return "", nil
	}

	node, err := cmd.getCurrentNode()
	if err != nil {
		return "", err
	}

	switch {
	case node.Generic != nil:
		return node.Generic.NodeName, nil
	case node.Container != nil:
		return node.Container.NodeName, nil
	case node.Remote != nil:
		return node.Remote.NodeName, nil
	case node.RemoteRDS != nil:
		return node.RemoteRDS.NodeName, nil
	default:
		return "", errors.Wrap(errNoNode, "unknown node type")
	}
}

func (cmd *annotationCommand) getCurrentNode() (*nodes.GetNodeOKBody, error) {
	status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
	if err != nil {
		return nil, err
	}

	params := &nodes.GetNodeParams{
		Body: nodes.GetNodeBody{
			NodeID: status.NodeID,
		},
		Context: Ctx,
	}

	result, err := client.Default.Nodes.GetNode(params)
	if err != nil {
		return nil, errors.Wrap(err, "default get node")
	}

	return result.GetPayload(), nil
}

func (cmd *annotationCommand) serviceNames() ([]string, error) {
	switch {
	case cmd.ServiceName != "":
		return []string{cmd.ServiceName}, nil
	case cmd.Service:
		return cmd.getCurrentNodeAllServices()
	default:
		return []string{}, nil
	}
}

func (cmd *annotationCommand) getCurrentNodeAllServices() ([]string, error) {
	status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
	if err != nil {
		return nil, err
	}

	params := &services.ListServicesParams{
		Body: services.ListServicesBody{
			NodeID: status.NodeID,
		},
		Context: Ctx,
	}

	result, err := client.Default.Services.ListServices(params)
	if err != nil {
		return nil, err
	}

	servicesNameList := []string{}
	for _, s := range result.Payload.Mysql {
		servicesNameList = append(servicesNameList, s.ServiceName)
	}
	for _, s := range result.Payload.Mongodb {
		servicesNameList = append(servicesNameList, s.ServiceName)
	}
	for _, s := range result.Payload.Postgresql {
		servicesNameList = append(servicesNameList, s.ServiceName)
	}
	for _, s := range result.Payload.Proxysql {
		servicesNameList = append(servicesNameList, s.ServiceName)
	}
	for _, s := range result.Payload.External {
		servicesNameList = append(servicesNameList, s.ServiceName)
	}

	return servicesNameList, nil
}

// Run runs annotation command.
func (cmd *annotationCommand) Run() (Result, error) {
	tags := strings.Split(cmd.Tags, ",")
	for i := range tags {
		tags[i] = strings.TrimSpace(tags[i])
	}

	nodeName, err := cmd.nodeName()
	if err != nil {
		return nil, err
	}

	serviceNames, err := cmd.serviceNames()
	if err != nil {
		return nil, err
	}

	_, err = managementClient.Default.Annotation.AddAnnotation(&annotation.AddAnnotationParams{
		Body: annotation.AddAnnotationBody{
			Text:         cmd.Text,
			Tags:         tags,
			NodeName:     nodeName,
			ServiceNames: serviceNames,
		},
		Context: Ctx,
	})
	if err != nil {
		return nil, err
	}

	return new(annotationResult), nil
}

// register command
var (
	Annotation  = new(annotationCommand)
	AnnotationC = kingpin.Command("annotate", "Add an annotation to Grafana charts")
)

func init() {
	AnnotationC.Arg("text", "Text of annotation").Required().StringVar(&Annotation.Text)
	AnnotationC.Flag("tags", "Tags to filter annotations. Multiple tags are separated by a comma").StringVar(&Annotation.Tags)
	AnnotationC.Flag("node", "Annotate current node").BoolVar(&Annotation.Node)
	AnnotationC.Flag("node-name", "Name of node to annotate").StringVar(&Annotation.NodeName)
	AnnotationC.Flag("service", "Annotate services of current node").BoolVar(&Annotation.Service)
	AnnotationC.Flag("service-name", "Name of service to annotate").StringVar(&Annotation.ServiceName)
}
