// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package agents

import (
	"github.com/AlekSi/pointer"

	"github.com/percona/pmm-managed/models"
)

type redactMode int

const (
	redactSecrets redactMode = iota
	exposeSecrets
)

// redactWords returns words that should be redacted from given Agent logs/output.
func redactWords(agent *models.Agent) []string {
	var words []string
	if s := pointer.GetString(agent.Password); s != "" {
		words = append(words, s)
	}
	if s := pointer.GetString(agent.AgentPassword); s != "" {
		words = append(words, s)
	}
	if s := pointer.GetString(agent.AWSSecretKey); s != "" {
		words = append(words, s)
	}
	if agent.AzureOptions != nil {
		if s := agent.AzureOptions.ClientSecret; s != "" {
			words = append(words, s)
		}
	}
	return words
}
