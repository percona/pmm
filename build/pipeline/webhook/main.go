// Command webhook is a GitHub webhook listener that dispatches PR-comment
// commands ("@bot build", "@bot run unit tests", ...) to executable scripts.
// Each script receives PR context via environment variables and writes its
// reply comment to $REPLY_FILE.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := LoadConfig()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	gh := NewGitHubClient(cfg.GitHubToken)
	queue := NewQueue(NewRunner(cfg, gh), cfg.QueueSize)
	go queue.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("POST /webhook", NewWebhookHandler(cfg, queue, gh))
	mux.Handle("GET /logs/{delivery}", logsHandler(cfg))

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		TLSConfig:         &tls.Config{MinVersion: tls.VersionTLS12},
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tlsEnabled := cfg.TLSCertFile != "" && cfg.TLSKeyFile != ""
	go func() {
		slog.Info("listening",
			"addr", cfg.Addr,
			"tls", tlsEnabled,
			"bot", cfg.BotName,
			"make_dir", cfg.MakeDir,
		)
		var err error
		if tlsEnabled {
			err = srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	queue.Close()
}
