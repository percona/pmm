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

package commands

import (
	"fmt"

	"gopkg.in/alecthomas/kingpin.v2"
)

type configCommand struct{}

func (cmd *configCommand) Run() (Result, error) {
	return nil, fmt.Errorf("Please use `pmm-agent setup`.") //nolint:stylecheck,golint
}

// register command
var (
	Config  = new(configCommand)
	ConfigC = kingpin.Command("config", "Deprecated, please use `pmm-agent setup` instead.").Hidden()
)
