package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

type Job struct {
	Delivery   string
	Owner      string
	Repo       string
	PRNumber   int
	Commenter  string
	CommentID  int64
	Command    string // raw "build client" / "run api tests" — for logs and replies
	MakeTarget string // sanitized make target ("client", "api-tests")
}

type Runner struct {
	cfg *Config
	gh  *GitHubClient
}

func NewRunner(cfg *Config, gh *GitHubClient) *Runner {
	return &Runner{cfg: cfg, gh: gh}
}

func (r *Runner) Run(job Job) {
	ctx, cancel := context.WithTimeout(context.Background(), r.cfg.JobTimeout)
	defer cancel()

	logger := jobLogger(job)
	logger.Info("job start")

	if err := r.gh.ReactToComment(ctx, job.Owner, job.Repo, job.CommentID, "eyes"); err != nil {
		logger.Warn("reaction failed", "err", err)
	}

	pr, err := r.gh.GetPR(ctx, job.Owner, job.Repo, job.PRNumber)
	if err != nil {
		r.report(ctx, job, fmt.Sprintf(":x: Failed to fetch PR metadata: `%v`", err))
		return
	}

	jobDir := filepath.Join(r.cfg.WorkDir, fmt.Sprintf("%s-%d-%s", job.Repo, job.PRNumber, job.Delivery))
	if err := os.MkdirAll(jobDir, 0o755); err != nil {
		r.report(ctx, job, fmt.Sprintf(":x: Failed to create work directory: `%v`", err))
		return
	}

	replyFile := filepath.Join(jobDir, "reply.md")
	logFile := filepath.Join(jobDir, "run.log")

	f, err := os.Create(logFile)
	if err != nil {
		r.report(ctx, job, fmt.Sprintf(":x: Failed to open log file: `%v`", err))
		return
	}
	defer f.Close()

	cmd := exec.CommandContext(ctx, "make", job.MakeTarget)
	// MakeDir is the directory containing the Makefile. Empty means inherit
	// the listener's cwd (e.g. WorkingDirectory= from the systemd unit).
	cmd.Dir = r.cfg.MakeDir
	cmd.Env = append(os.Environ(),
		"PR_NUMBER="+strconv.Itoa(job.PRNumber),
		"PR_HEAD_SHA="+pr.Head.SHA,
		"PR_HEAD_SHORT_SHA="+shortSHA(pr.Head.SHA),
		"PR_HEAD_REF="+pr.Head.Ref,
		"PR_HEAD_REPO="+pr.Head.Repo.FullName,
		"PR_HEAD_CLONE_URL="+pr.Head.Repo.CloneURL,
		"PR_BASE_REF="+pr.Base.Ref,
		"PR_REPO_OWNER="+job.Owner,
		"PR_REPO_NAME="+job.Repo,
		"PR_TAG="+fmt.Sprintf("PR-%d-%s", job.PRNumber, shortSHA(pr.Head.SHA)),
		"COMMENTER="+job.Commenter,
		"COMMAND="+job.Command,
		"REPLY_FILE="+replyFile,
		"JOB_DIR="+jobDir,
		"LOG_UUID="+job.Delivery,
	)
	cmd.Stdout = io.MultiWriter(f, os.Stdout)
	cmd.Stderr = io.MultiWriter(f, os.Stderr)

	start := time.Now()
	runErr := cmd.Run()
	elapsed := time.Since(start)

	if runErr != nil {
		logger.Error("job failed", "elapsed", elapsed, "err", runErr)
		body := fmt.Sprintf(":x: `%s` failed after %s: `%v`", job.Command, fmtDuration(elapsed), runErr)
		if reply := readReply(replyFile); reply != "" {
			body += "\n\n" + reply
		}
		r.report(ctx, job, body)
		return
	}

	logger.Info("job ok", "elapsed", elapsed)
	body := readReply(replyFile)
	if body == "" {
		body = fmt.Sprintf(":white_check_mark: `%s` completed in %s.", job.Command, fmtDuration(elapsed))
	}
	r.report(ctx, job, body)
}

func (r *Runner) report(ctx context.Context, job Job, body string) {
	if err := r.gh.PostComment(ctx, job.Owner, job.Repo, job.PRNumber, body); err != nil {
		jobLogger(job).Error("post comment failed", "err", err)
	}
}

func jobLogger(job Job) *slog.Logger {
	return slog.With(
		"delivery", job.Delivery,
		"repo", job.Owner+"/"+job.Repo,
		"pr", job.PRNumber,
		"command", job.Command,
	)
}

func readReply(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func shortSHA(sha string) string {
	if len(sha) < 7 {
		return sha
	}
	return sha[:7]
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
