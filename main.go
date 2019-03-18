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
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/percona/pmm/api/inventory/json/client"
	"github.com/percona/pmm/version"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/percona/pmm-admin/commands"
)

func main() {
	app := kingpin.New("pmm-agent", fmt.Sprintf("Version %s.", version.Version))
	app.HelpFlag.Short('h')
	app.Version(version.FullInfo())
	pmmServerAddressF := app.Flag("server-url", "PMM Server URL.").Required().String()
	debugF := app.Flag("debug", "Enable debug logging.").Bool()
	traceF := app.Flag("trace", "Enable trace logging (implies debug).").Bool()
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if *debugF {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if *traceF {
		logrus.SetLevel(logrus.TraceLevel)
		logrus.SetReportCaller(true)
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

	ctx, cancel := context.WithCancel(context.Background())

	// handle termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		signal.Stop(signals)
		logrus.Warnf("Got %s, shutting down...", unix.SignalName(s.(unix.Signal)))
		cancel()
	}()

	// use JSON APIs over HTTP/1.1
	transport := httptransport.New(u.Host, u.Path, []string{u.Scheme})
	transport.SetLogger(logrus.WithField("component", "client"))
	transport.Context = ctx
	transport.Debug = *debugF || *traceF
	// disable HTTP/2
	transport.Transport.(*http.Transport).TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	client.Default = client.New(transport, nil)

	cmd := commands.AddMySQLCmd{
		Username: "username",
		Password: "password",
	}
	cmd.Run()
}
