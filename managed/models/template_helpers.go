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

import (
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/pi/alert"
)

func checkUniqueTemplateName(q *reform.Querier, name string) error {
	if name == "" {
		panic("empty template name")
	}

	template := &Template{Name: name}
	err := q.Reload(template)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return err
	}

	return status.Errorf(codes.AlreadyExists, "Template with name %q already exists.", name)
}

// FindTemplates returns saved notification rule templates.
func FindTemplates(q *reform.Querier) ([]*Template, error) {
	structs, err := q.SelectAllFrom(TemplateTable, "")
	if err != nil {
		return nil, fmt.Errorf("failed to select notification rule templates: %w", err)
	}

	templates := make([]*Template, len(structs))
	for i, s := range structs {
		templates[i] = s.(*Template) //nolint:forcetypeassert
	}

	return templates, nil
}

// FindTemplateByName finds template by name.
func FindTemplateByName(q *reform.Querier, name string) (*Template, error) {
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty template name.")
	}

	template := &Template{Name: name}
	err := q.Reload(template)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "Template with name %q not found.", name)
		}
		return nil, err
	}

	return template, nil
}

// CreateTemplateParams are params for creating new rule template.
type CreateTemplateParams struct {
	Template *alert.Template
	Source   Source
}

// CreateTemplate creates rule template.
func CreateTemplate(q *reform.Querier, params *CreateTemplateParams) (*Template, error) {
	template := params.Template
	err := template.Validate()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid rule template: %v.", err)
	}

	err = checkUniqueTemplateName(q, template.Name)
	if err != nil {
		return nil, err
	}

	row, err := ConvertTemplate(template, params.Source)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Failed to convert template: %v.", err)
	}

	err = q.Insert(row)
	if err != nil {
		return nil, fmt.Errorf("failed to create rule template: %w", err)
	}

	return row, nil
}

// ChangeTemplateParams is params for changing existing rule template.
type ChangeTemplateParams struct {
	Template *alert.Template
	Name     string
}

// ChangeTemplate updates existing rule template.
func ChangeTemplate(q *reform.Querier, params *ChangeTemplateParams) (*Template, error) {
	if params.Name != params.Template.Name {
		return nil, status.Errorf(codes.InvalidArgument, "Mismatch names.")
	}

	row, err := FindTemplateByName(q, params.Name)
	if err != nil {
		return nil, err
	}

	template := params.Template
	err = template.Validate()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid rule template: %v.", err)
	}

	yaml, err := alert.ToYAML([]alert.Template{*template})
	if err != nil {
		return nil, err
	}

	p, err := ConvertParamsDefinitions(params.Template.Params)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid rule template parameters: %v.", err)
	}

	row.Name = template.Name
	row.Version = template.Version
	row.Summary = template.Summary
	row.Expr = template.StoredExpr()
	row.Params = p
	row.For = time.Duration(template.For)
	row.Severity = Severity(template.Severity)
	row.Yaml = yaml

	err = row.SetLabels(template.Labels)
	if err != nil {
		return nil, err
	}

	err = row.SetAnnotations(template.Annotations)
	if err != nil {
		return nil, err
	}

	err = q.Update(row)
	if err != nil {
		return nil, fmt.Errorf("failed to update rule template: %w", err)
	}

	return row, nil
}

// RemoveTemplate removes rule template with specified name.
func RemoveTemplate(q *reform.Querier, name string) error {
	_, err := FindTemplateByName(q, name)
	if err != nil {
		return err
	}

	err = q.Delete(&Template{Name: name})
	if err != nil {
		return fmt.Errorf("failed to delete rule template: %w", err)
	}
	return nil
}

// ConvertTemplate converts an alert template to the internal representation.
func ConvertTemplate(template *alert.Template, source Source) (*Template, error) {
	p, err := ConvertParamsDefinitions(template.Params)
	if err != nil {
		return nil, fmt.Errorf("invalid rule template parameters: %w", err)
	}

	yaml, err := alert.ToYAML([]alert.Template{*template})
	if err != nil {
		return nil, err
	}

	res := &Template{
		Name:     template.Name,
		Version:  template.Version,
		Summary:  template.Summary,
		Expr:     template.StoredExpr(),
		Params:   p,
		For:      time.Duration(template.For),
		Severity: Severity(template.Severity),
		Source:   source,
		Yaml:     yaml,
	}

	err = res.SetLabels(template.Labels)
	if err != nil {
		return nil, err
	}

	err = res.SetAnnotations(template.Annotations)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ConvertParamsDefinitions converts parameters definitions to the model.
func ConvertParamsDefinitions(params []alert.Parameter) (AlertExprParamsDefinitions, error) {
	res := make(AlertExprParamsDefinitions, 0, len(params))
	for _, param := range params {
		p := AlertExprParamDefinition{
			Name:    param.Name,
			Summary: param.Summary,
			Unit:    ParamUnit(param.Unit),
			Type:    ParamType(param.Type),
		}

		switch param.Type {
		case alert.Float:
			var fp FloatParam
			if param.Value != nil {
				def, err := param.GetValueForFloat()
				if err != nil {
					return nil, fmt.Errorf("failed to parse param value: %w", err)
				}
				fp.Default = new(def)
			}

			if len(param.Range) != 0 {
				pMin, pMax, err := param.GetRangeForFloat()
				if err != nil {
					return nil, fmt.Errorf("failed to parse param range: %w", err)
				}
				fp.Min, fp.Max = new(pMin), new(pMax)
			}

			p.FloatParam = &fp
		default:
			return nil, fmt.Errorf("unknown parameter type %s", param.Type)
		}

		res = append(res, p)
	}

	return res, nil
}
