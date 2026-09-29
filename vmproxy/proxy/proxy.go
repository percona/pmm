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

// Package proxy provides http reverse proxy functionality.
package proxy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Config defines options for starting proxy.
type Config struct {
	// Name of the header to check for filters. Case insensitive.
	HeaderName string
	// Address the proxy is listening on
	ListenAddress string
	// Target URL to forward requests to
	TargetURL *url.URL
}

// RunProxy starts proxy which adds extra filters based on configuration.
func RunProxy(cfg Config) error {
	logrus.Infof("Starting to proxy at http://%s to %s", cfg.ListenAddress, cfg.TargetURL.String())

	err := http.ListenAndServe(cfg.ListenAddress, getHandler(cfg)) //nolint:gosec
	return err
}

func getHandler(cfg Config) http.HandlerFunc {
	rProxy := &httputil.ReverseProxy{
		Director: director(cfg.TargetURL, cfg.HeaderName),
		// Without this, httputil's default handler reports upstream failures through
		// the standard logger, so they reach the log without a level and cannot be
		// filtered alongside everything else here.
		ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
			l := logrus.WithError(err).WithFields(logrus.Fields{
				"method": req.Method,
				"path":   req.URL.Path,
			})
			// A status is written in every branch: returning without one makes
			// net/http send 200, which would be worse than any error code here.
			switch {
			case errors.Is(err, context.Canceled):
				// The client hung up -- Grafana cancels superseded queries, and nginx
				// drops the upstream connection when a viewer navigates away. Nothing
				// failed upstream, so warning here would be routine noise.
				l.Debug("Client cancelled request")
				rw.WriteHeader(http.StatusBadGateway)
			case errors.Is(err, context.DeadlineExceeded):
				l.Warn("Timed out proxying request")
				rw.WriteHeader(http.StatusGatewayTimeout)
			default:
				l.Warn("Failed to proxy request")
				rw.WriteHeader(http.StatusBadGateway)
			}
		},
	}

	return func(rw http.ResponseWriter, req *http.Request) {
		logrus.Debugf("%s: %s", req.Method, req.URL)

		if failOnInvalidHeader(rw, req, cfg.HeaderName) {
			return
		}

		rProxy.ServeHTTP(rw, req)
	}
}

func failOnInvalidHeader(rw http.ResponseWriter, req *http.Request, headerName string) bool {
	if filters := req.Header.Get(headerName); filters != "" {
		_, err := parseFilters(filters)
		if err != nil {
			// The header value is client-supplied and carries access filters, so log
			// only the parse error, not the value itself.
			logrus.WithError(err).WithFields(logrus.Fields{
				"method": req.Method,
				"path":   req.URL.Path,
				"header": headerName,
			}).Warn("Rejecting request with unparsable filter header")
			rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
			rw.WriteHeader(http.StatusPreconditionFailed)
			io.WriteString(rw, fmt.Sprintf("Failed to parse %s header", headerName)) //nolint:errcheck
			return true
		}
	}

	return false
}

func director(target *url.URL, headerName string) func(*http.Request) {
	return func(req *http.Request) {
		prepareRequest(req, target, headerName)
	}
}

func prepareRequest(req *http.Request, target *url.URL, headerName string) {
	now := time.Now()

	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host

	hostHeader := target.Host
	if hostHeader != "" {
		req.Host = hostHeader
		req.Header.Set("Host", hostHeader)
	}
	if target.User != nil {
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(target.User.String())))
	}

	rp, err := target.Parse(strings.TrimPrefix(req.URL.Path, "/"))
	if err != nil {
		logrus.Error(err)
	}
	req.URL.Path = rp.Path

	// Replace extra filters if present
	if filters := req.Header.Get(headerName); filters != "" {
		q := req.URL.Query()
		q.Del("extra_filters[]")

		parsed, err := parseFilters(filters)
		if err != nil {
			logrus.Error(err)
		}

		for _, f := range parsed {
			q.Add("extra_filters[]", f)
		}

		req.URL.RawQuery = q.Encode()

		logrus.Debugf(
			"Parsed filters: %#v, Target URL: %s, Time spent: %s",
			parsed, req.URL, time.Since(now),
		)
	}

	// Do not trust the client
	req.Header.Del("X-Forwarded-For")

	if _, ok := req.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		req.Header.Set("User-Agent", "")
	}
}

func parseFilters(filters string) ([]string, error) {
	var parsed []string

	decoded, err := base64.StdEncoding.DecodeString(filters)
	if err != nil {
		return nil, fmt.Errorf("could not decode filters header: %w", err)
	}

	err = json.Unmarshal(decoded, &parsed)
	if err != nil {
		return nil, fmt.Errorf("could not parse filters JSON: %w", err)
	}

	return parsed, nil
}
