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
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/helpers"
	"github.com/percona/pmm/admin/pkg/flags"
	agents_info "github.com/percona/pmm/api/agentlocal/v1/json/client/agent_local_service"
	"github.com/percona/pmm/api/inventory/v1/types"
	"github.com/percona/pmm/api/server/v1/json/client"
	server "github.com/percona/pmm/api/server/v1/json/client/server_service"
	"github.com/percona/pmm/version"
)

var summaryResultT = ParseTemplate(`
{{ .Filename }} created.
`)

type summaryResult struct {
	Filename string `json:"filename"`
}

func (res *summaryResult) Result() {}

func (res *summaryResult) String() string {
	return RenderTemplate(summaryResultT, res)
}

// addData adds data from io.Reader to zip file with given name and time.
func addData(zipW *zip.Writer, name string, modTime time.Time, r io.Reader) {
	w, err := zipW.CreateHeader(&zip.FileHeader{
		Name:     name,
		Method:   zip.Deflate,
		Modified: modTime,
	})
	if err == nil {
		_, err = io.Copy(w, r)
	}
	if err != nil {
		logrus.Errorf("%s", err)
	}
}

// addFile adds data from fileName to zip file with given name.
func addFile(zipW *zip.Writer, name string, fileName string) {
	// do not read the whole file at once - it can be very big

	var r io.ReadCloser
	r, err := os.Open(fileName) //nolint:gosec
	if err != nil {
		// use error instead of file data
		logrus.Debugf("%s", err)
		r = io.NopCloser(bytes.NewReader([]byte(err.Error() + "\n")))
	}
	defer r.Close() //nolint:gosec,errcheck,nolintlint

	modTime := time.Now()
	if fi, _ := os.Stat(fileName); fi != nil {
		modTime = fi.ModTime()
	}

	addData(zipW, name, modTime, r)
}

// addClientCommand adds cmd.Run() results to zip file with given name.
func addClientCommand(zipW *zip.Writer, name string, cmd Command) {
	var b []byte
	res, err := cmd.RunCmd()
	if res != nil {
		b = append([]byte(res.String()), "\n\n"...)
	}
	if err != nil {
		b = append(b, err.Error()...)
	}

	addData(zipW, name, time.Now(), bytes.NewReader(b))
}

// addClientData adds all PMM Client data to zip file.
func addClientData(ctx context.Context, zipW *zip.Writer) {
	status, err := agentlocal.GetRawStatus(ctx, agentlocal.RequestNetworkInfo)
	if err != nil {
		logrus.Errorf("%s", err)
		return
	}

	addVMAgentTargets(ctx, zipW, status.AgentsInfo)

	// Redact user credentials if they exist
	if u, err := url.Parse(status.ServerInfo.URL); err == nil {
		if u.User.String() != "" {
			u.User = url.UserPassword("xxxxx", "xxxxx")
			status.ServerInfo.URL = u.String()
		}
	} else {
		logrus.Warnf("%s", err)
	}

	b, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		logrus.Debugf("%s", err)
		b = []byte(err.Error())
	}
	b = append(b, '\n')

	now := time.Now()
	addData(zipW, "client/status.json", now, bytes.NewReader(b))

	// FIXME get it via pmm-agent's API - it is _not_ a good idea to use exec there
	// golangci-lint should continue complain about it until it is fixed
	b, err = exec.Command("pmm-agent", "--version").CombinedOutput()
	if err != nil {
		logrus.Debugf("%s", err)
		b = []byte(err.Error())
	}
	addData(zipW, "client/pmm-agent-version.txt", now, bytes.NewReader(b))

	addData(zipW, "client/pmm-admin-version.txt", now, bytes.NewReader([]byte(version.FullInfo())))

	host := net.JoinHostPort(agentlocal.Localhost, fmt.Sprintf("%d", agentlocal.DefaultPMMAgentListenPort))
	err = downloadFile(ctx, zipW, fmt.Sprintf("http://%s/logs.zip", host), "client/pmm-agent")
	if err != nil {
		logrus.Warnf("%s", err)
	}

	if status.ConfigFilepath != "" {
		addFile(zipW, "client/pmm-agent-config.yaml", status.ConfigFilepath)
	}

	addClientCommand(zipW, "client/list.txt", &ListCommand{NodeID: status.RunsOnNodeID})
}

// addServerData adds logs.zip from PMM Server to zip file.
func addServerData(ctx context.Context, zipW *zip.Writer, usePprof bool) {
	var buf bytes.Buffer
	_, err := client.Default.ServerService.Logs(&server.LogsParams{Context: ctx, Pprof: &usePprof}, &buf)
	if err != nil {
		logrus.Errorf("%s", err)
		return
	}

	bufR := bytes.NewReader(buf.Bytes())
	zipR, err := zip.NewReader(bufR, bufR.Size())
	if err != nil {
		logrus.Errorf("%s", err)
		return
	}

	for _, rf := range zipR.File {
		rc, err := rf.Open()
		if err != nil {
			logrus.Errorf("%s", err)
			continue
		}

		addData(zipW, path.Join("server", rf.Name), rf.Modified, rc) //nolint:gosec

		rc.Close() //nolint:errcheck
	}
}

func addVMAgentTargets(ctx context.Context, zipW *zip.Writer, agentsInfo []*agents_info.StatusOKBodyAgentsInfoItems0) {
	now := time.Now()

	for _, agent := range agentsInfo {
		if pointer.GetString(agent.AgentType) == types.AgentTypeVMAgent {
			host := net.JoinHostPort(agentlocal.Localhost, fmt.Sprintf("%d", agent.ListenPort))
			b, err := getURL(ctx, fmt.Sprintf("http://%s/api/v1/targets", host))
			if err != nil {
				logrus.Debugf("%s", err)
				b = []byte(err.Error())
			}

			addData(zipW, "client/vmagent-targets.json", now, bytes.NewReader(b))
			var html []byte
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s/targets", host), nil)
			if err != nil {
				logrus.Debugf("%s", err)
				addData(zipW, "client/vmagent-targets.html", now, bytes.NewReader([]byte(err.Error())))
				return
			}
			req.Header.Set("accept", "text/html")
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				logrus.Debugf("%s", err)
				addData(zipW, "client/vmagent-targets.html", now, bytes.NewReader([]byte(err.Error())))
				return
			}
			defer res.Body.Close() //nolint:gosec,errcheck,nolintlint
			html, err = io.ReadAll(res.Body)
			if err != nil {
				logrus.Debugf("%s", err)
				html = []byte(err.Error())
			}
			addData(zipW, "client/vmagent-targets.html", now, bytes.NewReader(html))
			return
		}
	}
}

// getURL returns `GET url` response body.
func getURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("status code: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read response body")
	}
	return b, nil
}

// downloadFile download file and includes into zip file.
func downloadFile(ctx context.Context, zipW *zip.Writer, url, fileName string) error {
	b, err := getURL(ctx, url)
	if err != nil {
		return errors.WithStack(err)
	}

	responseReader := bytes.NewReader(b)

	zipReader, err := zip.NewReader(responseReader, responseReader.Size())
	if err != nil {
		return errors.Wrap(err, "cannot create ZipLogs reader")
	}

	for _, rf := range zipReader.File {
		rc, err := rf.Open()
		if err != nil {
			logrus.Errorf("%s", err)
			continue
		}
		addData(zipW, path.Join(fileName, rf.Name), rf.Modified, rc) //nolint:gosec

		rc.Close() //nolint:errcheck
	}
	return nil
}

type pprofData struct {
	name string
	data []byte
}

// addPprofData adds pprof data to zip file.
func addPprofData(ctx context.Context, zipW *zip.Writer, skipServer bool, globals *flags.GlobalFlags) {
	profiles := []struct {
		name    string
		urlPath string
	}{
		{
			"profile.pb.gz",
			"/profile?seconds=60",
		}, {
			"heap.pb.gz",
			"/heap?gc=1",
		}, {
			"trace.out",
			"/trace?seconds=10",
		},
	}

	host := net.JoinHostPort(agentlocal.Localhost, fmt.Sprintf("%d", globals.PMMAgentListenPort))
	sources := map[string]string{
		"client/pprof/pmm-agent": fmt.Sprintf("http://%s/debug/pprof", host),
	}

	isRunOnPmmServer, _ := helpers.IsOnPmmServer() //nolint:contextcheck

	if !skipServer && isRunOnPmmServer {
		sources["server/pprof/qan-api2"] = fmt.Sprintf("http://%s/debug/pprof", net.JoinHostPort(agentlocal.Localhost, "9933"))
	}

	for _, p := range profiles {
		// fetch the same profile from different sources in parallel

		var wg sync.WaitGroup
		ch := make(chan pprofData, len(sources))

		for dir, urlPrefix := range sources {
			wg.Add(1)

			go func(url, name string) {
				defer wg.Done()

				logrus.Infof("Getting %s ...", url)
				data, err := getURL(ctx, url)
				if err != nil {
					logrus.Warnf("%s", err)
					return
				}

				ch <- pprofData{
					name: name,
					data: data,
				}
			}(urlPrefix+p.urlPath, dir+"/"+p.name)
		}

		wg.Wait()
		close(ch)

		for res := range ch {
			addData(zipW, res.name, time.Now(), bytes.NewReader(res.data))
		}
	}
}

// SummaryCommand is used by Kong for CLI flags and commands.
type SummaryCommand struct {
	Filename   string `help:"Summary archive filename"`
	SkipServer bool   `help:"Skip fetching logs.zip from PMM Server"`
	Pprof      bool   `name:"pprof" help:"Include performance profiling data"`
}

func (cmd *SummaryCommand) makeArchive(ctx context.Context, globals *flags.GlobalFlags) error {
	var f *os.File
	var err error

	if f, err = os.Create(cmd.Filename); err != nil {
		return errors.WithStack(err)
	}

	defer func() {
		if e := f.Close(); e != nil && err == nil {
			err = errors.WithStack(e)
		}
	}()

	zipW := zip.NewWriter(f)

	defer func() {
		if e := zipW.Close(); e != nil && err == nil {
			err = errors.WithStack(e)
		}
	}()

	addClientData(ctx, zipW)

	if cmd.Pprof {
		addPprofData(ctx, zipW, cmd.SkipServer, globals)
	}

	if !cmd.SkipServer {
		addServerData(ctx, zipW, cmd.Pprof)
	}

	return nil
}

// RunCmdWithContext runs summary command.
func (cmd *SummaryCommand) RunCmdWithContext(ctx context.Context, globals *flags.GlobalFlags) (Result, error) {
	if cmd.Filename == "" {
		cmd.Filename = filename
	}

	if err := cmd.makeArchive(ctx, globals); err != nil {
		return nil, err
	}

	return &summaryResult{
		Filename: cmd.Filename,
	}, nil
}

// register command.
var (
	hostname, _ = os.Hostname()
	filename    = fmt.Sprintf("summary_%s_%s.zip",
		strings.ReplaceAll(hostname, ".", "_"), time.Now().Format("2006_01_02_15_04_05"))
)
