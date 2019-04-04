// pmm-admin
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

// Package commands provides base commands and helpers.
package commands

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"text/template"

	"github.com/sirupsen/logrus"
)

// Ctx is a shared context for all requests.
var Ctx = context.Background()

// Result is a common interface for all command results.
//
// In addition to methods of this interface, result is expected to work with json.Marshal.
type Result interface {
	Result()
	fmt.Stringer
}

// Command is a common interface for all commands.
//
// Command should:
//  * use logrus.Debug/Trace functions for debug logging;
//  * return result on success;
//  * return error on failure.
//
// Command should not:
//  * exit with logrus.Fatal, os.Exit, etc.
type Command interface {
	Run() (Result, error)
}

type ErrorResponse interface {
	error
	Code() int
}

type Error struct {
	Code  int
	Error string
}

func GetError(err ErrorResponse) Error {
	v := reflect.ValueOf(err)
	p := v.Elem().FieldByName("Payload")
	e := p.Elem().FieldByName("Error")
	return Error{
		Code:  err.Code(),
		Error: e.String(),
	}
}

// RenderTemplate renders given template with given data and returns result as string.
func RenderTemplate(t *template.Template, data interface{}) string {
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		logrus.Fatal(err)
	}
	return buf.String()
}
