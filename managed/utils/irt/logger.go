// Copyright (C) 2024 Percona LLC
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

package irt

import (
	"net/http"
	"net/http/httputil"
)

// WithLogger returns http.RoundTripper with request/response logger.
func WithLogger(t http.RoundTripper, printf func(format string, v ...interface{})) http.RoundTripper {
	return &loggerRoundTripper{
		t:      t,
		printf: printf,
	}
}

type loggerRoundTripper struct {
	t      http.RoundTripper
	printf func(format string, v ...interface{})
}

func (lrt *loggerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	b, _ := httputil.DumpRequestOut(req, true)
	if len(b) != 0 {
		lrt.printf("Request:\n%s", b)
	}

	resp, err := lrt.t.RoundTrip(req)

	if resp != nil {
		b, _ = httputil.DumpResponse(resp, true)
		if len(b) != 0 {
			lrt.printf("Response:\n%s", b)
		}
	}

	return resp, err
}
