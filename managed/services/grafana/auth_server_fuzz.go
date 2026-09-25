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

//go:build gofuzz
// +build gofuzz

// See https://github.com/dvyukov/go-fuzz

package grafana

import (
	"context"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

type clientStub struct{}

func (clientStub) getAuthUser(context.Context, http.Header, *logrus.Entry) (authUser, error) {
	return authUser{role: grafanaAdmin}, nil
}

func (clientStub) rotateSessionToken(context.Context, http.Header) ([]string, error) {
	return nil, nil
}

func Fuzz(data []byte) int {
	logrus.SetOutput(io.Discard)

	var c clientStub
	s := NewAuthServer(c, nil)

	req, err := http.NewRequest(http.MethodGet, string(data), nil)
	if err != nil {
		return 0
	}

	_, _, _ = s.authenticate(context.Background(), req, logrus.NewEntry(logrus.StandardLogger()))

	return 1
}
