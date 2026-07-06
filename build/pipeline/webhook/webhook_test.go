package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestVerifySignature(t *testing.T) {
	secret := "topsecret"
	body := []byte(`{"hello":"world"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	good := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	t.Run("valid", func(t *testing.T) {
		if err := verifySignature(body, good, secret); err != nil {
			t.Fatalf("want nil, got %v", err)
		}
	})
	t.Run("wrong body", func(t *testing.T) {
		if err := verifySignature([]byte("tampered"), good, secret); err == nil {
			t.Fatal("want mismatch error")
		}
	})
	t.Run("wrong secret", func(t *testing.T) {
		if err := verifySignature(body, good, "nope"); err == nil {
			t.Fatal("want mismatch error")
		}
	})
	t.Run("missing prefix", func(t *testing.T) {
		if err := verifySignature(body, "bad", secret); err == nil {
			t.Fatal("want malformed error")
		}
	})
	t.Run("empty header", func(t *testing.T) {
		if err := verifySignature(body, "", secret); err == nil {
			t.Fatal("want malformed error")
		}
	})
}

func TestExtractCommand(t *testing.T) {
	re := regexp.MustCompile(`(?im)^[\s>]*@` + regexp.QuoteMeta("pmm-bot") + `\s+(.+?)\s*$`)
	cases := []struct {
		name, body, want string
	}{
		{"simple", "@pmm-bot build", "build"},
		{"multiword", "@pmm-bot run unit tests", "run unit tests"},
		{"surrounding text", "hey team, can we\n\n@pmm-bot build\n\nthanks", "build"},
		{"leading spaces", "   @pmm-bot build", "build"},
		{"case insensitive mention", "@PMM-Bot build", "build"},
		{"trailing whitespace", "@pmm-bot build   \n", "build"},
		{"quote block", "> @pmm-bot build", "build"},
		{"no mention", "please build this", ""},
		{"wrong bot", "@other-bot build", ""},
		{"mention only", "@pmm-bot", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractCommand(re, tc.body)
			if got != tc.want {
				t.Errorf("extractCommand(%q) = %q, want %q", tc.body, got, tc.want)
			}
		})
	}
}

func TestParseMakeTarget(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cases := map[string]string{
			"build client":      "client",
			"Build Client":      "client",
			"run unit tests":    "unit-tests",
			"Run API tests":     "api-tests",
			"  run   API tests": "api-tests",
			"BUILD all":         "all",
		}
		for in, want := range cases {
			got, err := parseMakeTarget(in)
			if err != nil {
				t.Errorf("parseMakeTarget(%q) error: %v", in, err)
				continue
			}
			if got != want {
				t.Errorf("parseMakeTarget(%q) = %q, want %q", in, got, want)
			}
		}
	})

	t.Run("rejected", func(t *testing.T) {
		bad := []string{
			"",                   // empty
			"build",              // no target
			"deploy production",  // unknown verb
			"run /etc/passwd",    // path separator → invalid target
			"run ../escape",      // path traversal
			"run -j8",            // leading dash → invalid target
			"build target;rm -rf", // shell metachars → invalid target
		}
		for _, in := range bad {
			if _, err := parseMakeTarget(in); err == nil {
				t.Errorf("parseMakeTarget(%q) = nil error, want failure", in)
			}
		}
	})
}

func TestServeHTTP_DispatchesValidComment(t *testing.T) {
	cfg := &Config{
		WebhookSecret:       "s3cret",
		BotName:             "pmm-bot",
		AllowedAssociations: map[string]struct{}{"MEMBER": {}},
	}
	q := NewQueue(nil, 1)
	h := NewWebhookHandler(cfg, q, nil)

	payload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"number":       42,
			"pull_request": map[string]any{"url": "https://example.com/pr/42"},
		},
		"comment": map[string]any{
			"id":                 int64(1001),
			"body":               "@pmm-bot build client",
			"author_association": "MEMBER",
			"user":               map[string]any{"login": "alice", "type": "User"},
		},
		"repository": map[string]any{
			"name":  "pmm",
			"owner": map[string]any{"login": "percona"},
		},
	}
	body, _ := json.Marshal(payload)
	rec := postSigned(t, h, "issue_comment", body, cfg.WebhookSecret)
	if rec.Code != http.StatusAccepted {
		dumpBody(t, rec)
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}

	select {
	case job := <-q.ch:
		if job.MakeTarget != "client" || job.Command != "build client" ||
			job.PRNumber != 42 || job.Owner != "percona" || job.Repo != "pmm" {
			t.Errorf("unexpected job: %+v", job)
		}
	default:
		t.Fatal("expected a job in queue")
	}
}

func TestServeHTTP_RejectsBadSignature(t *testing.T) {
	cfg := &Config{WebhookSecret: "s3cret", BotName: "pmm-bot"}
	h := NewWebhookHandler(cfg, NewQueue(nil, 1), nil)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-GitHub-Event", "issue_comment")
	req.Header.Set("X-Hub-Signature-256", "sha256=deadbeef")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestServeHTTP_SkipsUnauthorizedAssociation(t *testing.T) {
	cfg := &Config{
		WebhookSecret:       "s3cret",
		BotName:             "pmm-bot",
		AllowedAssociations: map[string]struct{}{"MEMBER": {}},
	}
	q := NewQueue(nil, 1)
	h := NewWebhookHandler(cfg, q, nil)

	payload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"number":       42,
			"pull_request": map[string]any{"url": "x"},
		},
		"comment": map[string]any{
			"id":                 int64(1),
			"body":               "@pmm-bot build",
			"author_association": "NONE",
			"user":               map[string]any{"login": "stranger", "type": "User"},
		},
		"repository": map[string]any{
			"name":  "pmm",
			"owner": map[string]any{"login": "percona"},
		},
	}
	body, _ := json.Marshal(payload)
	rec := postSigned(t, h, "issue_comment", body, cfg.WebhookSecret)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	select {
	case job := <-q.ch:
		t.Fatalf("unexpected job enqueued: %+v", job)
	default:
	}
}

func TestServeHTTP_AllowlistGate(t *testing.T) {
	cfg := &Config{
		WebhookSecret:       "s3cret",
		BotName:             "pmm-bot",
		AllowedAssociations: map[string]struct{}{"MEMBER": {}},
		AllowedUsers:        map[string]struct{}{"alice": {}},
	}
	q := NewQueue(nil, 1)
	h := NewWebhookHandler(cfg, q, nil)

	mkPayload := func(login string) []byte {
		body, _ := json.Marshal(map[string]any{
			"action": "created",
			"issue": map[string]any{
				"number":       42,
				"pull_request": map[string]any{"url": "x"},
			},
			"comment": map[string]any{
				"id":                 int64(1),
				"body":               "@pmm-bot build all",
				"author_association": "MEMBER",
				"user":               map[string]any{"login": login, "type": "User"},
			},
			"repository": map[string]any{
				"name":  "pmm",
				"owner": map[string]any{"login": "percona"},
			},
		})
		return body
	}

	t.Run("on list dispatches", func(t *testing.T) {
		rec := postSigned(t, h, "issue_comment", mkPayload("Alice"), cfg.WebhookSecret)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
		}
		select {
		case <-q.ch:
		default:
			t.Fatal("expected job to be enqueued for allowlisted user")
		}
	})

	t.Run("off list rejected", func(t *testing.T) {
		rec := postSigned(t, h, "issue_comment", mkPayload("bob"), cfg.WebhookSecret)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
		}
		select {
		case job := <-q.ch:
			t.Fatalf("unexpected job for non-allowlisted user: %+v", job)
		default:
		}
	})
}

func TestServeHTTP_Ping(t *testing.T) {
	cfg := &Config{WebhookSecret: "s3cret", BotName: "pmm-bot"}
	h := NewWebhookHandler(cfg, NewQueue(nil, 1), nil)
	rec := postSigned(t, h, "ping", []byte(`{"zen":"Keep it simple."}`), cfg.WebhookSecret)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func postSigned(t *testing.T, h http.Handler, event string, body []byte, secret string) *httptest.ResponseRecorder {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", event)
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	req.Header.Set("X-GitHub-Delivery", "test-delivery-id")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func dumpBody(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	b, _ := io.ReadAll(rec.Body)
	t.Logf("response body: %s", string(b))
}
