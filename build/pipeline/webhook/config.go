package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr                string
	TLSCertFile         string
	TLSKeyFile          string
	WebhookSecret       string
	GitHubToken         string
	BotName             string
	MakeDir             string
	WorkDir             string
	AllowedAssociations map[string]struct{}
	AllowedUsers        map[string]struct{}
	JobTimeout          time.Duration
	QueueSize           int
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Addr:    getenv("LISTEN_ADDR", ":7799"),
		BotName: strings.TrimPrefix(getenv("BOT_NAME", "pmm-bot"), "@"),
		MakeDir: os.Getenv("MAKE_DIR"),
		WorkDir: getenv("WORK_DIR", "/var/lib/webhook"),
	}
	cfg.WebhookSecret = os.Getenv("GITHUB_WEBHOOK_SECRET")
	cfg.GitHubToken = os.Getenv("GITHUB_TOKEN")
	if cfg.WebhookSecret == "" {
		return nil, fmt.Errorf("GITHUB_WEBHOOK_SECRET is required")
	}
	if cfg.GitHubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN is required")
	}

	cfg.TLSCertFile = os.Getenv("TLS_CERT_FILE")
	cfg.TLSKeyFile = os.Getenv("TLS_KEY_FILE")
	if (cfg.TLSCertFile == "") != (cfg.TLSKeyFile == "") {
		return nil, fmt.Errorf("TLS_CERT_FILE and TLS_KEY_FILE must be set together")
	}

	cfg.AllowedAssociations = parseSet(getenv("ALLOWED_ASSOCIATIONS", "MEMBER,OWNER,COLLABORATOR"), strings.ToUpper)
	if len(cfg.AllowedAssociations) == 0 {
		return nil, fmt.Errorf("ALLOWED_ASSOCIATIONS must list at least one value")
	}

	// Optional username allowlist applied after the association check. Empty
	// means no extra restriction; non-empty means the commenter's login must
	// match (case-insensitive). Off-list commenters get a -1 reaction.
	cfg.AllowedUsers = parseSet(os.Getenv("ALLOWED_USERS"), strings.ToLower)

	timeout, err := time.ParseDuration(getenv("JOB_TIMEOUT", "2h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JOB_TIMEOUT: %w", err)
	}
	cfg.JobTimeout = timeout
	cfg.QueueSize = 16

	return cfg, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func parseSet(s string, normalize func(string) string) map[string]struct{} {
	m := make(map[string]struct{})
	for p := range strings.SplitSeq(s, ",") {
		p = strings.TrimSpace(p)
		if normalize != nil {
			p = normalize(p)
		}
		if p != "" {
			m[p] = struct{}{}
		}
	}
	return m
}
