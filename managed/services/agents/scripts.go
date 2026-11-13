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

package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/reform.v1"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
)

const (
	// ScriptRetentionEnv names the variable holding how long finished script runs are kept, as a
	// Go duration. Unset keeps them, which is what the jobs table does for every job type today.
	ScriptRetentionEnv = "PMM_SCRIPT_RUNS_RETENTION"

	scriptStartTimeout     = 10 * time.Second
	scriptProbeTimeout     = 2 * time.Second
	scriptCallGrace        = 5 * time.Second
	scriptRetentionEvery   = time.Minute
	scriptReconcileMinAge  = 10 * time.Second
	scriptReconcileSettle  = 2 * time.Second
	scriptLogsDefaultLimit = 1000
	scriptRunsDefaultLimit = 100
	scriptLostMessage      = "the run was lost: pmm-agent no longer has it (it restarted, or the result was dropped with the connection)"
)

// ScriptsService starts scripts on pmm-agents and serves the HTTP surface callers use for them.
type ScriptsService struct {
	db   *reform.DB
	r    *Registry
	jobs *JobsService
	l    *logrus.Entry

	// retention is how long finished runs are kept, in nanoseconds; 0 keeps them.
	retention atomic.Int64
}

// NewScriptsService returns a new scripts service.
func NewScriptsService(db *reform.DB, registry *Registry, jobs *JobsService) *ScriptsService {
	s := &ScriptsService{db: db, r: registry, jobs: jobs, l: logrus.WithField("component", "agents/scriptsService")}
	// A real change carries this as a numbered migration. Done here so the stock binary still
	// starts against the same database when the prototype is removed.
	_, err := db.Exec(`ALTER TABLE job_logs ADD COLUMN IF NOT EXISTS stream SMALLINT NOT NULL DEFAULT 0`)
	if err != nil {
		s.l.Errorf("Failed to add job_logs.stream: %s.", err)
	}
	return s
}

// Run deletes finished runs older than the retention period until ctx is done. The period comes
// from ScriptRetentionEnv at start and can be changed while running (setRetention).
func (s *ScriptsService) Run(ctx context.Context) {
	retention, err := time.ParseDuration(os.Getenv(ScriptRetentionEnv))
	if err == nil && retention > 0 {
		s.retention.Store(int64(retention))
	}
	s.l.Infof("Finished script runs are kept for %s (0 means without limit).", time.Duration(s.retention.Load()))
	ticker := time.NewTicker(scriptRetentionEvery)
	defer ticker.Stop()
	for {
		if keep := time.Duration(s.retention.Load()); keep > 0 {
			_, err := s.db.Exec(`DELETE FROM jobs WHERE type = $1 AND done AND updated_at <= $2`,
				models.RunScriptJob, models.Now().Add(-keep))
			if err != nil {
				s.l.Errorf("Failed to delete old script runs: %s.", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// setRetention changes how long finished runs are kept, until the next restart.
func (s *ScriptsService) setRetention(r *http.Request) (any, error) {
	seconds, err := strconv.ParseFloat(r.URL.Query().Get("seconds"), 64)
	if err != nil || seconds < 0 {
		return nil, scriptErrorf(http.StatusBadRequest, "seconds: a number of seconds, 0 for no limit")
	}
	s.retention.Store(int64(seconds * float64(time.Second)))
	return map[string]any{"retention_s": seconds}, nil
}

// Register adds the scripts routes to mux.
func (s *ScriptsService) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/scripts/runs", s.handle(s.startRun))
	mux.HandleFunc("GET /v1/scripts/runs", s.handle(s.listRuns))
	mux.HandleFunc("GET /v1/scripts/runs/{id}", s.handle(s.getRun))
	mux.HandleFunc("GET /v1/scripts/runs/{id}/logs", s.handle(s.getLogs))
	mux.HandleFunc("POST /v1/scripts/runs/{id}/stop", s.handle(s.stopRun))
	mux.HandleFunc("GET /v1/scripts/runs/{id}/files", s.handle(s.listFiles))
	mux.HandleFunc("GET /v1/scripts/runs/{id}/files/{name}", s.downloadFile)
	mux.HandleFunc("POST /v1/scripts/call", s.handle(s.call))
	mux.HandleFunc("GET /v1/scripts/hosts", s.handle(s.listHosts))
	mux.HandleFunc("PUT /v1/scripts/retention", s.handle(s.setRetention))
}

type scriptHost struct {
	NodeName   string `json:"node_name"`
	Address    string `json:"address"`
	PMMAgentID string `json:"pmm_agent_id"`
	Version    string `json:"version"`
	Connected  bool   `json:"connected"`
	CanRun     bool   `json:"can_run"`
	Reason     string `json:"reason"`
}

// listHosts returns every pmm-agent with whether it can take a script right now, and why not.
// "Connected" is the registry's word and stays true for a host that lost its network without
// closing the stream, so each connected agent is also asked a question only a script-capable
// agent can answer, with a short deadline.
func (s *ScriptsService) listHosts(r *http.Request) (any, error) {
	structs, err := s.db.SelectAllFrom(models.AgentTable, "WHERE agent_type = $1 ORDER BY agent_id", models.PMMAgentType)
	if err != nil {
		return nil, err
	}
	hosts := make([]*scriptHost, len(structs))
	var wg sync.WaitGroup
	for i, st := range structs {
		agent := st.(*models.Agent) //nolint:forcetypeassert
		host := &scriptHost{PMMAgentID: agent.AgentID, Version: pointer.GetString(agent.Version)}
		hosts[i] = host
		node, nodeErr := models.FindNodeByID(s.db.Querier, pointer.GetString(agent.RunsOnNodeID))
		if nodeErr == nil {
			host.NodeName, host.Address = node.NodeName, node.Address
		}
		conn, getErr := s.r.get(agent.AgentID)
		if getErr != nil {
			host.Reason = "pmm-agent is not connected"
			continue
		}
		host.Connected = true
		wg.Go(func() {
			ctx, cancel := context.WithTimeout(r.Context(), scriptProbeTimeout)
			defer cancel()
			_, probeErr := conn.channel.SendAndWaitResponse(ctx, &agentv1.ReadJobFileRequest{JobId: "probe"})
			switch {
			case probeErr == nil:
				host.CanRun = true
			case errors.Is(probeErr, context.DeadlineExceeded):
				host.Reason = fmt.Sprintf("connected, but pmm-agent did not answer within %s", scriptProbeTimeout)
			case status.Code(probeErr) == codes.Unimplemented || strings.Contains(probeErr.Error(), "unimplemented"):
				host.Reason = "pmm-agent is too old to run scripts"
			default:
				host.Reason = probeErr.Error()
			}
		})
	}
	wg.Wait()
	return map[string]any{"hosts": hosts}, nil
}

type scriptHTTPError struct {
	status int
	msg    string
}

func (e *scriptHTTPError) Error() string { return e.msg }

func scriptErrorf(status int, format string, args ...any) error {
	return &scriptHTTPError{status: status, msg: fmt.Sprintf(format, args...)}
}

func (s *ScriptsService) handle(f func(r *http.Request) (any, error)) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		res, err := f(r)
		rw.Header().Set("Content-Type", "application/json")
		if err != nil {
			status := http.StatusInternalServerError
			var httpErr *scriptHTTPError
			if errors.As(err, &httpErr) {
				status = httpErr.status
			}
			rw.WriteHeader(status)
			res = map[string]string{"error": err.Error()}
		}
		encodeErr := json.NewEncoder(rw).Encode(res)
		if encodeErr != nil {
			s.l.Warnf("Failed to write a response: %s.", encodeErr)
		}
	}
}

type scriptTarget struct {
	PMMAgentID string `json:"pmm_agent_id"`
	NodeName   string `json:"node_name"`
}

type scriptSource struct {
	URL           string `json:"url"`
	Checksum      string `json:"checksum"`
	Interpreter   string `json:"interpreter"`
	TLSSkipVerify bool   `json:"tls_skip_verify"`
}

func (src scriptSource) proto() *agentv1.ScriptSource {
	return &agentv1.ScriptSource{Url: src.URL, Checksum: src.Checksum, Interpreter: src.Interpreter, TlsSkipVerify: src.TLSSkipVerify}
}

type startRunRequest struct {
	scriptTarget
	scriptSource

	Args           []string          `json:"args"`
	Env            map[string]string `json:"env"`
	Stdin          string            `json:"stdin"`
	TimeoutS       float64           `json:"timeout_s"`
	MaxOutputBytes int64             `json:"max_output_bytes"`
	KeepFilesForS  float64           `json:"keep_files_for_s"`
	RunAs          string            `json:"run_as"`
}

type scriptRun struct {
	RunID              string    `json:"run_id"`
	PMMAgentID         string    `json:"pmm_agent_id"`
	NodeName           string    `json:"node_name"`
	State              string    `json:"state"`
	Done               bool      `json:"done"`
	Error              string    `json:"error"`
	ExitCode           *int32    `json:"exit_code"`
	PrepareError       string    `json:"prepare_error"`
	EndedBy            string    `json:"ended_by"`
	LeftoverKilled     uint32    `json:"leftover_killed"`
	DroppedOutputBytes int64     `json:"dropped_output_bytes"`
	AgentConnected     bool      `json:"agent_connected"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// resolveAgent returns the ID of the pmm-agent a request names, directly or by its node's name.
func (s *ScriptsService) resolveAgent(target scriptTarget) (string, error) {
	if target.PMMAgentID != "" {
		return target.PMMAgentID, nil
	}
	if target.NodeName == "" {
		return "", scriptErrorf(http.StatusBadRequest, "pmm_agent_id or node_name is required")
	}
	node, err := models.FindNodeByName(s.db.Querier, target.NodeName)
	if err != nil {
		return "", scriptErrorf(http.StatusNotFound, "node %q: %s", target.NodeName, err)
	}
	agents, err := models.FindPMMAgentsRunningOnNode(s.db.Querier, node.NodeID)
	if err != nil {
		return "", err
	}
	if len(agents) == 0 {
		return "", scriptErrorf(http.StatusNotFound, "node %q has no pmm-agent", target.NodeName)
	}
	return agents[0].AgentID, nil
}

func (s *ScriptsService) startRun(r *http.Request) (any, error) {
	var req startRunRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, scriptErrorf(http.StatusBadRequest, "body: %s", err)
	}
	agentID, err := s.resolveAgent(req.scriptTarget)
	if err != nil {
		return nil, err
	}

	// The record holds what was asked for, never the standard input or the values of the
	// environment: both may carry a secret.
	envNames := make([]string, 0, len(req.Env))
	for name := range req.Env {
		envNames = append(envNames, name)
	}
	sort.Strings(envNames)
	timeout := time.Duration(req.TimeoutS * float64(time.Second))
	job, err := models.CreateJob(s.db.Querier, models.CreateJobParams{
		PMMAgentID: agentID,
		Type:       models.RunScriptJob,
		Timeout:    timeout,
		Data: &models.JobData{RunScript: &models.RunScriptJobData{
			NodeName:    req.NodeName,
			URL:         req.URL,
			Checksum:    req.Checksum,
			Interpreter: req.Interpreter,
			Args:        req.Args,
			EnvNames:    envNames,
			RunAs:       req.RunAs,
		}},
	})
	if err != nil {
		return nil, err
	}

	startErr := s.sendStart(r.Context(), job, &req)
	if startErr != nil {
		job.Done = true
		job.Error = "not started: " + startErr.Error()
		err = s.db.Update(job)
		if err != nil {
			return nil, err
		}
	}
	return s.describe(job), nil
}

func (s *ScriptsService) sendStart(ctx context.Context, job *models.Job, req *startRunRequest) error {
	agent, err := s.r.get(job.PMMAgentID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, scriptStartTimeout)
	defer cancel()
	resp, err := agent.channel.SendAndWaitResponse(ctx, &agentv1.StartJobRequest{
		JobId:   job.ID,
		Timeout: durationpb.New(job.Timeout),
		Job: &agentv1.StartJobRequest_RunScript_{RunScript: &agentv1.StartJobRequest_RunScript{
			Source:         req.proto(),
			Args:           req.Args,
			Env:            req.Env,
			Stdin:          req.Stdin,
			MaxOutputBytes: req.MaxOutputBytes,
			KeepFilesFor:   durationpb.New(time.Duration(req.KeepFilesForS * float64(time.Second))),
			RunAs:          req.RunAs,
		}},
	})
	if err != nil {
		return err
	}
	if e := resp.(*agentv1.StartJobResponse).Error; e != "" { //nolint:forcetypeassert
		return errors.New(e)
	}
	return nil
}

func (s *ScriptsService) describe(job *models.Job) *scriptRun {
	run := &scriptRun{
		RunID:          job.ID,
		PMMAgentID:     job.PMMAgentID,
		Done:           job.Done,
		Error:          job.Error,
		AgentConnected: s.r.IsConnected(job.PMMAgentID),
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
	}
	if job.Data != nil && job.Data.RunScript != nil {
		run.NodeName = job.Data.RunScript.NodeName
	}
	var res *models.RunScriptJobResult
	if job.Result != nil {
		res = job.Result.RunScript
	}
	if res != nil {
		run.ExitCode = &res.ExitCode
		run.PrepareError = res.PrepareError
		run.EndedBy = res.EndedBy
		run.LeftoverKilled = res.LeftoverKilled
		run.DroppedOutputBytes = res.DroppedOutputBytes
	}
	switch {
	case !job.Done:
		run.State = "running"
	case res == nil:
		run.State = "error"
	case res.PrepareError != "":
		run.State = "prepare_failed"
	case res.EndedBy != "":
		run.State = res.EndedBy
	case res.ExitCode == 0:
		run.State = "succeeded"
	default:
		run.State = "failed"
	}
	return run
}

func (s *ScriptsService) findRun(r *http.Request) (*models.Job, error) {
	job, err := models.FindJobByID(s.db.Querier, r.PathValue("id"))
	if err != nil || job.Type != models.RunScriptJob {
		return nil, scriptErrorf(http.StatusNotFound, "run %q not found", r.PathValue("id"))
	}
	return job, nil
}

func (s *ScriptsService) getRun(r *http.Request) (any, error) {
	job, err := s.findRun(r)
	if err != nil {
		return nil, err
	}
	return s.describe(job), nil
}

func (s *ScriptsService) listRuns(r *http.Request) (any, error) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = scriptRunsDefaultLimit
	}
	tail := "WHERE type = $1"
	args := []any{models.RunScriptJob}
	if done := r.URL.Query().Get("done"); done != "" {
		tail += " AND done = $2"
		args = append(args, done == "true")
	}
	if agentID := r.URL.Query().Get("pmm_agent_id"); agentID != "" {
		args = append(args, agentID)
		tail += fmt.Sprintf(" AND pmm_agent_id = $%d", len(args))
	}
	args = append(args, limit)
	tail += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args))
	structs, err := s.db.SelectAllFrom(models.JobTable, tail, args...)
	if err != nil {
		return nil, err
	}
	runs := make([]*scriptRun, len(structs))
	for i, st := range structs {
		runs[i] = s.describe(st.(*models.Job)) //nolint:forcetypeassert
	}
	return map[string]any{"runs": runs}, nil
}

type scriptLogChunk struct {
	ChunkID int    `json:"chunk_id"`
	Stream  string `json:"stream"`
	Data    string `json:"data"`
}

func (s *ScriptsService) getLogs(r *http.Request) (any, error) {
	job, err := s.findRun(r)
	if err != nil {
		return nil, err
	}
	q := r.URL.Query()
	offset, _ := strconv.Atoi(q.Get("offset"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = scriptLogsDefaultLimit
	}
	rows, err := s.db.Query(
		`SELECT chunk_id, stream, data, last_chunk FROM job_logs WHERE job_id = $1 AND chunk_id >= $2 ORDER BY chunk_id LIMIT $3`,
		job.ID, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	want := q.Get("stream")
	chunks := []scriptLogChunk{}
	next := offset
	end := false
	for rows.Next() {
		var c scriptLogChunk
		var stream int
		var last bool
		err = rows.Scan(&c.ChunkID, &stream, &c.Data, &last)
		if err != nil {
			return nil, err
		}
		next = c.ChunkID + 1
		end = end || last
		c.Stream = "stdout"
		if agentv1.JobProgress_Logs_Stream(stream) == agentv1.JobProgress_Logs_STREAM_STDERR { //nolint:gosec // a small enum
			c.Stream = "stderr"
		}
		if last || (want != "" && want != c.Stream) {
			continue
		}
		chunks = append(chunks, c)
	}
	return map[string]any{"chunks": chunks, "next_offset": next, "end": end, "done": job.Done}, rows.Err()
}

func (s *ScriptsService) stopRun(r *http.Request) (any, error) {
	job, err := s.findRun(r)
	if err != nil {
		return nil, err
	}
	err = s.jobs.StopJob(job.ID)
	if err != nil {
		return nil, scriptErrorf(http.StatusConflict, "stop: %s", err)
	}
	return map[string]any{"run_id": job.ID, "stop_sent": !job.Done}, nil
}

func (s *ScriptsService) readFile(ctx context.Context, job *models.Job, name string, offset int64) (*agentv1.ReadJobFileResponse, error) {
	agent, err := s.r.get(job.PMMAgentID)
	if err != nil {
		return nil, scriptErrorf(http.StatusConflict, "%s", err)
	}
	ctx, cancel := context.WithTimeout(ctx, scriptStartTimeout)
	defer cancel()
	resp, err := agent.channel.SendAndWaitResponse(ctx, &agentv1.ReadJobFileRequest{JobId: job.ID, Name: name, Offset: offset})
	if err != nil {
		return nil, err
	}
	file := resp.(*agentv1.ReadJobFileResponse) //nolint:forcetypeassert
	if file.Error != "" {
		return nil, scriptErrorf(http.StatusNotFound, "%s", file.Error)
	}
	return file, nil
}

func (s *ScriptsService) listFiles(r *http.Request) (any, error) {
	job, err := s.findRun(r)
	if err != nil {
		return nil, err
	}
	resp, err := s.readFile(r.Context(), job, "", 0)
	if err != nil {
		return nil, err
	}
	files := []map[string]any{}
	for _, f := range resp.Files {
		files = append(files, map[string]any{"name": f.Name, "size": f.Size})
	}
	return map[string]any{"files": files}, nil
}

// downloadFile relays one file of a run from the host, piece by piece, without storing it.
func (s *ScriptsService) downloadFile(rw http.ResponseWriter, r *http.Request) {
	job, err := s.findRun(r)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}
	var offset int64
	for {
		piece, err := s.readFile(r.Context(), job, r.PathValue("name"), offset)
		if err != nil {
			if offset == 0 {
				http.Error(rw, err.Error(), http.StatusNotFound)
			}
			return
		}
		if offset == 0 {
			rw.Header().Set("Content-Type", "application/octet-stream")
			rw.Header().Set("Content-Length", strconv.FormatInt(piece.Size, 10))
		}
		_, err = rw.Write(piece.Data)
		if err != nil {
			return
		}
		offset += int64(len(piece.Data))
		if offset >= piece.Size || len(piece.Data) == 0 {
			return
		}
	}
}

type callRequest struct {
	scriptTarget
	scriptSource

	Args     []string `json:"args"`
	Stdin    string   `json:"stdin"`
	TimeoutS float64  `json:"timeout_s"`
}

// call runs one short operation on a host and returns its output. Nothing is stored.
func (s *ScriptsService) call(r *http.Request) (any, error) {
	start := time.Now()
	var req callRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, scriptErrorf(http.StatusBadRequest, "body: %s", err)
	}
	agentID, err := s.resolveAgent(req.scriptTarget)
	if err != nil {
		return nil, err
	}
	agent, err := s.r.get(agentID)
	if err != nil {
		return nil, scriptErrorf(http.StatusConflict, "%s", err)
	}
	timeout := time.Duration(req.TimeoutS * float64(time.Second))
	if timeout <= 0 {
		timeout = scriptStartTimeout
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout+scriptCallGrace)
	defer cancel()
	resp, err := agent.channel.SendAndWaitResponse(ctx, &agentv1.RunOperationRequest{
		Source:  req.proto(),
		Args:    req.Args,
		Stdin:   req.Stdin,
		Timeout: durationpb.New(timeout),
	})
	if err != nil {
		return nil, scriptErrorf(http.StatusGatewayTimeout, "%s", err)
	}
	op := resp.(*agentv1.RunOperationResponse) //nolint:forcetypeassert
	return map[string]any{
		"error":       op.Error,
		"exit_code":   op.ExitCode,
		"stdout":      string(op.Stdout),
		"stderr":      string(op.Stderr),
		"truncated":   op.Truncated,
		"duration_ms": float64(op.Duration.AsDuration().Microseconds()) / 1000,
		"managed_ms":  float64(time.Since(start).Microseconds()) / 1000,
	}, nil
}

// handleRunScriptResult stores the outcome of a run_script job.
func handleRunScriptResult(job *models.Job, res *agentv1.JobResult_RunScript) error {
	if job.Type != models.RunScriptJob {
		return fmt.Errorf("result type %s doesn't match job type %s", models.RunScriptJob, job.Type)
	}
	// A result that arrives after the run was closed as lost is the truth about it.
	job.Error = ""
	job.Result = &models.JobResult{RunScript: &models.RunScriptJobResult{
		ExitCode:           res.ExitCode,
		PrepareError:       res.PrepareError,
		EndedBy:            res.EndedBy,
		LeftoverKilled:     res.LeftoverKilled,
		DroppedOutputBytes: res.DroppedOutputBytes,
	}}
	return nil
}

// reconcileRunScripts asks a pmm-agent that has just connected about every script run the
// server still holds as unfinished for it, and closes the ones the agent no longer has.
// Without it a run whose agent restarted, or whose result was dropped with the old connection,
// would stay unfinished for ever.
func (s *JobsService) reconcileRunScripts(ctx context.Context, agent *pmmAgentInfo) {
	structs, err := s.db.SelectAllFrom(models.JobTable,
		"WHERE pmm_agent_id = $1 AND type = $2 AND NOT done AND created_at < $3",
		agent.id, models.RunScriptJob, models.Now().Add(-scriptReconcileMinAge))
	if err != nil {
		s.l.Errorf("Failed to list unfinished script runs of %s: %s.", agent.id, err)
		return
	}
	for _, st := range structs {
		job := st.(*models.Job) //nolint:forcetypeassert
		resp, err := agent.channel.SendAndWaitResponse(ctx, &agentv1.JobStatusRequest{JobId: job.ID})
		if err != nil {
			s.l.Warnf("Failed to ask %s about run %s: %s.", agent.id, job.ID, err)
			return
		}
		if resp.(*agentv1.JobStatusResponse).Alive { //nolint:forcetypeassert
			continue
		}
		// The result may be right behind the answer on the stream; let the handler take it.
		time.Sleep(scriptReconcileSettle)
		err = s.db.InTransaction(func(tx *reform.TX) error {
			current, findErr := models.FindJobByID(tx.Querier, job.ID)
			if findErr != nil || current.Done {
				return findErr
			}
			current.Done = true
			current.Error = scriptLostMessage
			return tx.Update(current)
		})
		if err != nil {
			s.l.Errorf("Failed to close lost run %s: %s.", job.ID, err)
		}
	}
}
