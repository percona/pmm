// Copyright (C) 2023 Percona LLC
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

// Package main is the main package for vmproxy application.
package main

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/alecthomas/kong"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/version"
	"github.com/percona/pmm/vmproxy/proxy"
)

type flags struct {
	Debug bool `help:"Enable debug logging"`

	TargetURL     *url.URL `default:"http://127.0.0.1:9090" help:"Target URL where to proxy requests"`
	ListenPort    int      `default:"1280" help:"Listen port for proxy"`
	ListenAddress string   `default:"127.0.0.1" help:"Listen address for proxy"`
	HeaderName    string   `default:"X-Proxy-Filter" help:"Header name to read filter configuration from. The content of the header shall be a base64 encoded JSON array with strings. Each string is a filter. Multiple filters are joined with a logical OR."` //nolint:lll
	HostHeader    string   `default:"" help:"Optional Host header value to set in the request, overrides existing"`
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

	if err := runProxy(opts, proxy.RunProxy); err != nil {
		logrus.Fatal(err)
	}
}

func runProxy(opts flags, proxyFn func(cfg proxy.Config) error) error {
	if opts.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	err := proxyFn(proxy.Config{
		HeaderName:    opts.HeaderName,
		ListenAddress: net.JoinHostPort(opts.ListenAddress, strconv.Itoa(opts.ListenPort)),
		TargetURL:     opts.TargetURL,
		HostHeader:    opts.HostHeader,
	})

	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	logrus.Info(err)

	return nil
}
