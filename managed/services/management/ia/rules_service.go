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
	"bytes"
	"context"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/percona-platform/saas/pkg/alert"
	"github.com/percona-platform/saas/pkg/common"
	"github.com/percona/promconfig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/api/managementpb"
	iav1beta1 "github.com/percona/pmm/api/managementpb/ia"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/dir"
	"github.com/percona/pmm/managed/utils/stringset"
)

const (
	rulesDir = "/etc/ia/rules"
)

// RulesService represents API for Integrated Alerting Rules.
type RulesService struct {
	db           *reform.DB
	l            *logrus.Entry
	templates    *TemplatesService
	vmalert      vmAlert
	alertManager alertManager
	rulesPath    string // used for testing

	iav1beta1.UnimplementedRulesServer
}

// NewRulesService creates an API for Integrated Alerting Rules.
func NewRulesService(db *reform.DB, templates *TemplatesService, vmalert vmAlert, alertManager alertManager) *RulesService {
	l := logrus.WithField("component", "management/ia/rules")

	err := dir.CreateDataDir(rulesDir, "pmm", "pmm", dirPerm)
	if err != nil {
		l.Error(err)
	}

	s := &RulesService{
		db:           db,
		l:            l,
		templates:    templates,
		vmalert:      vmalert,
		alertManager: alertManager,
		rulesPath:    rulesDir,
	}
	s.updateConfigurations()

	return s
}

// Enabled returns if service is enabled and can be used.
func (s *RulesService) Enabled() bool {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.WithError(err).Error("can't get settings")
		return false
	}
	return settings.IntegratedAlerting.Enabled
}

// TODO Move this and related types to https://github.com/percona/promconfig
// https://jira.percona.com/browse/PMM-7069
type ruleFile struct {
	Group []ruleGroup `yaml:"groups"`
}

type ruleGroup struct {
	Name  string `yaml:"name"`
	Rules []rule `yaml:"rules"`
}

type rule struct {
	Alert       string              `yaml:"alert"` // Rule ID.
	Expr        string              `yaml:"expr"`
	Duration    promconfig.Duration `yaml:"for"`
	Labels      map[string]string   `yaml:"labels,omitempty"`
	Annotations map[string]string   `yaml:"annotations,omitempty"`
}

// RemoveVMAlertRulesFiles removes all generated rules files (*.yml) on the ia path.
func (s *RulesService) RemoveVMAlertRulesFiles() error {
	matches, err := filepath.Glob(s.rulesPath + "/*.yml")
	if err != nil {
		return errors.WithStack(err)
	}
	for _, match := range matches {
		if err = os.RemoveAll(match); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// WriteVMAlertRulesFiles converts all available rules to VMAlert rule files.
func (s *RulesService) WriteVMAlertRulesFiles() {
	rules, err := models.FindRules(s.db.Querier)
	if err != nil {
		s.l.Errorf("Failed to get available alert rules: %+v", err)
		return
	}

	ruleFiles, err := s.prepareRulesFiles(rules)
	if err != nil {
		s.l.Errorf("Failed to prepare alert rule files: %+v", err)
		return
	}

	if err = s.RemoveVMAlertRulesFiles(); err != nil {
		s.l.Errorf("Failed to clean old alert rule files: %+v", err)
		return
	}

	for _, file := range ruleFiles {
		err = s.writeRuleFile(&file) //nolint:gosec
		if err != nil {
			s.l.Errorf("Failed to write alert rule file: %+v", err)
		}
	}
}

// prepareRulesFiles converts collected IA rules to Alertmanager rule files content.
func (s *RulesService) prepareRulesFiles(rules []*models.Rule) ([]ruleFile, error) {
	res := make([]ruleFile, 0, len(rules))
	for _, ruleM := range rules {
		if ruleM.Disabled {
			s.l.Debugf("Skipping rule %s as it is disabled.", ruleM.ID)
			continue
		}

		r := rule{
			Alert:       ruleM.ID,
			Duration:    promconfig.Duration(ruleM.For),
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
		}

		params := ruleM.ParamsValues.AsStringMap()

		var err error
		r.Expr, err = fillExprWithParams(ruleM.ExprTemplate, params)
		if err != nil {
			return nil, errors.Wrap(err, "failed to fill rule expression with parameters")
		}

		annotations, err := ruleM.GetAnnotations()
		if err != nil {
			return nil, errors.Wrap(err, "failed to read rule annotations")
		}
		// Copy annotations form template
		if err = transformMaps(annotations, r.Annotations, params); err != nil {
			return nil, errors.Wrap(err, "failed to fill template annotations placeholders")
		}

		r.Annotations["rule"] = ruleM.Name

		labels, err := ruleM.GetLabels()
		if err != nil {
			return nil, errors.Wrap(err, "failed to read rule labels")
		}

		// Copy labels form template
		if err = transformMaps(labels, r.Labels, params); err != nil {
			return nil, errors.Wrap(err, "failed to fill template labels placeholders")
		}

		customLabels, err := ruleM.GetCustomLabels()
		if err != nil {
			return nil, errors.Wrap(err, "failed to read rule custom labels")
		}

		// Add rule labels
		if err = transformMaps(customLabels, r.Labels, params); err != nil {
			return nil, errors.Wrap(err, "failed to fill rule labels placeholders")
		}

		// Do not add volatile values like `{{ $value }}` to labels as it will break alerts identity.
		r.Labels["ia"] = "1"
		r.Labels["severity"] = common.Severity(ruleM.Severity).String()
		r.Labels["rule_id"] = ruleM.ID
		r.Labels["template_name"] = ruleM.TemplateName

		res = append(res, ruleFile{
			Group: []ruleGroup{{
				Name:  "PMM Integrated Alerting",
				Rules: []rule{r},
			}},
		})
	}

	return res, nil
}

// fills templates found in labels and annotaitons with values.
func transformMaps(src map[string]string, dest map[string]string, data map[string]string) error {
	var buf bytes.Buffer
	for k, v := range src {
		buf.Reset()
		t, err := newParamTemplate().Parse(v)
		if err != nil {
			return err
		}
		if err = t.Execute(&buf, data); err != nil {
			return err
		}
		dest[k] = buf.String()
	}
	return nil
}

// dump the transformed IA templates to a file.
func (s *RulesService) writeRuleFile(rule *ruleFile) error {
	b, err := yaml.Marshal(rule)
	if err != nil {
		return errors.Errorf("failed to marshal rule %v", err)
	}
	b = append([]byte("---\n"), b...)

	alertRule := rule.Group[0].Rules[0]
	if alertRule.Alert == "" {
		return errors.New("alert rule not initialized")
	}

	fileName := strings.TrimPrefix(alertRule.Alert, "/rule_id/")
	path := s.rulesPath + "/" + fileName + ".yml"
	if err = ioutil.WriteFile(path, b, 0o644); err != nil {
		return errors.Errorf("failed to dump rule to file %s: %v", s.rulesPath, err)
	}

	return nil
}

// ListAlertRules returns a list of all Integrated Alerting rules.
func (s *RulesService) ListAlertRules(ctx context.Context, req *iav1beta1.ListAlertRulesRequest) (*iav1beta1.ListAlertRulesResponse, error) {
	var pageIndex int
	pageSize := math.MaxInt32
	if req.PageParams != nil {
		pageIndex = int(req.PageParams.Index)
		pageSize = int(req.PageParams.PageSize)
	}

	var rules []*models.Rule
	var channels []*models.Channel
	var totalItems int
	errTx := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		rules, err = models.FindRulesOnPage(tx.Querier, pageIndex, pageSize)
		if err != nil {
			return errors.WithStack(err)
		}

		totalItems, err = models.CountRules(tx.Querier)
		if err != nil {
			return errors.WithStack(err)
		}

		channelsIDs := make(map[string]struct{}, len(rules))
		for _, rule := range rules {
			for _, id := range rule.ChannelIDs {
				channelsIDs[id] = struct{}{}
			}
		}

		channels, err = models.FindChannelsByIDs(tx.Querier, stringset.ToSlice(channelsIDs))
		if err != nil {
			return errors.WithStack(err)
		}

		return nil
	})
	if errTx != nil {
		return nil, errors.WithStack(errTx)
	}

	res, err := s.convertAlertRules(rules, channels)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	totalPages := totalItems / pageSize
	if totalItems%pageSize > 0 {
		totalPages++
	}

	totals := &managementpb.PageTotals{
		TotalItems: int32(totalItems),
		TotalPages: int32(totalPages),
	}

	return &iav1beta1.ListAlertRulesResponse{Rules: res, Totals: totals}, nil
}

func (s *RulesService) convertAlertRules(rules []*models.Rule, channels []*models.Channel) ([]*iav1beta1.Rule, error) {
	res := make([]*iav1beta1.Rule, 0, len(rules))
	for _, rule := range rules {
		r, err := convertRule(s.l, rule, channels)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		res = append(res, r)
	}

	return res, nil
}

// CreateAlertRule creates Integrated Alerting rule.
func (s *RulesService) CreateAlertRule(ctx context.Context, req *iav1beta1.CreateAlertRuleRequest) (*iav1beta1.CreateAlertRuleResponse, error) {
	if req.TemplateName != "" && req.SourceRuleId != "" {
		return nil, status.Errorf(codes.InvalidArgument, "Both template name and source rule id are specified.")
	}
	if req.TemplateName == "" && req.SourceRuleId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Template name or source rule id should be specified.")
	}

	params := &models.CreateRuleParams{
		Name:         req.Name,
		Disabled:     req.Disabled,
		For:          req.For.AsDuration(),
		Severity:     models.Severity(req.Severity),
		CustomLabels: req.CustomLabels,
		ChannelIDs:   req.ChannelIds,
	}

	var err error
	params.ParamsValues, err = convertParamsValuesToModel(req.Params)
	if err != nil {
		return nil, err
	}

	params.Filters, err = convertFiltersToModel(req.Filters)
	if err != nil {
		return nil, err
	}

	if req.TemplateName != "" {
		template, ok := s.templates.getTemplates()[req.TemplateName]
		if !ok {
			return nil, status.Errorf(codes.NotFound, "Unknown template %s.", req.TemplateName)
		}

		params.TemplateName = template.Name
		params.Summary = template.Summary
		params.ExprTemplate = template.Expr
		params.DefaultFor = time.Duration(template.For)
		params.DefaultSeverity = models.Severity(template.Severity)
		params.Labels = template.Labels
		params.Annotations = template.Annotations

		params.ParamsDefinitions, err = models.ConvertParamsDefinitions(template.Params)
		if err != nil {
			return nil, err
		}
	} else {
		sourceRule, err := models.FindRuleByID(s.db.Querier, req.SourceRuleId)
		if err != nil {
			return nil, err
		}

		params.TemplateName = sourceRule.TemplateName
		params.Summary = sourceRule.Summary
		params.ExprTemplate = sourceRule.ExprTemplate
		params.DefaultFor = sourceRule.DefaultFor
		params.DefaultSeverity = sourceRule.DefaultSeverity
		params.ParamsDefinitions = sourceRule.ParamsDefinitions

		params.Labels, err = sourceRule.GetLabels()
		if err != nil {
			return nil, err
		}

		params.Annotations, err = sourceRule.GetAnnotations()
		if err != nil {
			return nil, err
		}
	}

	if err := validateParameters(params.ParamsDefinitions, params.ParamsValues); err != nil {
		return nil, err
	}

	// Check that we can compile expression with given parameters
	_, err = fillExprWithParams(params.ExprTemplate, params.ParamsValues.AsStringMap())
	if err != nil {
		return nil, err
	}

	var rule *models.Rule
	errTX := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		rule, err = models.CreateRule(tx.Querier, params)
		return err
	})
	if errTX != nil {
		return nil, errTX
	}

	s.updateConfigurations()

	return &iav1beta1.CreateAlertRuleResponse{RuleId: rule.ID}, nil
}

// UpdateAlertRule updates Integrated Alerting rule.
func (s *RulesService) UpdateAlertRule(ctx context.Context, req *iav1beta1.UpdateAlertRuleRequest) (*iav1beta1.UpdateAlertRuleResponse, error) {
	params := &models.ChangeRuleParams{
		Name:         req.Name,
		Disabled:     req.Disabled,
		For:          req.For.AsDuration(),
		Severity:     models.Severity(req.Severity),
		CustomLabels: req.CustomLabels,
		ChannelIDs:   req.ChannelIds,
	}

	var err error
	params.Filters, err = convertFiltersToModel(req.Filters)
	if err != nil {
		return nil, err
	}

	params.ParamsValues, err = convertParamsValuesToModel(req.Params)
	if err != nil {
		return nil, err
	}
	e := s.db.InTransaction(func(tx *reform.TX) error {
		rule, err := models.FindRuleByID(tx.Querier, req.RuleId)
		if err != nil {
			return err
		}

		if err = validateParameters(rule.ParamsDefinitions, params.ParamsValues); err != nil {
			return err
		}

		// Check that we can compile expression with given parameters
		if _, err = fillExprWithParams(rule.ExprTemplate, params.ParamsValues.AsStringMap()); err != nil {
			return errors.Wrap(err, "failed to fill expression template with parameters values")
		}

		_, err = models.ChangeRule(tx.Querier, req.RuleId, params)
		return err
	})
	if e != nil {
		return nil, e
	}

	s.updateConfigurations()

	return &iav1beta1.UpdateAlertRuleResponse{}, nil
}

// ToggleAlertRule allows switching between disabled and enabled states of an Alert Rule.
func (s *RulesService) ToggleAlertRule(ctx context.Context, req *iav1beta1.ToggleAlertRuleRequest) (*iav1beta1.ToggleAlertRuleResponse, error) {
	params := &models.ToggleRuleParams{Disabled: parseBooleanFlag(req.Disabled)}
	e := s.db.InTransaction(func(tx *reform.TX) error {
		_, err := models.ToggleRule(tx.Querier, req.RuleId, params)
		return err
	})
	if e != nil {
		return nil, e
	}

	s.updateConfigurations()

	return &iav1beta1.ToggleAlertRuleResponse{}, nil
}

// DeleteAlertRule deletes Integrated Alerting rule.
func (s *RulesService) DeleteAlertRule(ctx context.Context, req *iav1beta1.DeleteAlertRuleRequest) (*iav1beta1.DeleteAlertRuleResponse, error) {
	e := s.db.InTransaction(func(tx *reform.TX) error {
		return models.RemoveRule(tx.Querier, req.RuleId)
	})
	if e != nil {
		return nil, e
	}

	s.updateConfigurations()

	return &iav1beta1.DeleteAlertRuleResponse{}, nil
}

func (s *RulesService) updateConfigurations() {
	s.WriteVMAlertRulesFiles()
	s.vmalert.RequestConfigurationUpdate()
	s.alertManager.RequestConfigurationUpdate()
}

func convertModelToParamsDefinitions(definitions models.AlertExprParamsDefinitions) ([]*iav1beta1.ParamDefinition, error) {
	res := make([]*iav1beta1.ParamDefinition, 0, len(definitions))
	for _, definition := range definitions {
		t := alert.Type(definition.Type)
		p := &iav1beta1.ParamDefinition{
			Name:    definition.Name,
			Summary: definition.Summary,
			Unit:    convertParamUnit(alert.Unit(definition.Unit)),
			Type:    convertParamType(t),
		}

		switch t {
		case alert.Float:
			var value iav1beta1.FloatParamDefinition
			float := definition.FloatParam
			if float.Default != nil {
				value.HasDefault = true
				value.Default = pointer.GetFloat64(float.Default)
			}

			if float.Min != nil {
				value.HasMin = true
				value.Min = pointer.GetFloat64(float.Min)
			}

			if float.Max != nil {
				value.HasMax = true
				value.Max = pointer.GetFloat64(float.Max)
			}
			p.Value = &iav1beta1.ParamDefinition_Float{Float: &value}
		case alert.Bool, alert.String:
			return nil, errors.Errorf("unsupported parameter type %s", t)
		}

		// do not add `default:` to make exhaustive linter do its job

		res = append(res, p)
	}

	return res, nil
}

func convertModelToParamValues(values models.AlertExprParamsValues) ([]*iav1beta1.ParamValue, error) {
	res := make([]*iav1beta1.ParamValue, len(values))
	for i, param := range values {
		p := &iav1beta1.ParamValue{Name: param.Name}

		switch param.Type {
		case models.Bool:
			p.Type = iav1beta1.ParamType_BOOL
			p.Value = &iav1beta1.ParamValue_Bool{Bool: param.BoolValue}
		case models.Float:
			p.Type = iav1beta1.ParamType_FLOAT
			p.Value = &iav1beta1.ParamValue_Float{Float: param.FloatValue}
		case models.String:
			p.Type = iav1beta1.ParamType_STRING
			p.Value = &iav1beta1.ParamValue_String_{String_: param.StringValue}
		default:
			return nil, errors.Errorf("unknown rule param value type %s", param.Type)
		}
		res[i] = p
	}
	return res, nil
}

func convertParamsValuesToModel(params []*iav1beta1.ParamValue) (models.AlertExprParamsValues, error) {
	ruleParams := make(models.AlertExprParamsValues, len(params))
	for i, param := range params {
		p := models.AlertExprParamValue{Name: param.Name}

		switch param.Type {
		case iav1beta1.ParamType_PARAM_TYPE_INVALID:
			return nil, errors.New("invalid model rule param value type")
		case iav1beta1.ParamType_BOOL:
			p.Type = models.Bool
			p.BoolValue = param.GetBool()
		case iav1beta1.ParamType_FLOAT:
			p.Type = models.Float
			p.FloatValue = param.GetFloat()
		case iav1beta1.ParamType_STRING:
			p.Type = models.Float
			p.StringValue = param.GetString_()
		default:
			return nil, errors.New("invalid model rule param value type")
		}

		ruleParams[i] = p
	}
	return ruleParams, nil
}

func parseBooleanFlag(bf iav1beta1.BooleanFlag) *bool {
	switch bf {
	case iav1beta1.BooleanFlag_TRUE:
		return pointer.ToBool(true)
	case iav1beta1.BooleanFlag_FALSE:
		return pointer.ToBool(false)
	case iav1beta1.BooleanFlag_DO_NOT_CHANGE:
		return nil
	default:
		panic("unexpected value of boolean flag")
	}
}

func convertModelToFilterType(filterType models.FilterType) iav1beta1.FilterType {
	switch filterType {
	case models.Equal:
		return iav1beta1.FilterType_EQUAL
	case models.Regex:
		return iav1beta1.FilterType_REGEX
	default:
		return iav1beta1.FilterType_FILTER_TYPE_INVALID
	}
}

func convertFiltersToModel(filters []*iav1beta1.Filter) (models.Filters, error) {
	res := make(models.Filters, len(filters))
	for i, filter := range filters {
		f := models.Filter{
			Key: filter.Key,
		}

		// Unquote the first encountered quote.
		// Do it only for filters as only they can be set in PMM 2.13.
		f.Val = filter.Value
		for _, q := range []string{`"`, `'`} {
			if strings.HasPrefix(f.Val, q) && strings.HasSuffix(f.Val, q) {
				f.Val = strings.TrimPrefix(f.Val, q)
				f.Val = strings.TrimSuffix(f.Val, q)
				break
			}
		}

		switch filter.Type {
		case iav1beta1.FilterType_EQUAL:
			f.Type = models.Equal
		case iav1beta1.FilterType_REGEX:
			f.Type = models.Regex
		case iav1beta1.FilterType_FILTER_TYPE_INVALID:
			fallthrough
		default:
			return nil, status.Errorf(codes.InvalidArgument, "Unexpected filter type.")
		}
		res[i] = f
	}

	return res, nil
}

// Check interfaces.
var (
	_ iav1beta1.RulesServer = (*RulesService)(nil)
)
