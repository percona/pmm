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
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

var (
	allNodeTypes = map[string]string{
		"generic":   mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE,
		"container": mservice.RegisterNodeBodyNodeTypeNODETYPECONTAINERNODE,
		"remote":    mservice.RegisterNodeBodyNodeTypeNODETYPEREMOTENODE,
	}

	allServiceTypes = map[string]string{
		"mysql":      mservice.RemoveServiceBodyServiceTypeSERVICETYPEMYSQLSERVICE,
		"mongodb":    mservice.RemoveServiceBodyServiceTypeSERVICETYPEMONGODBSERVICE,
		"postgresql": mservice.RemoveServiceBodyServiceTypeSERVICETYPEPOSTGRESQLSERVICE,
		"proxysql":   mservice.RemoveServiceBodyServiceTypeSERVICETYPEPROXYSQLSERVICE,
		"haproxy":    mservice.RemoveServiceBodyServiceTypeSERVICETYPEHAPROXYSERVICE,
		"external":   mservice.RemoveServiceBodyServiceTypeSERVICETYPEEXTERNALSERVICE,
	}

	// AllServiceTypesKeys lists all possible service types.
	AllServiceTypesKeys = []string{"mysql", "mongodb", "postgresql", "proxysql", "haproxy", "external"}

	// MetricsModes lists all possible metrics modes.
	MetricsModes = []string{"auto", "push", "pull"}
)
