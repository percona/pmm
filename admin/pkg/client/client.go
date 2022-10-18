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

// Package client provides common methods for working with pmm-client
package client

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// ErrLatestVersionNotFound is returned when we cannot determine what the latest version is.
var ErrLatestVersionNotFound = fmt.Errorf("LatestVersionNotFound")

// GetLatestVersion retrieves latest version of pmm-client available.
func GetLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://github.com/percona/pmm/releases/latest", nil)
	if err != nil {
		return "", err
	}

	cl := &http.Client{ //nolint:exhaustruct
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 10 * time.Second,
	}
	res, err := cl.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close() //nolint:errcheck

	url, err := res.Location()
	if err != nil {
		logrus.Debug(err)
		return "", fmt.Errorf("%w: could not find latest version", ErrLatestVersionNotFound)
	}

	tag := path.Base(url.Path)
	latest := strings.TrimPrefix(tag, "v")

	return latest, nil
}
