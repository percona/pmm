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

// Package alerting contains PMM alerting functionality.
package alerting

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/percona/saas/pkg/alert"
	"github.com/percona/saas/pkg/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	alerting "github.com/percona/pmm/api/alerting/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/data"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/dir"
	"github.com/percona/pmm/managed/utils/envvars"
	"github.com/percona/pmm/managed/utils/platform"
	"github.com/percona/pmm/managed/utils/signatures"
)

const (
	templatesDir              = "/srv/alerting/templates"
	portalRequestTimeout      = 2 * time.Minute // time limit to get templates list from the portal
	defaultEvaluationInterval = time.Minute

	dirPerm = os.FileMode(0o775)
)

// Service is responsible alerting templates and rules creation from them.
type Service struct {
	db                 *reform.DB
	l                  *logrus.Entry
	platformClient     *platform.Client
	grafanaClient      grafanaClient
	userTemplatesPath  string
	platformPublicKeys []string

	rw        sync.RWMutex
	templates map[string]models.Template

	alerting.UnimplementedAlertingServiceServer
}

// NewService creates a new Service.
func NewService(db *reform.DB, platformClient *platform.Client, grafanaClient grafanaClient) (*Service, error) { //nolint:unparam
	l := logrus.WithField("component", "management/alerting")

	err := dir.CreateDataDir(templatesDir, "pmm", "pmm", dirPerm)
	if err != nil {
		l.Error(err)
	}

	var platformPublicKeys []string
	if k := envvars.GetPlatformPublicKeys(); k != nil {
		l.Warnf("Percona Platform public keys changed to %q.", k)
		platformPublicKeys = k
	}

	s := &Service{
		db:                 db,
		l:                  l,
		platformClient:     platformClient,
		grafanaClient:      grafanaClient,
		userTemplatesPath:  templatesDir,
		platformPublicKeys: platformPublicKeys,
		templates:          make(map[string]models.Template),
	}

	return s, nil
}

// Enabled returns if service is enabled and can be used.
func (s *Service) Enabled() bool {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.WithError(err).Error("can't get settings")
		return false
	}
	return settings.IsAlertingEnabled()
}

// GetTemplates return collected templates.
func (s *Service) GetTemplates() map[string]models.Template {
	s.rw.RLock()
	defer s.rw.RUnlock()

	res := make(map[string]models.Template, len(s.templates))
	for n, r := range s.templates {
		res[n] = r
	}
	return res
}

// CollectTemplates collects Percona Alerting rule templates from various sources like:
// builtin templates: read from the generated variable of type embed.FS
// SaaS templates: templates downloaded from checks service.
// User file templates: read from yaml files created by the user in `/srv/alerting/templates`.
// User API templates: in the DB created using the API.
func (s *Service) CollectTemplates(ctx context.Context) {
	var templates []*models.Template
	builtInTemplates, err := s.loadTemplatesFromAssets(ctx)
	if err != nil {
		s.l.Errorf("Failed to load built-in rule templates: %s.", err)
		return
	}
	templates = append(templates, builtInTemplates...)

	userDefinedTemplates, err := s.loadTemplatesFromUserFiles(ctx)
	if err != nil {
		s.l.Errorf("Failed to load user-defined rule templates: %s.", err)
		return
	}
	templates = append(templates, userDefinedTemplates...)

	dbTemplates, err := s.loadTemplatesFromDB()
	if err != nil {
		s.l.Errorf("Failed to load rule templates from DB: %s.", err)
		return
	}
	templates = append(templates, dbTemplates...)

	saasTemplates, err := s.downloadTemplates(ctx)
	if err != nil {
		// just log the error and don't return, if the user is not connected to SaaS
		// we should still collect and show the Built-In templates.
		s.l.Errorf("Failed to download rule templates from SaaS: %s.", err)
	}
	templates = append(templates, saasTemplates...)

	// replace previously stored templates with newly collected ones.
	s.rw.Lock()
	defer s.rw.Unlock()
	s.templates = make(map[string]models.Template, len(templates))
	for _, t := range templates {
		s.templates[t.Name] = *t
	}
}

// loadTemplatesFromAssets loads built-in alerting rule templates from pmm-managed binary's assets.
func (s *Service) loadTemplatesFromAssets(ctx context.Context) ([]*models.Template, error) {
	var res []*models.Template
	walkDirFunc := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return errors.Wrapf(err, "error occurred while traversing templates folder: %s", path)
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if d.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(data.AlertRuleTemplates, path)
		if err != nil {
			return errors.Wrapf(err, "failed to read rule template asset: %s", path)
		}

		// be strict about built-in templates
		params := &alert.ParseParams{
			DisallowUnknownFields:    true,
			DisallowInvalidTemplates: true,
		}
		templates, err := alert.Parse(bytes.NewReader(data), params)
		if err != nil {
			return errors.Wrapf(err, "failed to parse rule template asset: %s", path)
		}

		// built-in-specific validations
		// TODO move to some better / common place

		if l := len(templates); l != 1 {
			return errors.Errorf("%q should contain exactly one template, got %d", path, l)
		}

		t := templates[0]

		filename := filepath.Base(path)
		if strings.HasPrefix(filename, "pmm_") {
			return errors.Errorf("%q file name should not start with 'pmm_' prefix", path)
		}
		if !strings.HasPrefix(t.Name, "pmm_") {
			return errors.Errorf("%s %q: template name should start with 'pmm_' prefix", path, t.Name)
		}
		if expected := strings.TrimPrefix(t.Name, "pmm_") + ".yml"; filename != expected {
			return errors.Errorf("template file name %q should be %q", filename, expected)
		}
		if len(t.Annotations) != 2 || t.Annotations["summary"] == "" || t.Annotations["description"] == "" {
			return errors.Errorf("%s %q: template should contain exactly two annotations: summary and description", path, t.Name)
		}

		tm, err := models.ConvertTemplate(&t, models.BuiltInSource)
		if err != nil {
			return errors.Wrap(err, "failed to convert alert rule template")
		}

		res = append(res, tm)
		return nil
	}
	err := fs.WalkDir(data.AlertRuleTemplates, ".", walkDirFunc)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// loadTemplatesFromUserFiles loads user's alerting rule templates from /srv/alerting/templates.
func (s *Service) loadTemplatesFromUserFiles(ctx context.Context) ([]*models.Template, error) {
	paths, err := dir.FindFilesWithExtensions(s.userTemplatesPath, "yml", "yaml")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get paths")
	}

	res := make([]*models.Template, 0, len(paths))
	for _, path := range paths {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		data, err := os.ReadFile(path) //nolint:gosec
		if err != nil {
			s.l.Warnf("Failed to load rule template file %s.", path)
			continue
		}

		// be strict about user files
		params := &alert.ParseParams{
			DisallowUnknownFields:    true,
			DisallowInvalidTemplates: true,
		}
		templates, err := alert.Parse(bytes.NewReader(data), params)
		if err != nil {
			s.l.Warnf("Failed to parse rule template file %s.", path)
			continue
		}

		for _, t := range templates {
			if err = validateUserTemplate(&t); err != nil {
				s.l.Warnf("%s %s", path, err)
				continue
			}

			tm, err := models.ConvertTemplate(&t, models.UserFileSource)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert alert rule template")
			}

			res = append(res, tm)
		}
	}
	return res, nil
}

func (s *Service) loadTemplatesFromDB() ([]*models.Template, error) {
	var templates []*models.Template
	errTx := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		templates, err = models.FindTemplates(tx.Querier)
		return err
	})
	if errTx != nil {
		return nil, errors.Wrap(errTx, "failed to load rule templates from DB")
	}

	return templates, nil
}

// downloadTemplates downloads Percona Alerting templates from Percona Portal.
func (s *Service) downloadTemplates(ctx context.Context) ([]*models.Template, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if !settings.IsTelemetryEnabled() {
		s.l.Debug("Alert templates downloading skipped due to disabled telemetry.")
		return nil, nil
	}

	nCtx, cancel := context.WithTimeout(ctx, portalRequestTimeout)
	defer cancel()

	resp, err := s.platformClient.GetTemplates(nCtx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err = signatures.Verify(s.l, resp.File, resp.Signatures, s.platformPublicKeys); err != nil {
		return nil, err
	}

	// be liberal about files from SaaS for smooth transition to future versions
	params := &alert.ParseParams{
		DisallowUnknownFields:    false,
		DisallowInvalidTemplates: false,
	}
	templates, err := alert.Parse(strings.NewReader(resp.File), params)
	if err != nil {
		return nil, err
	}

	res := make([]*models.Template, 0, len(templates))
	for _, t := range templates {
		tm, err := models.ConvertTemplate(&t, models.SAASSource)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert alert rule template")
		}
		res = append(res, tm)
	}

	return res, nil
}

// validateUserTemplate validates user-provided template (API or file).
func validateUserTemplate(t *alert.Template) error {
	// TODO move to some better place

	if strings.HasPrefix(t.Name, "pmm_") || strings.HasPrefix(t.Name, "saas_") {
		return errors.Errorf("%s: template name should not start with 'pmm_' or 'saas_' prefix", t.Name)
	}

	// TODO more validations

	// validate expression template with fake parameters values
	params := make(map[string]string, len(t.Params))
	for _, p := range t.Params {
		var value string
		switch p.Type {
		case alert.Float:
			value = "0"
		case alert.Bool:
			value = "false"
		case alert.String:
			value = "param_text"
		default:
			return errors.Errorf("invalid parameter type %s", p.Type)
		}

		params[p.Name] = value
	}

	if _, err := fillExprWithParams(t.Expr, params); err != nil {
		return err
	}

	return nil
}

func convertSource(source models.Source) alerting.TemplateSource {
	switch source {
	case models.BuiltInSource:
		return alerting.TemplateSource_TEMPLATE_SOURCE_BUILT_IN
	case models.SAASSource:
		return alerting.TemplateSource_TEMPLATE_SOURCE_SAAS
	case models.UserFileSource:
		return alerting.TemplateSource_TEMPLATE_SOURCE_USER_FILE
	case models.UserAPISource:
		return alerting.TemplateSource_TEMPLATE_SOURCE_USER_API
	default:
		return alerting.TemplateSource_TEMPLATE_SOURCE_UNSPECIFIED
	}
}

func convertParamType(t models.ParamType) alerting.ParamType {
	switch t {
	case models.Float:
		return alerting.ParamType_PARAM_TYPE_FLOAT
	case models.Bool:
		return alerting.ParamType_PARAM_TYPE_BOOL
	case models.String:
		return alerting.ParamType_PARAM_TYPE_STRING
	}

	// do not add `default:` to make exhaustive linter do its job

	return alerting.ParamType_PARAM_TYPE_UNSPECIFIED
}

// ListTemplates returns a list of all collected Alert Rule Templates.
func (s *Service) ListTemplates(ctx context.Context, req *alerting.ListTemplatesRequest) (*alerting.ListTemplatesResponse, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if !settings.IsAlertingEnabled() {
		return nil, services.ErrAlertingDisabled
	}

	var pageIndex int
	var pageSize int
	if req.PageIndex != nil {
		pageIndex = int(*req.PageIndex)
	}
	if req.PageSize != nil {
		pageSize = int(*req.PageSize)
	}

	if req.Reload {
		s.CollectTemplates(ctx)
	}

	templates := s.GetTemplates()
	res := &alerting.ListTemplatesResponse{
		Templates:  make([]*alerting.Template, 0, len(templates)),
		TotalItems: int32(len(templates)),
		TotalPages: 1,
	}

	if pageSize > 0 {
		res.TotalPages = int32(len(templates) / pageSize)
		if len(templates)%pageSize > 0 {
			res.TotalPages++
		}
	}

	names := make([]string, 0, len(templates))
	for name := range templates {
		names = append(names, name)
	}
	sort.Strings(names)

	from, to := pageIndex*pageSize, (pageIndex+1)*pageSize
	if to > len(names) || to == 0 {
		to = len(names)
	}

	if from > len(names) {
		from = len(names)
	}

	for _, name := range names[from:to] {
		t, err := convertTemplate(s.l, templates[name])
		if err != nil {
			return nil, err
		}

		res.Templates = append(res.Templates, t)
	}

	return res, nil
}

// CreateTemplate creates a new template.
func (s *Service) CreateTemplate(ctx context.Context, req *alerting.CreateTemplateRequest) (*alerting.CreateTemplateResponse, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if !settings.IsAlertingEnabled() {
		return nil, services.ErrAlertingDisabled
	}

	pParams := &alert.ParseParams{
		DisallowUnknownFields:    true,
		DisallowInvalidTemplates: true,
	}

	templates, err := alert.Parse(strings.NewReader(req.Yaml), pParams)
	if err != nil {
		s.l.Errorf("failed to parse rule template form request: +%v", err)
		return nil, status.Errorf(codes.InvalidArgument, "Failed to parse rule template: %v.", err)
	}

	uniqueNames := make(map[string]struct{}, len(templates))
	for _, t := range templates {
		if _, ok := uniqueNames[t.Name]; ok {
			return nil, status.Errorf(codes.InvalidArgument, "Template with name '%s' declared more that once.", t.Name)
		}
		uniqueNames[t.Name] = struct{}{}
		if err = validateUserTemplate(&t); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "%s.", err)
		}
	}

	errTx := s.db.InTransaction(func(tx *reform.TX) error {
		for _, t := range templates {
			if _, err = models.CreateTemplate(tx.Querier, &models.CreateTemplateParams{
				Template: &t,
				Source:   models.UserAPISource,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if errTx != nil {
		return nil, errTx
	}

	s.CollectTemplates(ctx)

	return &alerting.CreateTemplateResponse{}, nil
}

// UpdateTemplate updates existing template, previously created via API.
func (s *Service) UpdateTemplate(ctx context.Context, req *alerting.UpdateTemplateRequest) (*alerting.UpdateTemplateResponse, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if !settings.IsAlertingEnabled() {
		return nil, services.ErrAlertingDisabled
	}

	parseParams := &alert.ParseParams{
		DisallowUnknownFields:    true,
		DisallowInvalidTemplates: true,
	}

	templates, err := alert.Parse(strings.NewReader(req.Yaml), parseParams)
	if err != nil {
		s.l.Errorf("failed to parse rule template form request: +%v", err)
		return nil, status.Error(codes.InvalidArgument, "Failed to parse rule template.")
	}

	if len(templates) != 1 {
		return nil, status.Error(codes.InvalidArgument, "Request should contain exactly one rule template.")
	}

	tmpl := templates[0]

	if err = validateUserTemplate(&tmpl); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s.", err)
	}

	changeParams := &models.ChangeTemplateParams{
		Template: &tmpl,
		Name:     req.Name,
	}

	e := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		_, err = models.ChangeTemplate(tx.Querier, changeParams)
		return err
	})
	if e != nil {
		return nil, e
	}

	s.CollectTemplates(ctx)

	return &alerting.UpdateTemplateResponse{}, nil
}

// DeleteTemplate deletes existing, previously created via API.
func (s *Service) DeleteTemplate(ctx context.Context, req *alerting.DeleteTemplateRequest) (*alerting.DeleteTemplateResponse, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if !settings.IsAlertingEnabled() {
		return nil, services.ErrAlertingDisabled
	}

	e := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		return models.RemoveTemplate(tx.Querier, req.Name)
	})
	if e != nil {
		return nil, e
	}

	s.CollectTemplates(ctx)

	return &alerting.DeleteTemplateResponse{}, nil
}

func convertTemplate(l *logrus.Entry, template models.Template) (*alerting.Template, error) {
	labels, err := template.GetLabels()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	annotations, err := template.GetAnnotations()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	t := &alerting.Template{
		Name:        template.Name,
		Summary:     template.Summary,
		Expr:        template.Expr,
		Params:      convertParamDefinitions(l, template.Params),
		For:         durationpb.New(template.For),
		Severity:    managementv1.Severity(template.Severity),
		Labels:      labels,
		Annotations: annotations,
		Source:      convertSource(template.Source),
		Yaml:        template.Yaml,
	}

	t.CreatedAt = timestamppb.New(template.CreatedAt)
	if err = t.CreatedAt.CheckValid(); err != nil {
		return nil, err
	}

	return t, nil
}

func convertParamDefinitions(l *logrus.Entry, params models.AlertExprParamsDefinitions) []*alerting.ParamDefinition {
	res := make([]*alerting.ParamDefinition, 0, len(params))
	for _, p := range params {
		pd := &alerting.ParamDefinition{
			Name:    p.Name,
			Summary: p.Summary,
			Unit:    convertParamUnit(p.Unit),
			Type:    convertParamType(p.Type),
		}

		switch p.Type {
		case models.Float:
			var fp alerting.FloatParamDefinition
			if p.FloatParam != nil {
				if p.FloatParam.Default != nil {
					fp.Default = p.FloatParam.Default
				}

				if p.FloatParam.Min != nil {
					fp.Min = p.FloatParam.Min
				}

				if p.FloatParam.Max != nil {
					fp.Max = p.FloatParam.Max
				}
			}

			pd.Value = &alerting.ParamDefinition_Float{Float: &fp}
			res = append(res, pd)

		case models.Bool, models.String:
			l.Warnf("Skipping unsupported parameter type %q.", p.Type)
		}

		// do not add `default:` to make exhaustive linter do its job
	}

	return res
}

// CreateRule creates alert rule from the given template.
func (s *Service) CreateRule(ctx context.Context, req *alerting.CreateRuleRequest) (*alerting.CreateRuleResponse, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if !settings.IsAlertingEnabled() {
		return nil, services.ErrAlertingDisabled
	}

	if req.TemplateName == "" {
		return nil, status.Error(codes.InvalidArgument, "Template name should be specified.")
	}

	if req.FolderUid == "" {
		return nil, status.Error(codes.InvalidArgument, "Folder UID should be specified.")
	}

	if req.Group == "" {
		return nil, status.Error(codes.InvalidArgument, "Rule group name should be specified.")
	}

	metricsDatasourceUID, err := s.grafanaClient.GetDatasourceUIDByID(ctx, 1) // 1 - it's id of Metrics datasource in PMM
	if err != nil {
		return nil, err
	}

	template, ok := s.GetTemplates()[req.TemplateName]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "Unknown template %s.", req.TemplateName)
	}

	paramsValues, err := convertParamsValuesToModel(req.Params)
	if err != nil {
		return nil, err
	}

	if err := validateParameters(template.Params, paramsValues); err != nil {
		return nil, err
	}

	forDuration := template.For
	if req.For != nil {
		forDuration = req.For.AsDuration()
	}

	expr, err := fillExprWithParams(template.Expr, paramsValues.AsStringMap())
	if err != nil {
		return nil, errors.Wrap(err, "failed to fill rule expression with parameters")
	}

	for _, filter := range req.Filters {
		switch filter.Type {
		case alerting.FilterType_FILTER_TYPE_MATCH:
			expr = fmt.Sprintf(`label_match(%s, "%s", "%s")`, expr, filter.Label, filter.Regexp)
		case alerting.FilterType_FILTER_TYPE_MISMATCH:
			expr = fmt.Sprintf(`label_mismatch(%s, "%s", "%s")`, expr, filter.Label, filter.Regexp)
		default:
			return nil, errors.Errorf("unknown filter type: %T", filter)
		}
	}

	ta, err := template.GetAnnotations()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get template annotations")
	}

	// Copy annotations form template
	annotations := make(map[string]string)
	if err = transformMaps(ta, annotations, paramsValues.AsStringMap()); err != nil {
		return nil, errors.Wrap(err, "failed to fill template annotations placeholders")
	}

	labels := make(map[string]string)
	// Copy labels form template
	if err = transformMaps(req.CustomLabels, labels, paramsValues.AsStringMap()); err != nil {
		return nil, errors.Wrap(err, "failed to fill rule labels placeholders")
	}

	tl, err := template.GetLabels()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get template labels")
	}

	// Add rule labels
	if err = transformMaps(tl, labels, paramsValues.AsStringMap()); err != nil {
		return nil, errors.Wrap(err, "failed to fill template labels placeholders")
	}

	// Do not add volatile values like `{{ $value }}` to labels as it will break alerts identity.
	labels["percona_alerting"] = "1" // TODO: do we actually need it?
	labels["severity"] = common.Severity(req.Severity).String()
	labels["template_name"] = req.TemplateName

	rule := services.Rule{
		GrafanaAlert: services.GrafanaAlert{
			Title:        req.Name,
			Condition:    "A",
			NoDataState:  "OK",
			ExecErrState: "Alerting",
			Data: []services.Data{
				{
					RefID:         "A",
					DatasourceUID: metricsDatasourceUID,
					// TODO: https://community.grafana.com/t/grafana-requires-time-range-for-alert-rule-creation-with-instant-promql-quieriy/70919
					RelativeTimeRange: services.RelativeTimeRange{From: 600, To: 0},
					Model: services.Model{
						Expr:    expr,
						RefID:   "A",
						Instant: true,
					},
				},
			},
		},
		For:         forDuration.String(),
		Annotations: annotations,
		Labels:      labels,
	}

	// TODO: align it with grafanas default value: https://grafana.com/docs/grafana/v9.0/setup-grafana/configure-grafana/#min_interval
	interval := defaultEvaluationInterval.String()
	if req.Interval != nil {
		interval = req.Interval.AsDuration().String()
	}

	err = s.grafanaClient.CreateAlertRule(ctx, req.FolderUid, req.Group, interval, &rule)
	if err != nil {
		return nil, err
	}

	return &alerting.CreateRuleResponse{}, nil
}

func convertParamsValuesToModel(params []*alerting.ParamValue) (AlertExprParamsValues, error) {
	ruleParams := make(AlertExprParamsValues, len(params))
	for i, param := range params {
		p := AlertExprParamValue{Name: param.Name}

		switch param.Type {
		case alerting.ParamType_PARAM_TYPE_UNSPECIFIED:
			return nil, errors.New("invalid model rule param value type")
		case alerting.ParamType_PARAM_TYPE_BOOL:
			p.Type = models.Bool
			p.BoolValue = param.GetBool()
		case alerting.ParamType_PARAM_TYPE_FLOAT:
			p.Type = models.Float
			p.FloatValue = param.GetFloat()
		case alerting.ParamType_PARAM_TYPE_STRING:
			p.Type = models.Float
			p.StringValue = param.GetString_()
		default:
			return nil, errors.New("invalid model rule param value type")
		}

		ruleParams[i] = p
	}
	return ruleParams, nil
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

func convertParamUnit(u models.ParamUnit) alerting.ParamUnit {
	switch u {
	case models.Percent:
		return alerting.ParamUnit_PARAM_UNIT_PERCENTAGE
	case models.Seconds:
		return alerting.ParamUnit_PARAM_UNIT_SECONDS
	}

	// do not add `default:` to make exhaustive linter do its job

	return alerting.ParamUnit_PARAM_UNIT_UNSPECIFIED
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

func validateParameters(definitions models.AlertExprParamsDefinitions, values AlertExprParamsValues) error {
	if len(definitions) != len(values) {
		return status.Errorf(codes.InvalidArgument, "Expression requires %d parameters, but got %d.",
			len(definitions), len(values))
	}

	valuesM := make(map[string]AlertExprParamValue)
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

		switch d.Type {
		case models.Float:
			v := d.FloatParam
			fv := value.FloatValue
			if v.Min != nil && pointer.GetFloat64(v.Min) > fv {
				return status.Errorf(codes.InvalidArgument, "Parameter %s value is less than required minimum.", d.Name)
			}

			if v.Max != nil && pointer.GetFloat64(v.Max) < fv {
				return status.Errorf(codes.InvalidArgument, "Parameter %s value is greater than required maximum.", d.Name)
			}
		default:
			// nothing
		}
	}

	return nil
}

// Check interfaces.
var (
	_ alerting.AlertingServiceServer = (*Service)(nil)
)

// AlertExprParamValue represents rule parameter value.
type AlertExprParamValue struct {
	Name        string           `json:"name"`
	Type        models.ParamType `json:"type"`
	BoolValue   bool             `json:"bool"`
	FloatValue  float64          `json:"float"`
	StringValue string           `json:"string"`
}

// AlertExprParamsValues represents rule parameters values slice.
type AlertExprParamsValues []AlertExprParamValue

// AsStringMap convert param values to string map, where parameter name is a map key and parameter value is a map value.
func (p AlertExprParamsValues) AsStringMap() map[string]string {
	m := make(map[string]string, len(p))
	for _, rp := range p {
		var value string
		switch rp.Type {
		case models.Float:
			value = fmt.Sprint(rp.FloatValue)
		case models.Bool:
			value = strconv.FormatBool(rp.BoolValue)
		case models.String:
			value = rp.StringValue
		}
		// do not add `default:` to make exhaustive linter do its job

		m[rp.Name] = value
	}

	return m
}
