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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

const (
	// RunScript is the type of a job that fetches a script, verifies it and runs it.
	RunScript = JobType("run_script")

	// JobIDEnv names the variable every process of a run carries. It is how the processes a
	// script left behind are found once the script itself is gone.
	JobIDEnv = "PMM_JOB_ID"
	// JobOutputDirEnv names the variable holding the directory a script leaves its files in.
	JobOutputDirEnv = "PMM_JOB_OUTPUT_DIR"

	scriptsDirName      = "run-script"
	outputDirName       = "out"
	readBufferSize      = 32 * 1024
	stopGracePeriod     = 5 * time.Second
	drainGracePeriod    = 2 * time.Second
	defaultKeepFilesFor = time.Hour
	basePath            = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
)

// RunScriptParams are the inputs of a RunScriptJob.
type RunScriptParams struct {
	Source         *agentv1.ScriptSource
	Args           []string
	Env            map[string]string
	Stdin          []byte
	MaxOutputBytes int64
	KeepFilesFor   time.Duration
	// RunAs names the OS user the script runs as; empty keeps pmm-agent's own.
	RunAs string
	// TempDir is pmm-agent's temporary directory; runs live in a directory of their own under it.
	TempDir string
}

// RunScriptJob fetches a script, verifies its checksum and runs it, streaming both outputs.
type RunScriptJob struct {
	id      string
	timeout time.Duration
	params  RunScriptParams
	l       *logrus.Entry

	credential *syscall.Credential
}

// NewRunScriptJob creates a run_script job.
func NewRunScriptJob(id string, timeout time.Duration, params RunScriptParams) *RunScriptJob {
	return &RunScriptJob{
		id:      id,
		timeout: timeout,
		params:  params,
		l:       logrus.WithFields(logrus.Fields{"id": id, "type": RunScript}),
	}
}

// ID returns Job ID.
func (j *RunScriptJob) ID() string { return j.id }

// Type returns Job type.
func (j *RunScriptJob) Type() JobType { return RunScript }

// Timeout returns Job timeout.
func (j *RunScriptJob) Timeout() time.Duration { return j.timeout }

// DSN returns an empty DSN: a script is not tied to one database instance, so only the runner's
// total capacity limits how many run at once.
func (j *RunScriptJob) DSN() string { return "" }

// Run executes the job. A script that could not be prepared and a script that failed both end in
// a JobResult; only what stops the job from reporting at all is returned as an error.
func (j *RunScriptJob) Run(ctx context.Context, send Send) error {
	runDir := RunDir(j.params.TempDir, j.id)
	outDir := filepath.Join(runDir, outputDirName)
	// The parent lets a run-as user reach its own run directory without listing the others.
	err := os.MkdirAll(filepath.Dir(runDir), 0o711) //nolint:gosec // traversal only
	if err == nil {
		err = os.Chmod(filepath.Dir(runDir), 0o711) //nolint:gosec // traversal only
	}
	if err == nil {
		err = os.MkdirAll(outDir, 0o700)
	}
	if err != nil {
		j.sendResult(send, &agentv1.JobResult_RunScript{ExitCode: -1, PrepareError: err.Error()})
		return nil
	}
	defer j.cleanup(runDir, outDir)

	script, err := FetchScript(ctx, j.params.Source, filepath.Join(runDir, "script"))
	if err != nil {
		j.sendResult(send, &agentv1.JobResult_RunScript{ExitCode: -1, PrepareError: err.Error()})
		return nil
	}
	credential, err := runAsCredential(j.params.RunAs, runDir, outDir, script)
	if err != nil {
		j.sendResult(send, &agentv1.JobResult_RunScript{ExitCode: -1, PrepareError: err.Error()})
		return nil
	}
	j.credential = credential

	j.execute(ctx, send, script, outDir)
	removeErr := os.Remove(script)
	if removeErr != nil {
		j.l.Warnf("Failed to remove the script: %s.", removeErr)
	}
	return nil
}

func (j *RunScriptJob) execute(ctx context.Context, send Send, script, outDir string) {
	// The job stops its script itself, group and leftovers, so the command is not bound to ctx.
	cmd := scriptCommand(context.WithoutCancel(ctx), j.params.Source, script, j.params.Args)
	cmd.Dir = outDir
	cmd.Env = scriptEnv(j.id, outDir, j.params.Env)
	cmd.Stdin = bytes.NewReader(j.params.Stdin)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL, Credential: j.credential}

	outR, outW, err := os.Pipe()
	if err != nil {
		j.sendResult(send, &agentv1.JobResult_RunScript{ExitCode: -1, PrepareError: err.Error()})
		return
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		outR.Close() //nolint:errcheck,gosec
		outW.Close() //nolint:errcheck,gosec
		j.sendResult(send, &agentv1.JobResult_RunScript{ExitCode: -1, PrepareError: err.Error()})
		return
	}
	cmd.Stdout = outW
	cmd.Stderr = errW

	// Start and Wait share one locked thread: Pdeathsig fires when the thread that forked the
	// child exits, not when the process does.
	started := make(chan error, 1)
	exited := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		startErr := cmd.Start()
		started <- startErr
		if startErr == nil {
			exited <- cmd.Wait()
		}
	}()
	startErr := <-started
	outW.Close() //nolint:errcheck,gosec
	errW.Close() //nolint:errcheck,gosec
	if startErr != nil {
		outR.Close() //nolint:errcheck,gosec
		errR.Close() //nolint:errcheck,gosec
		j.sendResult(send, &agentv1.JobResult_RunScript{ExitCode: -1, PrepareError: startErr.Error()})
		return
	}

	out := newOutputSender(j.id, send, j.params.MaxOutputBytes)
	var pumps sync.WaitGroup
	pumps.Go(func() { out.pump(outR, agentv1.JobProgress_Logs_STREAM_STDOUT) })
	pumps.Go(func() { out.pump(errR, agentv1.JobProgress_Logs_STREAM_STDERR) })

	result := &agentv1.JobResult_RunScript{}
	select {
	case <-exited:
	case <-ctx.Done():
		result.EndedBy = "stopped"
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result.EndedBy = "timeout"
		}
		stopGroup(cmd.Process.Pid, exited)
	}
	result.ExitCode = int32(cmd.ProcessState.ExitCode()) //nolint:gosec // exit codes fit

	result.LeftoverKilled = uint32(KillLeftovers(j.id)) //nolint:gosec // a process count

	// A process that outlived the script may still hold the pipes; killing it above closes them,
	// and the deadline covers one that could not be killed.
	drained := make(chan struct{})
	go func() {
		pumps.Wait()
		close(drained)
	}()
	select {
	case <-drained:
	case <-time.After(drainGracePeriod):
		outR.Close() //nolint:errcheck,gosec
		errR.Close() //nolint:errcheck,gosec
		<-drained
	}
	result.DroppedOutputBytes = out.dropped.Load()
	out.deliver(result)
}

func (j *RunScriptJob) sendResult(send Send, result *agentv1.JobResult_RunScript) {
	send(&agentv1.JobMessageRequest{Message: &agentv1.JobMessageRequest_Result{Result: &agentv1.JobResult{
		JobId:     j.id,
		Timestamp: timestamppb.Now(),
		Result:    &agentv1.JobResult_RunScript_{RunScript: result},
	}}})
}

// cleanup removes the run's directory at once when the script left no files, and after the
// keep period when it did.
func (j *RunScriptJob) cleanup(runDir, outDir string) {
	entries, err := os.ReadDir(outDir)
	if err != nil || len(entries) == 0 {
		removeErr := os.RemoveAll(runDir)
		if removeErr != nil {
			j.l.Warnf("Failed to remove %s: %s.", runDir, removeErr)
		}
		return
	}
	keep := j.params.KeepFilesFor
	if keep <= 0 {
		keep = defaultKeepFilesFor
	}
	time.AfterFunc(keep, func() {
		removeErr := os.RemoveAll(runDir)
		if removeErr != nil {
			j.l.Warnf("Failed to remove %s: %s.", runDir, removeErr)
		}
	})
}

// runAsCredential resolves the user a script is to run as and hands that user the run's
// directory, its output directory and the script. An empty name keeps pmm-agent's own user.
func runAsCredential(name string, paths ...string) (*syscall.Credential, error) {
	if name == "" {
		return nil, nil //nolint:nilnil // no credential change is a valid outcome
	}
	u, err := user.Lookup(name)
	if err != nil {
		return nil, fmt.Errorf("run as %q: %w", name, err)
	}
	uid, uidErr := strconv.ParseUint(u.Uid, 10, 32)
	gid, gidErr := strconv.ParseUint(u.Gid, 10, 32)
	if uidErr != nil || gidErr != nil {
		return nil, fmt.Errorf("run as %q: unusable ids %s:%s", name, u.Uid, u.Gid)
	}
	for _, p := range paths {
		err = os.Chown(p, int(uid), int(gid))
		if err != nil {
			return nil, fmt.Errorf("run as %q: %w", name, err)
		}
	}
	return &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}, nil
}

// SweepRunDirs removes the directories of runs that died with a pmm-agent that was killed: a
// clean stop removes the temporary directory itself. Call it before any job can start.
func SweepRunDirs(tempDir string) {
	base := filepath.Join(tempDir, scriptsDirName)
	entries, err := os.ReadDir(base)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.Name() == operationsCacheDir {
			continue
		}
		err = os.RemoveAll(filepath.Join(base, e.Name()))
		if err != nil {
			logrus.Warnf("Failed to remove %s: %s.", filepath.Join(base, e.Name()), err)
		}
	}
}

// RunDir returns the directory of one run under pmm-agent's temporary directory.
func RunDir(tempDir, jobID string) string {
	return filepath.Join(tempDir, scriptsDirName, strings.ReplaceAll(jobID, "/", "_"))
}

func scriptCommand(ctx context.Context, source *agentv1.ScriptSource, script string, args []string) *exec.Cmd {
	interpreter := source.GetInterpreter()
	if interpreter == "" {
		interpreter = "/bin/sh"
	}
	return exec.CommandContext(ctx, interpreter, append([]string{script}, args...)...) //nolint:gosec // the job is to run it
}

// scriptEnv is the environment a script starts with: nothing of pmm-agent's own, which holds the
// credentials it connects to the server with.
func scriptEnv(jobID, outDir string, extra map[string]string) []string {
	env := []string{
		"PATH=" + basePath,
		"HOME=" + os.Getenv("HOME"),
		"LANG=C.UTF-8",
		JobIDEnv + "=" + jobID,
		JobOutputDirEnv + "=" + outDir,
	}
	for k, v := range extra {
		env = append(env, k+"="+v)
	}
	return env
}

// stopGroup ends the script's process group: SIGTERM, then SIGKILL after the grace period.
func stopGroup(pid int, exited <-chan error) {
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	select {
	case <-exited:
	case <-time.After(stopGracePeriod):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		<-exited
	}
}

// KillLeftovers kills every process still carrying the run's JobIDEnv value and returns how many
// it found. A process that moved to its own session escapes the group signal but keeps the
// environment it was started with.
func KillLeftovers(jobID string) int {
	marker := []byte(JobIDEnv + "=" + jobID + "\x00")
	procs, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}
	self := os.Getpid()
	killed := 0
	for _, p := range procs {
		var pid int
		_, scanErr := fmt.Sscanf(p.Name(), "%d", &pid)
		if scanErr != nil || pid == self {
			continue
		}
		environ, readErr := os.ReadFile(filepath.Join("/proc", p.Name(), "environ"))
		if readErr != nil {
			continue
		}
		if !bytes.HasPrefix(environ, marker) && !bytes.Contains(environ, append([]byte{0}, marker...)) {
			continue
		}
		if syscall.Kill(pid, syscall.SIGKILL) == nil {
			killed++
		}
	}
	return killed
}

// outputSender turns what a script writes into ordered JobProgress chunks tagged by stream and
// keeps them, with the final result, in a queue on the host until the server has taken each one.
// Delivery runs in a goroutine of its own (run), so the connection going away while the script
// runs or as it ends never blocks the script and never drops its output or its outcome: the
// queue waits on the host for the connection to return and is then delivered in order, and the
// server stores each chunk at most once.
type outputSender struct {
	jobID   string
	send    Send
	limit   int64
	chunkID atomic.Uint32
	sent    atomic.Int64
	dropped atomic.Int64

	mu     sync.Mutex
	cond   *sync.Cond
	queue  []agentv1.AgentResponsePayload
	sealed bool
	done   chan struct{}
}

func newOutputSender(jobID string, send Send, limit int64) *outputSender {
	o := &outputSender{jobID: jobID, send: send, limit: limit, done: make(chan struct{})}
	o.cond = sync.NewCond(&o.mu)
	go o.run()
	return o
}

// run delivers queued messages in order. send blocks while the host is cut off, so only this
// goroutine waits for the connection; the pumps and the script go on filling the queue.
func (o *outputSender) run() {
	defer close(o.done)
	for {
		o.mu.Lock()
		for len(o.queue) == 0 && !o.sealed {
			o.cond.Wait()
		}
		if len(o.queue) == 0 {
			o.mu.Unlock()
			return
		}
		msg := o.queue[0]
		o.mu.Unlock()

		o.send(msg)

		o.mu.Lock()
		o.queue = o.queue[1:]
		o.mu.Unlock()
	}
}

func (o *outputSender) enqueue(msg agentv1.AgentResponsePayload) {
	o.mu.Lock()
	o.queue = append(o.queue, msg)
	o.cond.Signal()
	o.mu.Unlock()
}

func (o *outputSender) pump(r io.ReadCloser, stream agentv1.JobProgress_Logs_Stream) {
	defer r.Close() //nolint:errcheck
	buf := make([]byte, readBufferSize)
	var carry []byte
	for {
		n, err := r.Read(buf[len(carry):])
		data := buf[:len(carry)+n]
		copy(data, carry)
		carry = carry[:0]
		if err == nil {
			// Keep a multi-byte character cut by the read boundary for the next chunk.
			for i := 1; i <= utf8.UTFMax && i <= len(data); i++ {
				tail := data[len(data)-i:]
				if !utf8.RuneStart(tail[0]) {
					continue
				}
				if !utf8.FullRune(tail) {
					carry = append(carry, tail...)
					data = data[:len(data)-i]
				}
				break
			}
		}
		if len(data) > 0 {
			o.emit(strings.ToValidUTF8(string(data), "�"), stream)
		}
		if err != nil {
			return
		}
	}
}

func (o *outputSender) emit(data string, stream agentv1.JobProgress_Logs_Stream) {
	if o.limit > 0 && o.sent.Load() >= o.limit {
		o.dropped.Add(int64(len(data)))
		return
	}
	o.sent.Add(int64(len(data)))
	o.progress(&agentv1.JobProgress_Logs{ChunkId: o.chunkID.Add(1) - 1, Data: data, Stream: stream})
}

// progress queues one chunk as a message the server has to acknowledge.
func (o *outputSender) progress(logs *agentv1.JobProgress_Logs) {
	o.enqueue(&agentv1.JobMessageRequest{Message: &agentv1.JobMessageRequest_Progress{Progress: &agentv1.JobProgress{
		JobId:     o.jobID,
		Timestamp: timestamppb.Now(),
		Result:    &agentv1.JobProgress_Logs_{Logs: logs},
	}}})
}

// deliver queues the end-of-output marker and the final result, then waits until the whole queue
// has been handed to the server. The job stays running on the host until that is done, so a run
// is never recorded finished before its true outcome has reached the server.
func (o *outputSender) deliver(result *agentv1.JobResult_RunScript) {
	o.progress(&agentv1.JobProgress_Logs{ChunkId: o.chunkID.Add(1) - 1, Done: true})
	o.enqueue(&agentv1.JobMessageRequest{Message: &agentv1.JobMessageRequest_Result{Result: &agentv1.JobResult{
		JobId:     o.jobID,
		Timestamp: timestamppb.Now(),
		Result:    &agentv1.JobResult_RunScript_{RunScript: result},
	}}})
	o.mu.Lock()
	o.sealed = true
	o.cond.Signal()
	o.mu.Unlock()
	<-o.done
}
