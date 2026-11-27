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
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/alecthomas/kong"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/utils/secretfile"
	"github.com/percona/pmm/version"
	"github.com/percona/pmm/vmproxy/proxy"
)

//nolint:lll
type flags struct {
	Debug bool `help:"Enable debug logging"`

	TargetURL          *url.URL `default:"http://127.0.0.1:9090" help:"Target URL where to proxy requests"`
	TargetUsername     string   `help:"Username for the target with --target-password-file, in place of the one in the target URL"`
	TargetPasswordFile string   `help:"File with the password for the target, in place of the one in the target URL"`
	ListenPort         int      `default:"1280" help:"Listen port for proxy"`
	ListenAddress      string   `default:"127.0.0.1" help:"Listen address for proxy"`
	HeaderName         string   `default:"X-Proxy-Filter" help:"Header name to read filter configuration from. The content of the header shall be a base64 encoded JSON array with strings. Each string is a filter. Multiple filters are joined with a logical OR."`
	AdminHeaderName    string   `default:"X-Proxy-Admin" help:"Header name pmm-managed sets to mark a request as coming from an admin, which allows the admin-only VictoriaMetrics diagnostics. nginx overwrites it on every route reaching this proxy, so clients cannot set it."`
}

func main() {
	var opts flags
	kong.Parse(
		&opts,
		kong.Name("vmproxy"),
		kong.Description("Version "+version.Version),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:             true,
			NoExpandSubcommands: true,
		}),
	)

	err := runProxy(opts, proxy.RunProxy)
	if err != nil {
		logrus.Fatal(err)
	}
}

func runProxy(opts flags, proxyFn func(cfg proxy.Config) error) error {
	if opts.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	target, err := targetWithCredentials(opts.TargetURL, opts.TargetUsername, opts.TargetPasswordFile)
	if err != nil {
		return err
	}

	err = proxyFn(proxy.Config{
		HeaderName:      opts.HeaderName,
		AdminHeaderName: opts.AdminHeaderName,
		ListenAddress:   net.JoinHostPort(opts.ListenAddress, strconv.Itoa(opts.ListenPort)),
		TargetURL:       target,
	})

	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	logrus.Info(err)

	return nil
}

// targetWithCredentials returns the target URL with the username and the password from passwordFile, when that is set.
func targetWithCredentials(target *url.URL, username, passwordFile string) (*url.URL, error) {
	if passwordFile == "" {
		return target, nil
	}
	password, err := secretfile.Read(passwordFile)
	if err != nil {
		return nil, fmt.Errorf("cannot read the target password: %w", err)
	}
	if username == "" {
		username = target.User.Username()
	}
	u := *target
	u.User = url.UserPassword(username, password)
	return &u, nil
}
