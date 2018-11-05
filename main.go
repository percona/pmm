// pmm-admin
// Copyright (C) 2018 Percona LLC
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

package main

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"os"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/percona/pmm/api/json/client"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/Percona-Lab/pmm-admin/commands"
)

var (
	// Version is an application version.
	// TODO Set it during the build.
	Version = "2.0.0-dev"
)

func main() {
	app := kingpin.New("pmm-admin", "Version "+Version+".")
	app.HelpFlag.Short('h')
	app.Version(Version)
	pmmServerAddressF := app.Flag("server-url", "PMM Server URL.").Envar("PMM_ADMIN_SERVER_URL").Required().String()
	debugF := app.Flag("debug", "Enable debug output.").Envar("PMM_ADMIN_DEBUG").Bool()
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if *debugF {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("Debug logging enabled.")
	}

	u, err := url.Parse(*pmmServerAddressF)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Debugf("PMM Server URL: %#v.", u)
	if u.Path == "" {
		u.Path = "/"
	}
	if u.Host == "" || u.Scheme == "" {
		logrus.Fatal("Invalid PMM Server URL.")
	}

	// use JSON APIs over HTTP/1.1
	transport := httptransport.New(u.Host, u.Path, []string{u.Scheme})
	transport.SetLogger(logrus.WithField("component", "client"))
	transport.Debug = *debugF
	// disable HTTP/2
	transport.Transport.(*http.Transport).TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	client.Default = client.New(transport, nil)

	cmd := commands.AddMySQLCmd{
		Username: "username",
		Password: "password",
	}
	cmd.Run()
}
