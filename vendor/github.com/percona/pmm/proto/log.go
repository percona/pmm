/*
   Copyright (c) 2016, Percona LLC and/or its affiliates. All rights reserved.

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package proto

import (
	"encoding/json"
	"time"
)

// http://en.wikipedia.org/wiki/Syslog#Severity_levels
const (
	LOG_EMERGENCY byte = iota // not used
	LOG_ALERT                 // not used
	LOG_CRITICAL              // not used
	LOG_ERROR
	LOG_WARNING
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)

var LogLevelNumber map[string]byte = map[string]byte{
	"emergency": LOG_EMERGENCY,
	"alert":     LOG_ALERT,
	"critical":  LOG_CRITICAL,
	"error":     LOG_ERROR,
	"warning":   LOG_WARNING,
	"notice":    LOG_NOTICE,
	"info":      LOG_INFO,
	"debug":     LOG_DEBUG,
}

var LogLevelName []string = []string{
	"emergency",
	"alert",
	"critical",
	"error",
	"warning",
	"notice",
	"info",
	"debug",
}

type LogEntry struct {
	Ts      time.Time
	Level   byte
	Service string
	Msg     string
	Offline bool `json:"-"`
}

func (e *LogEntry) String() string {
	bytes, err := json.Marshal(e)
	if err != nil {
		return ""
	}
	return string(bytes)
}
