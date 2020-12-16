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
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona-platform/saas/pkg/alert"
	"github.com/percona/pmm/api/managementpb"
	iav1beta1 "github.com/percona/pmm/api/managementpb/ia"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm-managed/models"
)

func convertParamUnit(u string) iav1beta1.ParamUnit {
	// TODO: check possible variants.
	switch strings.ToLower(u) {
	case "%", "percentage":
		return iav1beta1.ParamUnit_PERCENTAGE
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

func convertSeverity(severity models.Severity) managementpb.Severity {
	switch severity {
	case models.UnknownSeverity:
		return managementpb.Severity_SEVERITY_INVALID
	case models.EmergencySeverity:
		return managementpb.Severity_SEVERITY_EMERGENCY
	case models.AlertSeverity:
		return managementpb.Severity_SEVERITY_ALERT
	case models.CriticalSeverity:
		return managementpb.Severity_SEVERITY_CRITICAL
	case models.ErrorSeverity:
		return managementpb.Severity_SEVERITY_ERROR
	case models.WarningSeverity:
		return managementpb.Severity_SEVERITY_WARNING
	case models.NoticeSeverity:
		return managementpb.Severity_SEVERITY_NOTICE
	case models.InfoSeverity:
		return managementpb.Severity_SEVERITY_INFO
	case models.DebugSeverity:
		return managementpb.Severity_SEVERITY_DEBUG
	default:
		return managementpb.Severity_SEVERITY_INVALID
	}
}
