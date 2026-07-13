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

package models

import "time"

//go:generate go tool reform

// LogParserPreset represents a parser preset for OTEL filelog receivers (e.g. mysql_error).
// Presets define the operator YAML fragment used to parse log lines; they are stored in DB
// so custom presets can be added later via API.
//
//reform:log_parser_presets
type LogParserPreset struct {
	ID           string    `reform:"id,pk"`
	Name         string    `reform:"name"`
	Description  *string   `reform:"description"`
	OperatorYAML string    `reform:"operator_yaml"`
	BuiltIn      bool      `reform:"built_in"`
	CreatedAt    time.Time `reform:"created_at"`
	UpdatedAt    time.Time `reform:"updated_at"`
}
