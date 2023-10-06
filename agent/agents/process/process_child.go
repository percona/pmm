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

//go:build ignore
// +build ignore

// Run it with:
//   go run -tags child process_child.go

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/agent/agents/process"
	"github.com/percona/pmm/api/inventorypb"
)

func main() {
	flag.Parse()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	l := logrus.NewEntry(logger)

	p := process.New(&process.Params{Path: "sleep", Args: []string{"100500"}}, nil, l)
	go p.Run(context.Background())

	// Wait until the process is running.
	state := <-p.Changes()
	if state != inventorypb.AgentStatus_STARTING {
		panic("process isn't moved to starting state.")
	}
	state = <-p.Changes()
	if state != inventorypb.AgentStatus_RUNNING {
		panic("process isn't moved to running state.")
	}

	// Printing PID of the child process to let test check if the child process is dead or not.
	fmt.Println(process.GetPID(p)) //nolint:forbidigo
	time.Sleep(30 * time.Second)   // Waiting until test kills this process.
}
