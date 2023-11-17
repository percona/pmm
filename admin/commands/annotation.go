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
	"strings"

	"github.com/pkg/errors"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/helpers"
	"github.com/percona/pmm/api/inventorypb/v1/json/client"
	nodes "github.com/percona/pmm/api/inventorypb/v1/json/client/nodes_service"
	services "github.com/percona/pmm/api/inventorypb/v1/json/client/services_service"
	managementClient "github.com/percona/pmm/api/managementpb/json/client"
	annotation "github.com/percona/pmm/api/managementpb/json/client/annotation_service"
)

var annotationResultT = ParseTemplate(`
Annotation added.
`)

// annotationResult is a result of annotation command.
type annotationResult struct{}

// Result is a command run result.
func (res *annotationResult) Result() {}

// String stringifies command result.
func (res *annotationResult) String() string {
	return RenderTemplate(annotationResultT, res)
}

// AnnotationCommand is used by Kong for CLI flags and commands.
type AnnotationCommand struct {
	Text        string   `arg:"" help:"Text of annotation"`
	Tags        []string `help:"Tags to filter annotations. Multiple tags are separated by a comma"`
	Node        bool     `help:"Annotate current node"`
	NodeName    string   `help:"Name of node to annotate"`
	Service     bool     `help:"Annotate services of current node"`
	ServiceName string   `help:"Name of service to annotate"`
}

func (cmd *AnnotationCommand) nodeName() (string, error) {
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

	return helpers.GetNodeName(node)
}

func (cmd *AnnotationCommand) getCurrentNode() (*nodes.GetNodeOKBody, error) {
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

	result, err := client.Default.NodesService.GetNode(params)
	if err != nil {
		return nil, errors.Wrap(err, "default get node")
	}

	return result.GetPayload(), nil
}

func (cmd *AnnotationCommand) serviceNames() ([]string, error) {
	switch {
	case cmd.ServiceName != "":
		return []string{cmd.ServiceName}, nil
	case cmd.Service:
		return cmd.getCurrentNodeAllServices()
	default:
		return []string{}, nil
	}
}

func (cmd *AnnotationCommand) getCurrentNodeAllServices() ([]string, error) {
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

	result, err := client.Default.ServicesService.ListServices(params)
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

// RunCmd runs annotation command.
func (cmd *AnnotationCommand) RunCmd() (Result, error) {
	for i := range cmd.Tags {
		cmd.Tags[i] = strings.TrimSpace(cmd.Tags[i])
	}

	nodeName, err := cmd.nodeName()
	if err != nil {
		return nil, err
	}

	serviceNames, err := cmd.serviceNames()
	if err != nil {
		return nil, err
	}

	_, err = managementClient.Default.AnnotationService.AddAnnotation(&annotation.AddAnnotationParams{
		Body: annotation.AddAnnotationBody{
			Text:         cmd.Text,
			Tags:         cmd.Tags,
			NodeName:     nodeName,
			ServiceNames: serviceNames,
		},
		Context: Ctx,
	})
	if err != nil {
		return nil, err
	}

	return &annotationResult{}, nil
}
