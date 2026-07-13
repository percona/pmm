// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v3"
)

var logParserPresetNameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

var logParserRegexFieldLineRe = regexp.MustCompile(`(?m)^(\s*regex:\s+)([^'"\n][^\n]*)$`)

var logParserUnquotedRegexRunOnRe = regexp.MustCompile(`(?m)^(\s*regex:\s+)(.+?\$\s*)parse_from:`)

// ValidateLogParserPresetName checks name matches OtelCollector receiver id rules (alphanumeric + underscore).
func ValidateLogParserPresetName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("empty preset name")
	}
	if !logParserPresetNameRe.MatchString(name) {
		return fmt.Errorf("preset name %q must match %s", name, logParserPresetNameRe.String())
	}
	return nil
}

// NormalizeLogParserOperatorYAML fixes common copy/paste issues before validation.
// Keep in sync with ui/apps/pmm/src/api/logParserPresets.ts normalizeOperatorYaml.
func NormalizeLogParserOperatorYAML(operatorYAML string) string {
	s := strings.ReplaceAll(operatorYAML, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.NewReplacer(
		"\u2018", "'", "\u2019", "'",
		"\u201c", "\"", "\u201d", "\"",
	).Replace(s)
	// Pasted from JSON string literals (literal \n instead of newlines).
	if !strings.Contains(s, "\n") && strings.Contains(s, `\n`) {
		s = strings.ReplaceAll(s, `\n`, "\n")
	}
	// Regex value run into the next field on the same line (common when copying presets).
	s = strings.ReplaceAll(s, "' parse_from:", "'\n  parse_from:")
	s = strings.ReplaceAll(s, "' parse_to:", "'\n  parse_to:")
	s = strings.ReplaceAll(s, "' - type:", "'\n- type:")
	s = strings.ReplaceAll(s, "\" parse_from:", "\"\n  parse_from:")
	s = strings.ReplaceAll(s, "\" parse_to:", "\"\n  parse_to:")
	s = strings.ReplaceAll(s, "\" - type:", "\"\n- type:")
	s = logParserUnquotedRegexRunOnRe.ReplaceAllString(s, "$1'$2'\n  parse_from:")
	s = quoteUnquotedRegexLines(s)
	s = dedentOperatorYAML(s)
	return strings.TrimSpace(s)
}

// dedentOperatorYAML removes common leading indentation (TrimSpace only strips the first line).
func dedentOperatorYAML(s string) string {
	lines := strings.Split(s, "\n")
	min := -1 //nolint:predeclared,revive
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if min == -1 || indent < min {
			min = indent //nolint:revive
		}
	}
	if min <= 0 {
		return s
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			out = append(out, "")
			continue
		}
		if len(line) >= min {
			out = append(out, line[min:])
			continue
		}
		out = append(out, strings.TrimLeft(line, " \t"))
	}
	return strings.Join(out, "\n")
}

func quoteUnquotedRegexLines(s string) string {
	return logParserRegexFieldLineRe.ReplaceAllStringFunc(s, func(line string) string {
		sub := logParserRegexFieldLineRe.FindStringSubmatch(line)
		if len(sub) != 3 { //nolint:mnd
			return line
		}
		prefix, value := sub[1], strings.TrimSpace(sub[2])
		if value == "" || value[0] == '\'' || value[0] == '"' {
			return line
		}
		if strings.Contains(value, ":") {
			return prefix + "'" + value + "'"
		}
		return line
	})
}

func normalizeAndValidateLogParserOperatorYAML(operatorYAML string) (string, error) {
	operatorYAML = NormalizeLogParserOperatorYAML(operatorYAML)
	if operatorYAML == "" {
		return "", errors.New("operator_yaml is required")
	}
	var ops []map[string]any
	err := yaml.Unmarshal([]byte(operatorYAML), &ops)
	if err != nil {
		hint := operatorYAMLValidationHint(err)
		if hint != "" {
			return "", fmt.Errorf("operator_yaml must be a YAML array of operator objects: %w; %s", err, hint)
		}
		return "", fmt.Errorf("operator_yaml must be a YAML array of operator objects: %w", err)
	}
	if len(ops) == 0 {
		return "", errors.New("operator_yaml must contain at least one operator")
	}
	for i, op := range ops {
		t, ok := op["type"].(string)
		if !ok || strings.TrimSpace(t) == "" {
			return "", fmt.Errorf("operator %d: missing or invalid type", i)
		}
	}
	return operatorYAML, nil
}

// ValidateLogParserOperatorYAML ensures operator_yaml is a non-empty YAML sequence of objects with a type field.
func ValidateLogParserOperatorYAML(operatorYAML string) error {
	_, err := normalizeAndValidateLogParserOperatorYAML(operatorYAML)
	return err
}

func operatorYAMLValidationHint(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "mapping values are not allowed"):
		return "quote regex values that contain colons and put each field (parse_from, parse_to, etc.) on its own line"
	case strings.Contains(msg, "found unexpected end of stream"):
		return "check that quoted regex values are closed before the next field"
	default:
		return ""
	}
}

// ListOtelCollectorAgentIDsReferencingLogParserPreset returns agent_ids of OTEL collectors using presetName in log_sources.
func ListOtelCollectorAgentIDsReferencingLogParserPreset(q *reform.Querier, presetName string) ([]string, error) {
	presetName = strings.TrimSpace(presetName)
	if presetName == "" {
		return nil, errors.New("empty preset name")
	}
	t := OtelCollectorType
	agents, err := FindAgents(q, AgentFilters{AgentType: &t})
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, a := range agents {
		labels, err := a.GetCustomLabels()
		if err != nil {
			return nil, err
		}
		entries, err := ParseOtelLogSourcesFromLabels(labels)
		if err != nil {
			continue
		}
		for _, e := range entries {
			p := e.Preset
			if p == "" {
				p = otelPresetRaw
			}
			if p == presetName {
				ids = append(ids, a.AgentID)
				break
			}
		}
	}
	return ids, nil
}

// CreateLogParserPreset inserts a custom preset (built_in=false).
func CreateLogParserPreset(q *reform.Querier, name, description, operatorYAML string) (*LogParserPreset, error) {
	if err := ValidateLogParserPresetName(name); err != nil { //nolint:noinlineerr
		return nil, err
	}
	operatorYAML, err := normalizeAndValidateLogParserOperatorYAML(operatorYAML)
	if err != nil {
		return nil, err
	}
	existing, err := FindLogParserPresetByName(q, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("log parser preset %q already exists", name)
	}
	now := time.Now()
	var descPtr *string
	if strings.TrimSpace(description) != "" {
		d := strings.TrimSpace(description)
		descPtr = &d
	}
	row := &LogParserPreset{
		ID:           uuid.NewString(),
		Name:         name,
		Description:  descPtr,
		OperatorYAML: operatorYAML,
		BuiltIn:      false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := q.Insert(row); err != nil { //nolint:noinlineerr
		return nil, err
	}
	return row, nil
}

// UpdateLogParserPreset updates description and/or operator_yaml. Name and built_in are not changed.
func UpdateLogParserPreset(q *reform.Querier, id string, description *string, operatorYAML *string) (*LogParserPreset, error) {
	if id == "" {
		return nil, errors.New("empty preset id")
	}
	row, err := FindLogParserPresetByID(q, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, reform.ErrNoRows
	}
	if operatorYAML != nil {
		normalized, err := normalizeAndValidateLogParserOperatorYAML(*operatorYAML)
		if err != nil {
			return nil, err
		}
		row.OperatorYAML = normalized
	}
	if description != nil {
		d := strings.TrimSpace(*description)
		if d == "" {
			row.Description = nil
		} else {
			row.Description = &d
		}
	}
	row.UpdatedAt = time.Now()
	if err := q.Update(row); err != nil { //nolint:noinlineerr
		return nil, err
	}
	return row, nil
}

// DeleteLogParserPreset removes a non-built-in preset. Caller must verify no references.
func DeleteLogParserPreset(q *reform.Querier, id string) error {
	if id == "" {
		return errors.New("empty preset id")
	}
	row, err := FindLogParserPresetByID(q, id)
	if err != nil {
		return err
	}
	if row == nil {
		return reform.ErrNoRows
	}
	if row.BuiltIn {
		return errors.New("cannot delete built-in log parser preset")
	}
	return q.Delete(row)
}
