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
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/version"
	"github.com/percona/pmm/vmproxy/proxy"
)

//nolint:lll
type flags struct {
	Debug bool `help:"Enable debug logging"`

	TargetURL          *url.URL `default:"http://127.0.0.1:9090" help:"Target URL where to proxy requests"`
	TargetUsernameFile string   `help:"File with the username to authenticate to the target with, instead of the target URL"`
	TargetPasswordFile string   `help:"File with the password to authenticate to the target with, instead of the target URL"`
	ListenPort      int      `default:"1280" help:"Listen port for proxy"`
	ListenAddress   string   `default:"127.0.0.1" help:"Listen address for proxy"`
	HeaderName      string   `default:"X-Proxy-Filter" help:"Header name to read filter configuration from. The content of the header shall be a base64 encoded JSON array with strings. Each string is a filter. Multiple filters are joined with a logical OR."`
	AdminHeaderName string   `default:"X-Proxy-Admin" help:"Header name pmm-managed sets to mark a request as coming from an admin, which allows the admin-only VictoriaMetrics diagnostics. nginx overwrites it on every route reaching this proxy, so clients cannot set it."`
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

	targetURL, err := withCredentialsFromFiles(opts.TargetURL, opts.TargetUsernameFile, opts.TargetPasswordFile)
	if err != nil {
		return err
	}

	err = proxyFn(proxy.Config{
		HeaderName:      opts.HeaderName,
		AdminHeaderName: opts.AdminHeaderName,
		ListenAddress:   net.JoinHostPort(opts.ListenAddress, strconv.Itoa(opts.ListenPort)),
		TargetURL:       targetURL,
	})

	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	logrus.Info(err)

	return nil
}

// withCredentialsFromFiles returns a copy of target with the credentials that the given files hold,
// or target itself if no file is given. Kept in files, the credentials are neither on the command
// line nor in the configuration that runs vmproxy.
func withCredentialsFromFiles(target *url.URL, usernameFile, passwordFile string) (*url.URL, error) {
	if usernameFile == "" && passwordFile == "" {
		return target, nil
	}
	if target.User != nil {
		return nil, errors.New("the target URL carries credentials, and files with credentials are given too")
	}

	var username, password string
	for _, f := range []struct {
		path  string
		value *string
	}{
		{usernameFile, &username},
		{passwordFile, &password},
	} {
		if f.path == "" {
			continue
		}
		b, err := os.ReadFile(f.path)
		if err != nil {
			return nil, fmt.Errorf("cannot read the credentials of the target: %w", err)
		}
		*f.value = strings.TrimRight(string(b), "\r\n")
	}

	withCredentials := *target
	withCredentials.User = url.UserPassword(username, password)
	return &withCredentials, nil
}
