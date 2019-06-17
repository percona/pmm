// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package parser

import "testing"

type testLogger struct {
	t testing.TB
}

func (tl *testLogger) Warnf(format string, v ...interface{}) {
	tl.t.Helper()
	tl.t.Logf("WARN : "+format, v...)
}

func (tl *testLogger) Infof(format string, v ...interface{}) {
	tl.t.Helper()
	tl.t.Logf("INFO : "+format, v...)
}

func (tl *testLogger) Debugf(format string, v ...interface{}) {
	tl.t.Helper()
	tl.t.Logf("DEBUG: "+format, v...)
}

func (tl *testLogger) Tracef(format string, v ...interface{}) {
	tl.t.Helper()
	tl.t.Logf("TRACE: "+format, v...)
}

// check interface
var _ Logger = (*testLogger)(nil)
