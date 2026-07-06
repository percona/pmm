package main

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// deliveryRE constrains the path parameter to the shape of a GitHub delivery
// UUID (hex chars + dashes). Strict pattern + length cap blocks path traversal
// and globs that could spread across the work dir.
var deliveryRE = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)

// logsHandler serves run.log files keyed by the GitHub delivery UUID. The job
// dirs on disk are named `<repo>-<pr>-<delivery>`, so we glob for the unique
// suffix and serve the run.log inside.
func logsHandler(cfg *Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		delivery := r.PathValue("delivery")
		if !deliveryRE.MatchString(delivery) {
			http.Error(w, "invalid delivery id", http.StatusBadRequest)
			return
		}

		matches, err := filepath.Glob(filepath.Join(cfg.WorkDir, "*-"+delivery))
		if err != nil || len(matches) == 0 {
			http.NotFound(w, r)
			return
		}
		if len(matches) > 1 {
			slog.Warn("multiple log dirs match delivery", "delivery", delivery, "count", len(matches))
		}

		path := filepath.Join(matches[0], "run.log")
		// Belt-and-suspenders: ensure the resolved path stays inside WorkDir
		// even if a symlink is in play.
		clean, err := filepath.EvalSymlinks(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		root, _ := filepath.EvalSymlinks(cfg.WorkDir)
		if !strings.HasPrefix(clean, root+string(os.PathSeparator)) {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		slog.Info("serving log", "delivery", delivery, "remote", r.RemoteAddr)
		http.ServeFile(w, r, clean)
	})
}
