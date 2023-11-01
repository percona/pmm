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

// Package auth contains functions to work with auth tokens and headers.
package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

// GetTokenFromHeaders returns authorization token if it is found in provided HTTP headers.
func GetTokenFromHeaders(authHeaders http.Header) string {
	authHeader := authHeaders.Get("Authorization")
	switch {
	case strings.HasPrefix(authHeader, "Bearer"):
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
	case strings.HasPrefix(authHeader, "Basic"):
		h := strings.TrimPrefix(authHeader, "Basic")
		t, err := base64.StdEncoding.DecodeString(strings.TrimSpace(h))
		if err != nil {
			return ""
		}
		tk := string(t)
		if strings.HasPrefix(tk, "api_key:") || strings.HasPrefix(tk, "service_token:") {
			return strings.Split(tk, ":")[1]
		}
	}

	return ""
}

// GetHeadersFromContext returns authorization headers if they are found in provided context.
func GetHeadersFromContext(ctx context.Context) (http.Header, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("cannot get headers from metadata")
	}
	// get authorization from headers.
	authorizationHeaders := headers.Get("Authorization")
	cookieHeaders := headers.Get("grpcgateway-cookie")
	if len(authorizationHeaders) == 0 && len(cookieHeaders) == 0 {
		return nil, status.Error(codes.Unauthenticated, "Authorization error.")
	}

	authHeaders := make(http.Header)
	if len(authorizationHeaders) != 0 {
		authHeaders.Add("Authorization", authorizationHeaders[0])
	}
	if len(cookieHeaders) != 0 {
		for _, header := range cookieHeaders {
			authHeaders.Add("Cookie", header)
		}
	}
	return authHeaders, nil
}
