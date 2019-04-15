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

// +build ignore

// Run it with:
//   go run -tags child process_child.go

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm-agent/agents/process"
)

func main() {
	flag.Parse()
	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)
	l := logrus.NewEntry(logger)

	p := process.New(&process.Params{Path: "sleep", Args: []string{"100500"}}, l)
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

	fmt.Println(process.GetPID(p)) // Printing PID of the child process to let test check if the child process is dead or not.
	time.Sleep(30 * time.Second)   // Waiting until test kills this process.
}
