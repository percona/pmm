// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package supervisord

import (
	"archive/zip"
	"bufio"
	"bytes"
	"container/ring"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/percona/pmm/utils/pdeathsig"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/percona/pmm-managed/utils/logger"
)

const (
	maxLogReadLines = 1000
	maxLogReadBytes = 1024 * 1024
)

// fileContent represents logs.zip item.
type fileContent struct {
	Name     string
	Modified time.Time
	Data     []byte
	Err      error
}

// Logs is responsible for interactions with logs.
type Logs struct {
	pmmVersion       string
	pmmUpdateChecker *PMMUpdateChecker
}

// NewLogs creates a new Logs service.
// n is a number of last lines of log to read.
func NewLogs(pmmVersion string, pmmUpdateChecker *PMMUpdateChecker) *Logs {
	return &Logs{
		pmmVersion:       pmmVersion,
		pmmUpdateChecker: pmmUpdateChecker,
	}
}

// Zip creates .zip archive with all logs.
func (l *Logs) Zip(ctx context.Context, w io.Writer) error {
	start := time.Now()
	log := logger.Get(ctx).WithField("component", "logs")
	log.WithField("d", time.Since(start).Seconds()).Info("Starting...")
	defer func() {
		log.WithField("d", time.Since(start).Seconds()).Info("Done.")
	}()

	zw := zip.NewWriter(w)
	now := time.Now().UTC()

	files := l.files(ctx)
	log.WithField("d", time.Since(start).Seconds()).Infof("Collected %d files.", len(files))

	for _, file := range files {
		if ctx.Err() != nil {
			log.WithField("d", time.Since(start).Seconds()).Warnf("%s; skipping the rest of the files", ctx.Err())
			break
		}

		if file.Err != nil {
			log.WithField("d", time.Since(start).Seconds()).Errorf("%s: %s", file.Name, file.Err)

			// do not let a single error break the whole archive
			if len(file.Data) > 0 {
				file.Data = append(file.Data, "\n\n"...)
			}
			file.Data = append(file.Data, file.Err.Error()...)
		}

		if file.Modified.IsZero() {
			file.Modified = now
		}

		f, err := zw.CreateHeader(&zip.FileHeader{
			Name:     file.Name,
			Method:   zip.Deflate,
			Modified: file.Modified,
		})
		if err != nil {
			return errors.Wrap(err, "failed to create zip file header")
		}
		if _, err = f.Write(file.Data); err != nil {
			return errors.Wrap(err, "failed to write zip file data")
		}
	}

	if err := addAdminSummary(ctx, zw); err != nil {
		// do not let it break the whole archive
		log.WithField("d", time.Since(start).Seconds()).Errorf("addAdminSummary: %+v", err)
	}

	if err := zw.Close(); err != nil {
		return errors.Wrap(err, "failed to close zip file")
	}
	return nil
}

// files reads log/config files and returns content.
func (l *Logs) files(ctx context.Context) []fileContent {
	files := make([]fileContent, 0, 20)

	// add logs
	logs, err := filepath.Glob("/srv/logs/*.log")
	if err != nil {
		logger.Get(ctx).WithField("component", "logs").Error(err)
	}
	for _, f := range logs {
		b, m, err := readLog(f, maxLogReadLines, maxLogReadBytes)
		files = append(files, fileContent{
			Name:     filepath.Base(f),
			Modified: m,
			Data:     b,
			Err:      err,
		})
	}

	// add configs
	for _, f := range []string{
		"/etc/nginx/nginx.conf",
		"/etc/nginx/conf.d/pmm.conf",
		"/etc/nginx/conf.d/pmm-ssl.conf",

		"/etc/prometheus.yml",

		"/etc/supervisord.conf",
		"/etc/supervisord.d/pmm.ini",
		"/etc/supervisord.d/prometheus.ini",
		"/etc/supervisord.d/qan-api2.ini",

		"/usr/local/percona/pmm2/config/pmm-agent.yaml",
	} {
		b, m, err := readFile(f)
		files = append(files, fileContent{
			Name:     filepath.Base(f),
			Modified: m,
			Data:     b,
			Err:      err,
		})
	}

	// add PMM version
	files = append(files, fileContent{
		Name: "pmm-version.txt",
		Data: []byte(l.pmmVersion + "\n"),
	})

	// add supervisord status
	b, err := readCmdOutput(ctx, "supervisorctl", "status")
	files = append(files, fileContent{
		Name: "supervisorctl_status.log",
		Data: b,
		Err:  err,
	})

	// add systemd status for OVF/AMI
	b, err = readCmdOutput(ctx, "systemctl", "-l", "status")
	files = append(files, fileContent{
		Name: "systemctl_status.log",
		Data: b,
		Err:  err,
	})

	// add Prometheus targets
	b, err = readURL(ctx, "http://127.0.0.1:9090/prometheus/api/v1/targets")
	files = append(files, fileContent{
		Name: "prometheus_targets.json",
		Data: b,
		Err:  err,
	})

	// update checker installed info
	b, err = json.Marshal(l.pmmUpdateChecker.Installed(ctx))
	files = append(files, fileContent{
		Name: "installed.json",
		Data: b,
		Err:  err,
	})

	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files
}

// readLog reads last lines (up to given number of lines and bytes) from given file,
// and returns them together with modification time.
func readLog(name string, maxLines int, maxBytes int64) ([]byte, time.Time, error) {
	var m time.Time
	f, err := os.Open(name) //nolint:gosec
	if err != nil {
		return nil, m, errors.WithStack(err)
	}
	defer f.Close() //nolint:errcheck

	fi, err := f.Stat()
	if err != nil {
		return nil, m, errors.WithStack(err)
	}
	m = fi.ModTime()
	if fi.Size() > maxBytes {
		if _, err = f.Seek(-maxBytes, io.SeekEnd); err != nil {
			return nil, m, errors.WithStack(err)
		}
	}

	r := ring.New(maxLines)
	s := bufio.NewScanner(f)
	for s.Scan() {
		r.Value = []byte(s.Text() + "\n")
		r = r.Next()
	}
	if err = s.Err(); err != nil {
		return nil, m, errors.WithStack(err)
	}

	res := make([]byte, 0, maxBytes)
	r.Do(func(v interface{}) {
		if v != nil {
			res = append(res, v.([]byte)...)
		}
	})
	return res, m, nil
}

// readFile reads the whole file and returns content together with modification time.
func readFile(name string) ([]byte, time.Time, error) {
	var m time.Time
	b, err := ioutil.ReadFile(name) //nolint:gosec
	if err != nil {
		return nil, m, errors.WithStack(err)
	}

	if fi, err := os.Stat(name); err == nil {
		m = fi.ModTime()
	}
	return b, m, nil
}

// readCmdOutput reads command's combined output.
func readCmdOutput(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...) //nolint:gosec
	pdeathsig.Set(cmd, unix.SIGKILL)
	return cmd.CombinedOutput()
}

// readURL reads HTTP GET url response.
func readURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer resp.Body.Close() //nolint:errcheck

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// indent JSON output
	mt, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mt == "application/json" {
		var buf bytes.Buffer
		if json.Indent(&buf, b, "", "  ") == nil {
			b = buf.Bytes()
		}
	}
	return b, nil
}

func addAdminSummary(ctx context.Context, zw *zip.Writer) error {
	sf, err := ioutil.TempFile("", "*-pmm-admin-summary.zip")
	if err != nil {
		return errors.WithStack(err)
	}
	if err := sf.Close(); err != nil {
		return errors.WithStack(err)
	}
	defer os.Remove(sf.Name()) //nolint:errcheck

	cmd := exec.CommandContext(ctx, "pmm-admin", "summary", "--skip-server", "--filename", sf.Name()) //nolint:gosec
	pdeathsig.Set(cmd, unix.SIGKILL)
	cmd.Stdout = os.Stderr // stdout to stderr
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return errors.Wrap(err, "cannot run pmm-admin summary")
	}

	zr, err := zip.OpenReader(sf.Name())
	if err != nil {
		return errors.WithStack(err)
	}
	defer zr.Close() //nolint:errcheck

	for _, file := range zr.File {
		fw, err := zw.CreateHeader(&zip.FileHeader{
			Name:     file.Name,
			Method:   zip.Deflate,
			Modified: file.Modified,
		})
		if err != nil {
			return errors.Wrap(err, "failed to create zip file header")
		}

		fr, err := file.Open()
		if err != nil {
			return errors.WithStack(err)
		}

		if _, err = io.Copy(fw, fr); err != nil {
			fr.Close() //nolint:errcheck
			return errors.WithStack(err)
		}

		if err = fr.Close(); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
