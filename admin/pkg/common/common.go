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

// Package common holds common methods used in admin.
package common

import (
	"os/exec"

	"github.com/pkg/errors"
)

// LookupCommand tries to find command and returns its path if found.
func LookupCommand(cmd string) (string, error) {
	path, err := exec.LookPath(cmd)
	if err != nil {
		var execError *exec.Error
		if ok := errors.As(err, &execError); ok {
			if ok := errors.As(execError, &exec.ErrNotFound); ok {
				return "", nil
			}
		}
		return "", err
	}

	return path, nil
}
