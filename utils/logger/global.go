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

package logger

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
)

// SetupGlobalLogger configures logrus.StandardLogger() to enable multiline-friendly formatter
// in both development (with terminal) and production (without terminal) with default prettyfier.
func SetupGlobalLogger() {
	logrus.SetFormatter(&logrus.TextFormatter{
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
	})
}
