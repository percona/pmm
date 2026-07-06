package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const maxPayloadBytes = 10 << 20 // 10 MiB — GitHub caps webhook payloads at ~25 MiB.

type issueCommentEvent struct {
	Action  string `json:"action"`
	Comment struct {
		ID                int64  `json:"id"`
		Body              string `json:"body"`
		AuthorAssociation string `json:"author_association"`
		User              struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"user"`
		HTMLURL string `json:"html_url"`
	} `json:"comment"`
	Issue struct {
		Number      int `json:"number"`
		PullRequest *struct {
			URL string `json:"url"`
		} `json:"pull_request,omitempty"`
	} `json:"issue"`
	Repository struct {
		Name  string `json:"name"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
}

type WebhookHandler struct {
	cfg   *Config
	queue *Queue
	gh    *GitHubClient
	re    *regexp.Regexp
}

func NewWebhookHandler(cfg *Config, q *Queue, gh *GitHubClient) *WebhookHandler {
	re := regexp.MustCompile(`(?im)^[\s>]*@` + regexp.QuoteMeta(cfg.BotName) + `\s+(.+?)\s*$`)
	return &WebhookHandler{cfg: cfg, queue: q, gh: gh, re: re}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	event := r.Header.Get("X-GitHub-Event")
	delivery := r.Header.Get("X-GitHub-Delivery")

	body, err := io.ReadAll(io.LimitReader(r.Body, maxPayloadBytes))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	if err := verifySignature(body, r.Header.Get("X-Hub-Signature-256"), h.cfg.WebhookSecret); err != nil {
		slog.Warn("signature rejected", "delivery", delivery, "err", err)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	switch event {
	case "ping":
		w.WriteHeader(http.StatusOK)
		return
	case "issue_comment":
	default:
		w.WriteHeader(http.StatusAccepted)
		return
	}

	var ev issueCommentEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	if ev.Action != "created" || ev.Issue.PullRequest == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if strings.EqualFold(ev.Comment.User.Type, "Bot") {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	cmd := extractCommand(h.re, ev.Comment.Body)
	if cmd == "" {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	repoFull := ev.Repository.Owner.Login + "/" + ev.Repository.Name
	logger := slog.With(
		"delivery", delivery,
		"repo", repoFull,
		"pr", ev.Issue.Number,
		"commenter", ev.Comment.User.Login,
		"command", cmd,
	)
	logger.Info("command received")

	if _, ok := h.cfg.AllowedAssociations[ev.Comment.AuthorAssociation]; !ok {
		logger.Warn("unauthorized commenter", "association", ev.Comment.AuthorAssociation)
		w.WriteHeader(http.StatusAccepted)
		return
	}

	target, err := parseMakeTarget(cmd)
	if err != nil {
		logger.Warn("rejected command", "err", err)
		w.WriteHeader(http.StatusAccepted)
		return
	}

	if len(h.cfg.AllowedUsers) > 0 {
		if _, ok := h.cfg.AllowedUsers[strings.ToLower(ev.Comment.User.Login)]; !ok {
			logger.Info("commenter not on allowlist", "login", ev.Comment.User.Login)
			h.reactUnauthorized(r.Context(), ev.Repository.Owner.Login, ev.Repository.Name, ev.Comment.ID)
			w.WriteHeader(http.StatusAccepted)
			return
		}
	}

	job := Job{
		Delivery:   delivery,
		Owner:      ev.Repository.Owner.Login,
		Repo:       ev.Repository.Name,
		PRNumber:   ev.Issue.Number,
		Commenter:  ev.Comment.User.Login,
		CommentID:  ev.Comment.ID,
		Command:    cmd,
		MakeTarget: target,
	}
	if err := h.queue.Enqueue(job); err != nil {
		logger.Error("enqueue failed", "err", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func extractCommand(re *regexp.Regexp, body string) string {
	m := re.FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// allowedVerbs are the leading words a commenter may use; all of them map to
// `make`. The remaining words become the make target.
var allowedVerbs = map[string]struct{}{
	"build": {},
	"run":   {},
}

// makeTargetRE matches a conservative subset of valid make target names:
// lowercase alphanumerics plus '-', '_', '.', no leading punctuation.
var makeTargetRE = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// parseMakeTarget turns a comment command like "Run API tests" into the
// corresponding make target ("api-tests"). Returns an error if the verb is
// not allowlisted or the resulting target fails validation.
func parseMakeTarget(cmd string) (string, error) {
	fields := strings.Fields(cmd)
	if len(fields) < 2 {
		return "", fmt.Errorf("expected '<verb> <target...>', got %q", cmd)
	}
	verb := strings.ToLower(fields[0])
	if _, ok := allowedVerbs[verb]; !ok {
		return "", fmt.Errorf("unknown verb %q (allowed: build, run)", verb)
	}
	target := strings.ToLower(strings.Join(fields[1:], "-"))
	if !makeTargetRE.MatchString(target) {
		return "", fmt.Errorf("invalid make target %q", target)
	}
	return target, nil
}

// reactUnauthorized fires a 😕 reaction on the triggering comment to signal
// that the commenter is not on the allowlist. Best-effort, with a short
// timeout — failure here doesn't affect the request response.
func (h *WebhookHandler) reactUnauthorized(parent context.Context, owner, repo string, commentID int64) {
	if h.gh == nil {
		return
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	if err := h.gh.ReactToComment(ctx, owner, repo, commentID, "confused"); err != nil {
		slog.Warn("react failed", "owner", owner, "repo", repo, "comment_id", commentID, "err", err)
	}
}

func verifySignature(body []byte, header, secret string) error {
	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) {
		return errors.New("missing or malformed signature")
	}
	got, err := hex.DecodeString(header[len(prefix):])
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	if !hmac.Equal(got, mac.Sum(nil)) {
		return errors.New("signature mismatch")
	}
	return nil
}
