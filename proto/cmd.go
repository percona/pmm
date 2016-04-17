/*
   Copyright (c) 2016, Percona LLC and/or its affiliates. All rights reserved.

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package proto

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// Sent by API to agent
type Cmd struct {
	Id        string `json:",omitempty"` // set by API
	Ts        time.Time
	User      string
	AgentUUID string
	Service   string
	Cmd       string
	Data      []byte `json:",omitempty"`
}

// Sent by agent in response to every command
type Reply struct {
	Id    string // set by API
	Cmd   string // original Cmd.Cmd
	Error string // success if empty
	Data  []byte `json:",omitempty"`
}

// Data for StartService and StopService command replies
type ServiceData struct {
	Name   string
	Config []byte `json:",omitempty"` // cloud-tools/<service>/config.go
}

// Reply from agent to Version command.  The two can differ is agent self-update
// but hasn't be restarted yet.
type Version struct {
	Installed string
	Running   string
	Revision  string
}

func (cmd *Cmd) Reply(data interface{}, errs ...error) *Reply {
	reply := &Reply{
		Id:  cmd.Id,
		Cmd: cmd.Cmd,
	}
	if len(errs) > 0 {
		errmsgs := make([]string, len(errs))
		for i, err := range errs {
			if err == nil {
				continue
			}
			errmsgs[i] = err.Error()
		}
		reply.Error = strings.Join(errmsgs, "\n")
	}
	if data != nil {
		codedData, jsonErr := json.Marshal(data)
		if jsonErr != nil {
			log.Fatal(jsonErr) // shouldn't happen
		}
		reply.Data = codedData
	}
	return reply
}

func (cmd *Cmd) String() string {
	cmdx := *cmd
	cmdx.Data = nil
	return fmt.Sprintf("Cmd[Service:%s Cmd:%s Ts:'%s' User:%s AgentUUID:%s Id:%s]",
		cmdx.Service, cmdx.Cmd,
		cmdx.Ts, cmdx.User, cmd.AgentUUID,
		cmdx.Id)
}

func (reply *Reply) String() string {
	replyx := *reply
	replyx.Data = nil
	return fmt.Sprintf("Reply[Cmd:%s Error:'%s' Id:%s]", replyx.Cmd, replyx.Error, replyx.Id)
}
