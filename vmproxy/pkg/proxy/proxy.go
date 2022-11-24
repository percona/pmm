// Copyright 2019 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package proxy provides http reverse proxy functionality
package proxy

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/sirupsen/logrus"
)

// Config defines options for starting proxy
type Config struct {
	// Name of the header to check for filters. Case insensitive.
	HeaderName string
	// Address the proxy is listening on
	ListenAddress string
	// Target URL to forward requests to
	TargetURL *url.URL
}

// StartProxy starts proxy which adds extra filters based on configuration.
func StartProxy(cfg Config) {
	logrus.Infof("Starting to proxy at http://%s to %s", cfg.ListenAddress, cfg.TargetURL.String())

	err := http.ListenAndServe(cfg.ListenAddress, getHandler(cfg))
	logrus.Error(err)
}

func getHandler(cfg Config) http.HandlerFunc {
	rProxy := &httputil.ReverseProxy{
		Director: director(cfg.TargetURL, cfg.HeaderName),
	}

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if failOnInvalidHeader(rw, req, cfg.HeaderName) {
			return
		}

		rProxy.ServeHTTP(rw, req)
	})
}

func failOnInvalidHeader(rw http.ResponseWriter, req *http.Request, headerName string) bool {
	if filters := req.Header.Get(headerName); filters != "" {
		if len(parseFilters(filters)) == 0 {
			rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
			rw.WriteHeader(http.StatusPreconditionFailed)
			io.WriteString(rw, "Failed to parse headers")
			return true
		}
	}

	return false
}

func director(target *url.URL, headerName string) func(*http.Request) {
	return func(req *http.Request) {
		logrus.Infof("%s: %s", req.Method, req.URL)
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host

		logrus.Infof("Pre-filters: %s", req.Header.Get(headerName))
		// Replace extra filters if present
		if filters := req.Header.Get(headerName); filters != "" {
			q := req.URL.Query()
			q.Del("extra_filters")

			for _, f := range parseFilters(filters) {
				q.Add("extra_filters", f)
			}

			logrus.Infof("filters: %#v", filters)
			req.URL.RawQuery = q.Encode()
		}

		// Do not trust the client
		req.Header.Del("X-Forwarded-For")

		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}
}

func parseFilters(filters string) []string {
	var parsed []string

	decoded, err := base64.StdEncoding.DecodeString(filters)
	if err != nil {
		logrus.Errorf("could not decode filters header. %v", err)
		return parsed
	}

	if err := json.Unmarshal(decoded, &parsed); err != nil {
		logrus.Errorf("could not parse filters JSON. %v", err)
		return parsed
	}

	return parsed
}
