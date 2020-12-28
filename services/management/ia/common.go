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

package ia

import (
	"os"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona-platform/saas/pkg/alert"
	"github.com/percona/pmm/api/managementpb"
	iav1beta1 "github.com/percona/pmm/api/managementpb/ia"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm-managed/models"
)

const dirPerm = os.FileMode(0o775)

func convertParamUnit(u string) iav1beta1.ParamUnit {
	switch u {
	case "%":
		return iav1beta1.ParamUnit_PERCENTAGE
	case "s":
		return iav1beta1.ParamUnit_SECONDS
	default:
		return iav1beta1.ParamUnit_PARAM_UNIT_INVALID
	}
}

func convertTemplate(l *logrus.Entry, template templateInfo) (*iav1beta1.Template, error) {
	t := &iav1beta1.Template{
		Name:        template.Name,
		Summary:     template.Summary,
		Expr:        template.Expr,
		Params:      make([]*iav1beta1.TemplateParam, 0, len(template.Params)),
		For:         ptypes.DurationProto(time.Duration(template.For)),
		Severity:    managementpb.Severity(template.Severity),
		Labels:      template.Labels,
		Annotations: template.Annotations,
		Source:      template.Source,
		Yaml:        template.Yaml,
	}

	if template.CreatedAt != nil {
		var err error
		if t.CreatedAt, err = ptypes.TimestampProto(*template.CreatedAt); err != nil {
			return nil, err
		}
	}

	for _, p := range template.Params {
		tp := &iav1beta1.TemplateParam{
			Name:    p.Name,
			Summary: p.Summary,
			Unit:    convertParamUnit(p.Unit),
			Type:    convertParamType(p.Type),
		}

		switch p.Type {
		case alert.Float:
			value, err := p.GetValueForFloat()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get value for float parameter")
			}

			fp := &iav1beta1.TemplateFloatParam{
				HasDefault: true,           // TODO remove or fill with valid value.
				Default:    float32(value), // TODO eliminate conversion.
			}

			if p.Range != nil {
				min, max, err := p.GetRangeForFloat()
				if err != nil {
					return nil, errors.Wrap(err, "failed to get range for float parameter")
				}

				fp.HasMin = true      // TODO remove or fill with valid value.
				fp.Min = float32(min) // TODO eliminate conversion.,
				fp.HasMax = true      // TODO remove or fill with valid value.
				fp.Max = float32(max) // TODO eliminate conversion.,
			}

			tp.Value = &iav1beta1.TemplateParam_Float{Float: fp}

			t.Params = append(t.Params, tp)

		default:
			l.Warnf("Skipping unexpected parameter type %q for %q.", p.Type, template.Name)
		}
	}

	return t, nil
}

func convertRule(l *logrus.Entry, rule *models.Rule, template templateInfo, channels []*models.Channel) (*iav1beta1.Rule, error) {
	r := &iav1beta1.Rule{
		RuleId:   rule.ID,
		Disabled: rule.Disabled,
		Summary:  rule.Summary,
		Severity: managementpb.Severity(rule.Severity),
		For:      ptypes.DurationProto(rule.For),
	}

	var err error
	r.CreatedAt, err = ptypes.TimestampProto(rule.CreatedAt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert timestamp")
	}

	r.Template, err = convertTemplate(l, template)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert template")
	}

	r.Params, err = convertModelToRuleParams(rule.Params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert rule parameters")
	}

	r.CustomLabels, err = rule.GetCustomLabels()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load rule labels")
	}

	r.Filters = make([]*iav1beta1.Filter, len(rule.Filters))
	for i, filter := range rule.Filters {
		r.Filters[i] = &iav1beta1.Filter{
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
