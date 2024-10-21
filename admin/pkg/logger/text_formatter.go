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

// Package logger provides helpers for logger.
package logger

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// TextFormatter formats logs into text.
type TextFormatter struct{}

// Format renders a single log entry.
func (f *TextFormatter) Format(entry *logrus.Entry) ([]byte, error) { //nolint:unparam
	b := &bytes.Buffer{}
	if entry.Buffer != nil {
		b = entry.Buffer
	}

	// Remove a single newline if it already exists in the message to keep
	// the behavior of logrus text_formatter the same as the stdlib log package
	entry.Message = strings.TrimSuffix(entry.Message, "\n")

	caller := ""

	if entry.HasCaller() {
		funcVal := fmt.Sprintf("%s()", entry.Caller.Function)
		fileVal := fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)
		caller = fileVal + " " + funcVal + " "
	}

	if entry.Level >= logrus.DebugLevel {
		now := time.Now().UTC().Format("2006-01-02 15:04:05.999999999Z")
		fmt.Fprintf(b, "%s %s: ", strings.ToUpper(entry.Level.String()), now)
	}
	fmt.Fprintf(b, "%s%-44s\n", caller, entry.Message)

	return b.Bytes(), nil
}
