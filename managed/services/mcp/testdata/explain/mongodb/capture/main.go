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

// Command capture prints what pmm-agent's MongoDB explain action returns for a QAN query example.
//
// Usage: capture <dsn> <example.json>
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/percona/pmm/agent/runner/actions"
	agentv1 "github.com/percona/pmm/api/agent/v1"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: capture <dsn> <example.json>")
		os.Exit(2)
	}
	out, err := capture(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}

// capture runs the explain action on the example in file.
func capture(dsn, file string) ([]byte, error) {
	example, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	params := &agentv1.StartActionRequest_MongoDBExplainParams{Dsn: dsn, Query: strings.TrimSpace(string(example))}
	action, err := actions.NewMongoDBExplainAction("capture", time.Minute, params, os.TempDir())
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return action.Run(ctx)
}
