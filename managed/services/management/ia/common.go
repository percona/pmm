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

package ia

import (
	"bytes"
	"os"
	"text/template"

	"github.com/AlekSi/pointer"
	"github.com/percona-platform/saas/pkg/alert"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/api/managementpb"
	alerting "github.com/percona/pmm/api/managementpb/alerting"
	iav1beta1 "github.com/percona/pmm/api/managementpb/ia"
	"github.com/percona/pmm/managed/models"
)

const (
	dirPerm = os.FileMode(0o775)
)

func convertParamUnit(u alert.Unit) alerting.ParamUnit {
	switch u {
	case alert.Percentage:
		return alerting.ParamUnit_PERCENTAGE
	case alert.Seconds:
		return alerting.ParamUnit_SECONDS
	}

	// do not add `default:` to make exhaustive linter do its job

	return alerting.ParamUnit_PARAM_UNIT_INVALID
}

func convertRule(l *logrus.Entry, rule *models.Rule, channels []*models.Channel) (*iav1beta1.Rule, error) { //nolint:staticcheck
	r := &iav1beta1.Rule{ //nolint:staticcheck
		RuleId:          rule.ID,
		TemplateName:    rule.TemplateName,
		Disabled:        rule.Disabled,
		Summary:         rule.Summary,
		Name:            rule.Name,
		ExprTemplate:    rule.ExprTemplate,
		DefaultSeverity: managementpb.Severity(rule.DefaultSeverity),
		Severity:        managementpb.Severity(rule.Severity),
		DefaultFor:      durationpb.New(rule.DefaultFor),
		For:             durationpb.New(rule.For),
		CreatedAt:       timestamppb.New(rule.CreatedAt),
	}

	var err error
	if err = r.CreatedAt.CheckValid(); err != nil {
		return nil, errors.Wrap(err, "failed to convert timestamp")
	}

	r.Expr, err = fillExprWithParams(rule.ExprTemplate, rule.ParamsValues.AsStringMap())
	if err != nil {
		return nil, errors.Wrap(err, "failed to fill expression template with parameters values")
	}

	r.ParamsDefinitions, err = convertModelToParamsDefinitions(rule.ParamsDefinitions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert parameters definitions")
	}

	r.ParamsValues, err = convertModelToParamValues(rule.ParamsValues)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert parameters values")
	}

	r.CustomLabels, err = rule.GetCustomLabels()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load rule custom labels")
	}

	r.Labels, err = rule.GetLabels()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load rule labels")
	}

	r.Annotations, err = rule.GetAnnotations()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load rule annotations")
	}

	r.Filters = make([]*iav1beta1.Filter, len(rule.Filters)) //nolint:staticcheck
	for i, filter := range rule.Filters {
		r.Filters[i] = &iav1beta1.Filter{ //nolint:staticcheck
			Type:  convertModelToFilterType(filter.Type),
			Key:   filter.Key,
			Value: filter.Val,
		}
	}

	cm := make(map[string]*models.Channel)
	for _, channel := range channels {
		cm[channel.ID] = channel
	}

	for _, id := range rule.ChannelIDs {
		channel, ok := cm[id]
		if !ok {
			l.Warningf("Skip missing channel with ID %s", id)
			continue
		}

		c, err := convertChannel(channel)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert channel")
		}
		r.Channels = append(r.Channels, c)
	}

	return r, nil
}

func newParamTemplate() *template.Template {
	return template.New("").Option("missingkey=error").Delims("[[", "]]")
}

func fillExprWithParams(expr string, values map[string]string) (string, error) {
	var buf bytes.Buffer
	t, err := newParamTemplate().Parse(expr)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse expression")
	}
	if err = t.Execute(&buf, values); err != nil {
		return "", errors.Wrap(err, "failed to fill expression placeholders")
	}
	return buf.String(), nil
}

func validateParameters(definitions models.AlertExprParamsDefinitions, values models.AlertExprParamsValues) error {
	if len(definitions) != len(values) {
		return status.Errorf(codes.InvalidArgument, "Expression requires %d parameters, but got %d.",
			len(definitions), len(values))
	}

	valuesM := make(map[string]models.AlertExprParamValue)
	for _, v := range values {
		valuesM[v.Name] = v
	}

	for _, d := range definitions {
		value, ok := valuesM[d.Name]
		if !ok {
			return status.Errorf(codes.InvalidArgument, "Parameter %s is missing.", d.Name)
		}

		if string(d.Type) != string(value.Type) {
			return status.Errorf(codes.InvalidArgument, "Parameter %s has type %s instead of %s.", d.Name, value.Type, d.Type)
		}

		if d.Type == models.Float {
			v := d.FloatParam
			fv := value.FloatValue
			if v.Min != nil && pointer.GetFloat64(v.Min) > fv {
				return status.Errorf(codes.InvalidArgument, "Parameter %s value is less than required minimum.", d.Name)
			}

			if v.Max != nil && pointer.GetFloat64(v.Max) < fv {
				return status.Errorf(codes.InvalidArgument, "Parameter %s value is greater than required maximum.", d.Name)
			}
		}
	}

	return nil
}
