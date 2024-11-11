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
	"time"

	"github.com/AlekSi/pointer"
	"github.com/percona/saas/pkg/alert"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
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
		return errors.WithStack(err)
	}

	return status.Errorf(codes.AlreadyExists, "Template with name %q already exists.", name)
}

// FindTemplates returns saved notification rule templates.
func FindTemplates(q *reform.Querier) ([]*Template, error) {
	structs, err := q.SelectAllFrom(TemplateTable, "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to select notification rule templates")
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
		return nil, errors.WithStack(err)
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
	if err := template.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid rule template: %v.", err)
	}

	if err := checkUniqueTemplateName(q, template.Name); err != nil {
		return nil, err
	}

	row, err := ConvertTemplate(template, params.Source)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Failed to convert template: %v.", err)
	}

	if err = q.Insert(row); err != nil {
		return nil, errors.Wrap(err, "failed to create rule template")
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
	if err := template.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid rule template: %v.", err)
	}

	yaml, err := alert.ToYAML([]alert.Template{*template})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	p, err := ConvertParamsDefinitions(params.Template.Params)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid rule template parameters: %v.", err)
	}

	row.Name = template.Name
	row.Version = template.Version
	row.Summary = template.Summary
	row.Expr = template.Expr
	row.Params = p
	row.For = time.Duration(template.For)
	row.Severity = Severity(template.Severity)
	row.Yaml = yaml

	if err = row.SetLabels(template.Labels); err != nil {
		return nil, err
	}

	if err = row.SetAnnotations(template.Annotations); err != nil {
		return nil, err
	}

	if err = q.Update(row); err != nil {
		return nil, errors.Wrap(err, "failed to update rule template")
	}

	return row, nil
}

// RemoveTemplate removes rule template with specified name.
func RemoveTemplate(q *reform.Querier, name string) error {
	_, err := FindTemplateByName(q, name)
	if err != nil {
		return err
	}

	if err = q.Delete(&Template{Name: name}); err != nil {
		return errors.Wrap(err, "failed to delete rule template")
	}
	return nil
}

// ConvertTemplate converts an alert template to the internal representation.
func ConvertTemplate(template *alert.Template, source Source) (*Template, error) {
	p, err := ConvertParamsDefinitions(template.Params)
	if err != nil {
		return nil, errors.Errorf("invalid rule template parameters: %v.", err) //nolint:revive
	}

	yaml, err := alert.ToYAML([]alert.Template{*template})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := &Template{
		Name:     template.Name,
		Version:  template.Version,
		Summary:  template.Summary,
		Expr:     template.Expr,
		Params:   p,
		For:      time.Duration(template.For),
		Severity: Severity(template.Severity),
		Source:   source,
		Yaml:     yaml,
	}

	if err := res.SetLabels(template.Labels); err != nil {
		return nil, err
	}

	if err := res.SetAnnotations(template.Annotations); err != nil {
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
					return nil, errors.Wrap(err, "failed to parse param value")
				}
				fp.Default = pointer.ToFloat64(def)
			}

			if len(param.Range) != 0 {
				min, max, err := param.GetRangeForFloat()
				if err != nil {
					return nil, errors.Wrap(err, "failed to parse param range")
				}
				fp.Min, fp.Max = pointer.ToFloat64(min), pointer.ToFloat64(max)
			}

			p.FloatParam = &fp
		default:
			return nil, errors.Errorf("unknown parameter type %s", param.Type)
		}

		res = append(res, p)
	}

	return res, nil
}
