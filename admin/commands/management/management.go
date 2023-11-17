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

// Package management provides management commands.
package management

import (
	node "github.com/percona/pmm/api/managementpb/v1/json/client/node_service"
	"github.com/percona/pmm/api/managementpb/v1/json/client/service"
)

var (
	allNodeTypes = map[string]string{
		"generic":   node.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE,
		"container": node.RegisterNodeBodyNodeTypeNODETYPECONTAINERNODE,
		"remote":    node.RegisterNodeBodyNodeTypeNODETYPEREMOTENODE,
	}

	allServiceTypes = map[string]string{
		"mysql":      service.RemoveServiceBodyServiceTypeSERVICETYPEMYSQLSERVICE,
		"mongodb":    service.RemoveServiceBodyServiceTypeSERVICETYPEMONGODBSERVICE,
		"postgresql": service.RemoveServiceBodyServiceTypeSERVICETYPEPOSTGRESQLSERVICE,
		"proxysql":   service.RemoveServiceBodyServiceTypeSERVICETYPEPROXYSQLSERVICE,
		"haproxy":    service.RemoveServiceBodyServiceTypeSERVICETYPEHAPROXYSERVICE,
		"external":   service.RemoveServiceBodyServiceTypeSERVICETYPEEXTERNALSERVICE,
	}

	// AllServiceTypesKeys lists all possible service types.
	AllServiceTypesKeys = []string{"mysql", "mongodb", "postgresql", "proxysql", "haproxy", "external"}

	// MetricsModes lists all possible metrics modes.
	MetricsModes = []string{"auto", "push", "pull"}
)
