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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
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
	// Optional Host header value to set in the request
	HostHeader string
}

// RunProxy starts proxy which adds extra filters based on configuration.
func RunProxy(cfg Config) error {
	logrus.Infof("Starting to proxy at http://%s to %s", cfg.ListenAddress, cfg.TargetURL.String())

	err := http.ListenAndServe(cfg.ListenAddress, getHandler(cfg)) //nolint:gosec
	return err
}

func getHandler(cfg Config) http.HandlerFunc {
	rProxy := &httputil.ReverseProxy{
		Director: director(cfg.TargetURL, cfg.HeaderName, strings.TrimSpace(cfg.HostHeader)),
	}

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		logrus.Infof("%s: %s", req.Method, req.URL)

		if failOnInvalidHeader(rw, req, cfg.HeaderName) {
			return
		}

		rProxy.ServeHTTP(rw, req)
	})
}

func failOnInvalidHeader(rw http.ResponseWriter, req *http.Request, headerName string) bool {
	if filters := req.Header.Get(headerName); filters != "" {
		if _, err := parseFilters(filters); err != nil {
			rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
			rw.WriteHeader(http.StatusPreconditionFailed)
			io.WriteString(rw, fmt.Sprintf("Failed to parse %s header", headerName)) //nolint:errcheck
			return true
		}
	}

	return false
}

func prepareRequest(req *http.Request, target *url.URL, headerName string, hostHeader string) {
	now := time.Now()

	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host

	// Update or add Host header if hostHeader is provided
	if hostHeader != "" {
		req.Host = hostHeader
		req.Header.Set("Host", hostHeader)
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
			"Received filters: %s, Parsed filters: %#v, Query: %s, Target URL: %s, Time spent: %s",
			filters, parsed, req.URL.RawQuery, req.URL, time.Since(now))
	}

	// Do not trust the client
	req.Header.Del("X-Forwarded-For")

	if _, ok := req.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		req.Header.Set("User-Agent", "")
	}
}

func director(target *url.URL, headerName string, hostHeader string) func(*http.Request) {
	return func(req *http.Request) {
		prepareRequest(req, target, headerName, hostHeader)
	}
}

func parseFilters(filters string) ([]string, error) {
	var parsed []string

	decoded, err := base64.StdEncoding.DecodeString(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not decode filters header")
	}

	if err := json.Unmarshal(decoded, &parsed); err != nil {
		return nil, errors.Wrapf(err, "could not parse filters JSON")
	}

	return parsed, nil
}
