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

// Package main is the main package for vmproxy application.
package main

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/alecthomas/kong"

	"github.com/percona/pmm/version"
	"github.com/percona/pmm/vmproxy/pkg/proxy"
)

type flags struct {
	TargetURL     *url.URL `default:"http://127.0.0.1:9090" help:"Target URL where to proxy requests"`
	ListenPort    int      `default:"1280" help:"Listen port for proxy"`
	ListenAddress string   `default:"127.0.0.1" help:"Listen address for proxy"`
	HeaderName    string   `default:"X-Proxy-Filter" help:"Header name to read filters configuration from. The content of the header shall be a base64 encoded JSON array with strings. Each string is a filter. Multiple filters are join with a logical OR."` //nolint:lll
}

func main() {
	var opts flags
	kong.Parse(
		&opts,
		kong.Name("vmproxy"),
		kong.Description(fmt.Sprintf("Version %s", version.Version)),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:             true,
			NoExpandSubcommands: true,
		}),
	)

	proxy.StartProxy(proxy.Config{
		HeaderName:    opts.HeaderName,
		ListenAddress: net.JoinHostPort(opts.ListenAddress, strconv.Itoa(opts.ListenPort)),
		TargetURL:     opts.TargetURL,
	})
}
