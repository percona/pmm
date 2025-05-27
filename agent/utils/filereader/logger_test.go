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

package filereader

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

// check interface.
var _ Logger = (*testLogger)(nil)
