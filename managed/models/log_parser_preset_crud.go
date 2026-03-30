// Copyright (C) 2026 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package models

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v3"
)

var logParserPresetNameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// ValidateLogParserPresetName checks name matches OtelCollector receiver id rules (alphanumeric + underscore).
func ValidateLogParserPresetName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("empty preset name")
	}
	if !logParserPresetNameRe.MatchString(name) {
		return errors.Errorf("preset name %q must match %s", name, logParserPresetNameRe.String())
	}
	return nil
}

// ValidateLogParserOperatorYAML ensures operator_yaml is a non-empty YAML sequence of objects with a type field.
func ValidateLogParserOperatorYAML(operatorYAML string) error {
	operatorYAML = strings.TrimSpace(operatorYAML)
	if operatorYAML == "" {
		return errors.New("operator_yaml is required")
	}
	var ops []map[string]interface{}
	if err := yaml.Unmarshal([]byte(operatorYAML), &ops); err != nil {
		return errors.Wrap(err, "operator_yaml must be a YAML array of operator objects")
	}
	if len(ops) == 0 {
		return errors.New("operator_yaml must contain at least one operator")
	}
	for i, op := range ops {
		t, ok := op["type"].(string)
		if !ok || strings.TrimSpace(t) == "" {
			return errors.Errorf("operator %d: missing or invalid type", i)
		}
	}
	return nil
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
			return nil, errors.WithStack(err)
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
	if err := ValidateLogParserPresetName(name); err != nil {
		return nil, err
	}
	if err := ValidateLogParserOperatorYAML(operatorYAML); err != nil {
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
	if err := q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
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
		if err := ValidateLogParserOperatorYAML(*operatorYAML); err != nil {
			return nil, err
		}
		row.OperatorYAML = strings.TrimSpace(*operatorYAML)
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
	if err := q.Update(row); err != nil {
		return nil, errors.WithStack(err)
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
	return errors.WithStack(q.Delete(row))
}
