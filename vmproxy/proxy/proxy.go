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
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// readOnlyPaths are the VictoriaMetrics endpoints reachable through the proxy. The proxy is
// the only route Grafana's Metrics data source has to VictoriaMetrics, and Grafana forwards
// whatever sub-path it is given, so anything not listed here -- snapshots, the admin and
// debug surface, ingestion -- would otherwise be reachable by any user who can query a
// dashboard. Admins reach the rest through the admin-gated /prometheus location in nginx,
// which does not pass through the proxy.
//
// Label values are matched separately by labelValuesPath.
var readOnlyPaths = map[string]struct{}{
	"/api/v1/query":            {},
	"/api/v1/query_range":      {},
	"/api/v1/query_exemplars":  {},
	"/api/v1/series":           {},
	"/api/v1/labels":           {},
	"/api/v1/metadata":         {},
	"/api/v1/rules":            {},
	"/api/v1/alerts":           {},
	"/api/v1/status/buildinfo": {},
	"/api/v1/export":           {},

	// Cardinality diagnostics. VictoriaMetrics applies extra_filters[] here, so a
	// restricted viewer sees only the series their own filters select. The scrape target
	// endpoints behave the opposite way -- VictoriaMetrics ignores the filters and hands
	// back every target with its listen port -- so they are not listed, and an admin
	// reaches them through the marker below.
	"/api/v1/status/tsdb": {},
}

// adminHeaderValue is the only value the admin marker is honoured with; pmm-managed sets it.
const adminHeaderValue = "1"

// Config defines options for starting proxy.
type Config struct {
	// Name of the header to check for filters. Case insensitive.
	HeaderName string
	// Address the proxy is listening on
	ListenAddress string
	// Target URL to forward requests to
	TargetURL *url.URL
	// Name of the header marking a request as coming from an admin. Case insensitive.
	AdminHeaderName string
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

		if failOnDisallowedPath(rw, req, cfg.AdminHeaderName) {
			return
		}

		if failOnInvalidHeader(rw, req, cfg.HeaderName) {
			return
		}

		rProxy.ServeHTTP(rw, req)
	}
}

// failOnDisallowedPath answers the request with 403 and reports whether it did, so the
// caller returns instead of proxying a path VictoriaMetrics must not be asked for.
func failOnDisallowedPath(rw http.ResponseWriter, req *http.Request, adminHeaderName string) bool {
	if isPathAllowed(req.URL.Path, isAdminRequest(req, adminHeaderName)) {
		return false
	}

	// Logged so a legitimate path missing from the allow-list can be found from the
	// logs of whoever reports the broken panel.
	logrus.WithFields(logrus.Fields{
		"method": req.Method,
		"path":   req.URL.Path,
	}).Warn("Refusing request to a path outside the read-only allow-list")

	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(http.StatusForbidden)
	io.WriteString(rw, "Path is not allowed through the VictoriaMetrics proxy") //nolint:errcheck,gosec

	return true
}

// isPathAllowed reports whether the path may be forwarded to VictoriaMetrics.
func isPathAllowed(p string, isAdmin bool) bool {
	// The allow-list exists to bound what a dashboard user reaches through Grafana's data
	// source, which is never marked: /graph requires no role, so pmm-managed does not
	// authenticate it and the marker is never set on it. An admin arrives only through the
	// /prometheus and /victoriametrics locations, which already require an admin, and the
	// same endpoints are reachable directly under /prometheus without crossing this proxy,
	// so restricting admins here removes capability without removing any exposure.
	if isAdmin {
		return true
	}

	cleaned := normalizePath(p)

	if _, ok := readOnlyPaths[cleaned]; ok {
		return true
	}

	return labelValuesPath.MatchString(cleaned)
}

// normalizePath reduces the shapes the same endpoint arrives in to one. The nginx config
// passes the original URI for the /prometheus/api/v1 location and rewrites it for
// /victoriametrics/, and Grafana's data source sends it unprefixed. Cleaning first stops
// traversal walking out of a listed path.
func normalizePath(p string) string {
	cleaned := path.Clean(p)
	if cleaned == "/prometheus" || strings.HasPrefix(cleaned, "/prometheus/") {
		cleaned = strings.TrimPrefix(cleaned, "/prometheus")
	}

	return cleaned
}

// isAdminRequest reports whether pmm-managed authenticated this caller as an admin. The
// header is not a credential: nginx overwrites it on every location that can reach the
// proxy, so a client cannot supply one, and the proxy listens on loopback only.
func isAdminRequest(req *http.Request, headerName string) bool {
	if headerName == "" {
		return false
	}

	return req.Header.Get(headerName) == adminHeaderValue
}

// labelValuesPath matches /api/v1/label/<name>/values, which carries the label name as a
// path segment and so cannot be matched literally. The name is a single segment: it must be
// present and must not span a slash.
var labelValuesPath = regexp.MustCompile(`^/api/v1/label/[^/]+/values$`)

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
			io.WriteString(rw, fmt.Sprintf("Failed to parse %s header", headerName)) //nolint:errcheck,gosec
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
