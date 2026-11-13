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

package jobs

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5" //nolint:gosec // the checksum PMM Extensions publishes today
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/types/known/durationpb"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

const (
	// OperationsAllowFileEnv names the variable holding the path of a file that lists, one per
	// line, the checksums of the operations this host accepts. Unset accepts what the server sends.
	OperationsAllowFileEnv = "PMM_AGENT_OPERATIONS_ALLOW_FILE"

	operationsCacheDir      = "operations"
	maxScriptSize           = 16 * 1024 * 1024
	maxOperationOutput      = 1024 * 1024
	maxOperationsAtOnce     = 8
	defaultOperationTimeout = 10 * time.Second
	maxOperationTimeout     = time.Minute
	fetchTimeout            = 30 * time.Second
	maxFileReadLength       = 2 * 1024 * 1024
)

var operationSlots = semaphore.NewWeighted(maxOperationsAtOnce)

// FetchScript downloads the script named by source to dest and verifies its checksum. A file
// that does not match is removed.
func FetchScript(ctx context.Context, source *agentv1.ScriptSource, dest string) (string, error) {
	sum, err := newChecksum(source.GetChecksum())
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.GetUrl(), nil)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	client := &http.Client{}
	if source.GetTlsSkipVerify() {
		client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} //nolint:gosec // asked for
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch: %s answered %s", source.GetUrl(), resp.Status)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o700) //nolint:gosec // dest is ours
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	_, err = io.Copy(io.MultiWriter(f, sum.h), io.LimitReader(resp.Body, maxScriptSize))
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = sum.verify()
	}
	if err != nil {
		_ = os.Remove(dest)
		return "", err
	}
	return dest, nil
}

type checksum struct {
	h    hash.Hash
	name string
	want string
}

func newChecksum(spec string) (*checksum, error) {
	name, want, ok := strings.Cut(spec, ":")
	if !ok || want == "" {
		return nil, fmt.Errorf("checksum %q is not <algorithm>:<hex>", spec)
	}
	c := &checksum{name: name, want: strings.ToLower(want)}
	switch name {
	case "sha256":
		c.h = sha256.New()
	case "md5":
		c.h = md5.New() //nolint:gosec // see the import
	default:
		return nil, fmt.Errorf("checksum algorithm %q is not supported", name)
	}
	return c, nil
}

func (c *checksum) verify() error {
	got := hex.EncodeToString(c.h.Sum(nil))
	if got != c.want {
		return fmt.Errorf("checksum mismatch: expected %s:%s, got %s:%s", c.name, c.want, c.name, got)
	}
	return nil
}

// RunOperation runs one short operation and returns its output in the response. The script is
// fetched once per checksum and kept; every run verifies the kept copy again.
func RunOperation(ctx context.Context, tempDir string, req *agentv1.RunOperationRequest) *agentv1.RunOperationResponse {
	resp := &agentv1.RunOperationResponse{}
	err := operationAllowed(req.GetSource().GetChecksum())
	if err != nil {
		resp.Error = err.Error()
		return resp
	}
	if !operationSlots.TryAcquire(1) {
		resp.Error = fmt.Sprintf("%d operations are already running on this host", maxOperationsAtOnce)
		return resp
	}
	defer operationSlots.Release(1)

	timeout := req.GetTimeout().AsDuration()
	if timeout <= 0 {
		timeout = defaultOperationTimeout
	}
	timeout = min(timeout, maxOperationTimeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	script, err := cachedScript(ctx, tempDir, req.GetSource())
	if err != nil {
		resp.Error = err.Error()
		return resp
	}

	start := time.Now()
	cmd := scriptCommand(ctx, req.GetSource(), script, req.GetArgs())
	cmd.Dir = "/"
	cmd.Env = scriptEnv("", "", nil)
	cmd.Stdin = strings.NewReader(req.GetStdin())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout := &cappedBuffer{limit: maxOperationOutput}
	stderr := &cappedBuffer{limit: maxOperationOutput}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.WaitDelay = time.Second
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }

	runErr := cmd.Run()
	resp.Duration = durationpb.New(time.Since(start))
	resp.Stdout = stdout.buf.Bytes()
	resp.Stderr = stderr.buf.Bytes()
	resp.Truncated = stdout.cut || stderr.cut
	switch {
	case cmd.ProcessState == nil:
		resp.ExitCode = -1
		resp.Error = runErr.Error()
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		resp.ExitCode = -1
		resp.Error = fmt.Sprintf("timed out after %s", timeout)
	default:
		resp.ExitCode = int32(cmd.ProcessState.ExitCode()) //nolint:gosec // exit codes fit
	}
	return resp
}

func operationAllowed(sum string) error {
	path := os.Getenv(OperationsAllowFileEnv)
	if path == "" {
		return nil
	}
	f, err := os.Open(path) //nolint:gosec // the operator names it
	if err != nil {
		return fmt.Errorf("operations are limited on this host and the list cannot be read: %w", err)
	}
	defer f.Close() //nolint:errcheck
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == sum {
			return nil
		}
	}
	return fmt.Errorf("operation %s is not on this host's approved list", sum)
}

func cachedScript(ctx context.Context, tempDir string, source *agentv1.ScriptSource) (string, error) {
	sum, err := newChecksum(source.GetChecksum())
	if err != nil {
		return "", err
	}
	dir := filepath.Join(tempDir, scriptsDirName, operationsCacheDir)
	err = os.MkdirAll(dir, 0o700)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, sum.name+"-"+sum.want)
	kept, err := os.ReadFile(path) //nolint:gosec // path is built from a validated checksum
	if err == nil {
		sum.h.Write(kept)
		if sum.verify() == nil {
			return path, nil
		}
	}
	tmp := fmt.Sprintf("%s.%d", path, time.Now().UnixNano())
	_, err = FetchScript(ctx, source, tmp)
	if err != nil {
		return "", err
	}
	return path, os.Rename(tmp, path)
}

type cappedBuffer struct {
	buf   bytes.Buffer
	limit int
	cut   bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	room := b.limit - b.buf.Len()
	if room < len(p) {
		b.cut = true
		if room > 0 {
			b.buf.Write(p[:room])
		}
		return len(p), nil
	}
	return b.buf.Write(p)
}

// ReadJobFile returns a piece of one file a job left in its output directory, or the directory's
// listing when no name is given.
func ReadJobFile(tempDir string, req *agentv1.ReadJobFileRequest) *agentv1.ReadJobFileResponse {
	resp := &agentv1.ReadJobFileResponse{}
	outDir := filepath.Join(RunDir(tempDir, req.GetJobId()), outputDirName)
	if req.GetName() == "" {
		entries, err := os.ReadDir(outDir)
		if err != nil {
			resp.Error = err.Error()
			return resp
		}
		for _, e := range entries {
			info, infoErr := e.Info()
			if infoErr != nil || !info.Mode().IsRegular() {
				continue
			}
			resp.Files = append(resp.Files, &agentv1.ReadJobFileResponse_File{Name: e.Name(), Size: info.Size()})
		}
		return resp
	}
	if filepath.Base(req.GetName()) != req.GetName() {
		resp.Error = "a file name cannot contain a path"
		return resp
	}
	f, err := os.Open(filepath.Join(outDir, req.GetName())) //nolint:gosec // confined to the run's directory above
	if err != nil {
		resp.Error = err.Error()
		return resp
	}
	defer f.Close() //nolint:errcheck
	info, err := f.Stat()
	if err != nil {
		resp.Error = err.Error()
		return resp
	}
	resp.Size = info.Size()
	length := req.GetLength()
	if length <= 0 || length > maxFileReadLength {
		length = maxFileReadLength
	}
	data := make([]byte, length)
	n, err := f.ReadAt(data, req.GetOffset())
	if err != nil && !errors.Is(err, io.EOF) {
		resp.Error = err.Error()
		return resp
	}
	resp.Data = data[:n]
	return resp
}
