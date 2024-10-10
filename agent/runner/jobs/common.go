// Copyright (C) 2023 Percona LLC
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

package jobs

import (
	"net"
	"net/url"
	"strconv"
)

const maxLogsChunkSize = 50

// DBConnConfig contains required properties for connection to DB.
type DBConnConfig struct {
	User     string
	Password string
	Address  string
	Port     int
	Socket   string
}

func (c *DBConnConfig) createDBURL() *url.URL {
	var host string
	switch {
	case c.Address != "":
		if c.Port > 0 {
			host = net.JoinHostPort(c.Address, strconv.Itoa(c.Port))
		} else {
			host = c.Address
		}
	case c.Socket != "":
		host = c.Socket
	}

	var user *url.Userinfo
	if c.User != "" {
		user = url.UserPassword(c.User, c.Password)
	}

	return &url.URL{
		Scheme: "mongodb",
		User:   user,
		Host:   host,
	}
}
