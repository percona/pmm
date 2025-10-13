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

// Package flags holds global flags.
package flags

import (
	"fmt"
	"net/url"
	"os"

	"github.com/percona/pmm/version"
)

var isJSON = false

// GlobalFlags stores flags global to all commands.
type GlobalFlags struct {
	ServerURL               *url.URL    `placeholder:"SERVER-URL" help:"PMM Server URL in https://username:password@pmm-server-host/ format"`
	SkipTLSCertificateCheck bool        `name:"server-insecure-tls" help:"Skip PMM Server TLS certificate validation"`
	EnableDebug             bool        `name:"debug" help:"Enable debug logging"`
	EnableTrace             bool        `name:"trace" help:"Enable trace logging (implies debug)"`
	PMMAgentListenPort      uint32      `default:"${defaultListenPort}" help:"Set listen port of pmm-agent"`
	JSON                    jsonFlag    `help:"Enable JSON output"`
	Version                 versionFlag `short:"v" help:"Show application version"`
}

type versionFlag bool

// BeforeApply is run before the version flag is applied.
func (v versionFlag) BeforeApply() error {
	// For backwards compatibility we scan for "--json" flag.
	// Kong parses the flags from left to right which breaks compatibility
	// if the --json flag is after --version flag.
	if !isJSON {
		for _, arg := range os.Args[1:] {
			if arg == "--json" {
				isJSON = true
			}
		}
	}

	if isJSON {
		fmt.Println(version.FullInfoJSON()) //nolint:forbidigo
	} else {
		fmt.Println(version.FullInfo()) //nolint:forbidigo
	}
	os.Exit(0)

	return nil
}

type jsonFlag bool

// BeforeApply is run before the json flag is applied.
func (v jsonFlag) BeforeApply() error {
	// See comment in versionFlag.BeforeApply() for context
	isJSON = true
	return nil
}
