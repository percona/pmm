// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
)

// GetLoggerFormatter returns instance of TextFormatter with predefined parameters.
func GetLoggerFormatter() *logrus.TextFormatter {
	return &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02T15:04:05.000-07:00",
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			_, function := filepath.Split(f.Function)

			// keep a single directory name as a compromise between brevity and unambiguity
			var dir string
			dir, file := filepath.Split(f.File)
			dir = filepath.Base(dir)
			file = fmt.Sprintf("%s/%s:%d", dir, file, f.Line)

			return function, file
		},
	}
}

// SetupGlobalLogger configures logrus.StandardLogger() to enable multiline-friendly formatter
// in both development (with terminal) and production (without terminal) with default prettyfier.
func SetupGlobalLogger() {
	logrus.SetFormatter(GetLoggerFormatter())
}
