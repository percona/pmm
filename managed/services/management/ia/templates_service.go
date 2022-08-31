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
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/percona-platform/saas/pkg/alert"
	"github.com/percona-platform/saas/pkg/common"
	"github.com/percona/promconfig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/managementpb"
	iav1beta1 "github.com/percona/pmm/api/managementpb/ia"
	"github.com/percona/pmm/managed/data"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/dir"
	"github.com/percona/pmm/managed/utils/envvars"
	"github.com/percona/pmm/managed/utils/platform"
	"github.com/percona/pmm/managed/utils/signatures"
)

const (
	templatesDir         = "/srv/ia/templates"
	portalRequestTimeout = 2 * time.Minute // time limit to get templates list from the portal

)

// templateInfo represents alerting rule template information from various sources.
//
// TODO We already have models.Template, iav1beta1.Template, and alert.Template.
//
//	We probably can remove that type.
type templateInfo struct {
	alert.Template
	Yaml      string
	Source    iav1beta1.TemplateSource
	CreatedAt *time.Time
}

// TemplatesService is responsible for interactions with IA rule templates.
type TemplatesService struct {
	db                 *reform.DB
	l                  *logrus.Entry
	platformClient     *platform.Client
	userTemplatesPath  string
	platformPublicKeys []string

	rw        sync.RWMutex
	templates map[string]templateInfo

	iav1beta1.UnimplementedTemplatesServer
}

// NewTemplatesService creates a new TemplatesService.
func NewTemplatesService(db *reform.DB, platformClient *platform.Client) (*TemplatesService, error) {
	l := logrus.WithField("component", "management/ia/templates")

	err := dir.CreateDataDir(templatesDir, "pmm", "pmm", dirPerm)
	if err != nil {
		l.Error(err)
	}

	var platformPublicKeys []string
	if k := envvars.GetPlatformPublicKeys(); k != nil {
		l.Warnf("Percona Platform public keys changed to %q.", k)
		platformPublicKeys = k
	}

	s := &TemplatesService{
		db:                 db,
		l:                  l,
		platformClient:     platformClient,
		userTemplatesPath:  templatesDir,
		platformPublicKeys: platformPublicKeys,
		templates:          make(map[string]templateInfo),
	}

	return s, nil
}

// Enabled returns if service is enabled and can be used.
func (s *TemplatesService) Enabled() bool {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.WithError(err).Error("can't get settings")
		return false
	}
	return settings.IntegratedAlerting.Enabled
}

func newParamTemplate() *template.Template {
	return template.New("").Option("missingkey=error").Delims("[[", "]]")
}

// getTemplates return collected templates.
func (s *TemplatesService) getTemplates() map[string]templateInfo {
	s.rw.RLock()
	defer s.rw.RUnlock()

	res := make(map[string]templateInfo, len(s.templates))
	for n, r := range s.templates {
		res[n] = r
	}
	return res
}

// CollectTemplates collects IA rule templates from various sources like:
// builtin templates: read from the generated variable of type embed.FS
// SaaS templates: templates downloaded from checks service.
// user file templates: read from yaml files created by the user in `/srv/ia/templates`
// user API templates: in the DB created using the API.
func (s *TemplatesService) CollectTemplates(ctx context.Context) {
	builtInTemplates, err := s.loadTemplatesFromAssets(ctx)
	if err != nil {
		s.l.Errorf("Failed to load built-in rule templates: %s.", err)
		return
	}

	userDefinedTemplates, err := s.loadTemplatesFromUserFiles(ctx)
	if err != nil {
		s.l.Errorf("Failed to load user-defined rule templates: %s.", err)
		return
	}

	dbTemplates, err := s.loadTemplatesFromDB()
	if err != nil {
		s.l.Errorf("Failed to load rule templates from DB: %s.", err)
		return
	}

	saasTemplates, err := s.downloadTemplates(ctx)
	if err != nil {
		// just log the error and don't return, if the user is not connected to SaaS
		// we should still collect and show the Built-In templates.
		s.l.Errorf("Failed to download rule templates from SaaS: %s.", err)
	}

	templates := make([]templateInfo, 0, len(builtInTemplates)+len(userDefinedTemplates)+len(dbTemplates)+len(saasTemplates))

	for _, t := range builtInTemplates {
		templates = append(templates, templateInfo{
			Template: t,
			Source:   iav1beta1.TemplateSource_BUILT_IN,
		})
	}

	for _, t := range userDefinedTemplates {
		templates = append(templates, templateInfo{
			Template: t,
			Source:   iav1beta1.TemplateSource_USER_FILE,
		})
	}

	for _, t := range saasTemplates {
		templates = append(templates, templateInfo{
			Template: t,
			Source:   iav1beta1.TemplateSource_SAAS,
		})
	}

	templates = append(templates, dbTemplates...)

	// replace previously stored templates with newly collected ones.
	s.rw.Lock()
	defer s.rw.Unlock()
	s.templates = make(map[string]templateInfo, len(templates))
	for _, t := range templates {
		// TODO Check for name clashes? Allow users to re-define built-in templates?
		// Reserve prefix for built-in or user-defined templates?
		// https://jira.percona.com/browse/PMM-7023

		s.templates[t.Name] = t
	}
}

// loadTemplatesFromAssets loads built-in alerting rule templates from pmm-managed binary's assets.
func (s *TemplatesService) loadTemplatesFromAssets(ctx context.Context) ([]alert.Template, error) {
	var res []alert.Template
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

		data, err := fs.ReadFile(data.IATemplates, path)
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

		res = append(res, t)
		return nil
	}
	err := fs.WalkDir(data.IATemplates, ".", walkDirFunc)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// loadTemplatesFromUserFiles loads user's alerting rule templates from /srv/ia/templates.
func (s *TemplatesService) loadTemplatesFromUserFiles(ctx context.Context) ([]alert.Template, error) {
	paths, err := dir.FindFilesWithExtensions(s.userTemplatesPath, "yml", "yaml")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get paths")
	}

	res := make([]alert.Template, 0, len(paths))
	for _, path := range paths {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		data, err := ioutil.ReadFile(path) //nolint:gosec
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
			if err = validateUserTemplate(&t); err != nil { //nolint:gosec
				s.l.Warnf("%s %s", path, err)
				continue
			}

			res = append(res, t)
		}
	}
	return res, nil
}

func (s *TemplatesService) loadTemplatesFromDB() ([]templateInfo, error) {
	var templates []models.Template
	e := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		templates, err = models.FindTemplates(tx.Querier)
		return err
	})
	if e != nil {
		return nil, errors.Wrap(e, "failed to load rule templates from DB")
	}

	res := make([]templateInfo, 0, len(templates))
	for _, t := range templates {
		t := t
		params := make([]alert.Parameter, 0, len(t.Params))
		for _, param := range t.Params {
			p := alert.Parameter{
				Name:    param.Name,
				Summary: param.Summary,
				Unit:    alert.Unit(param.Unit),
				Type:    alert.Type(param.Type),
			}

			switch alert.Type(param.Type) {
			case alert.Float:
				f := param.FloatParam

				if f.Default != nil {
					p.Value = *f.Default
				}

				if f.Min != nil && f.Max != nil {
					p.Range = []interface{}{*f.Min, *f.Max}
				}
			}

			params = append(params, p)
		}

		labels, err := t.GetLabels()
		if err != nil {
			return nil, errors.Wrap(err, "failed to load template labels")
		}

		annotations, err := t.GetAnnotations()
		if err != nil {
			return nil, errors.Wrap(err, "failed to load template annotations")
		}

		res = append(res,
			templateInfo{
				Template: alert.Template{
					Name:        t.Name,
					Version:     t.Version,
					Summary:     t.Summary,
					Expr:        t.Expr,
					Params:      params,
					For:         promconfig.Duration(t.For),
					Severity:    common.Severity(t.Severity),
					Labels:      labels,
					Annotations: annotations,
				},
				Yaml:      t.Yaml,
				Source:    convertSource(t.Source),
				CreatedAt: &t.CreatedAt,
			},
		)
	}
	return res, nil
}

// downloadTemplates downloads IA templates from SaaS.
func (s *TemplatesService) downloadTemplates(ctx context.Context) ([]alert.Template, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if settings.Telemetry.Disabled {
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

	return templates, nil
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

func convertSource(source models.Source) iav1beta1.TemplateSource {
	switch source {
	case models.BuiltInSource:
		return iav1beta1.TemplateSource_BUILT_IN
	case models.SAASSource:
		return iav1beta1.TemplateSource_SAAS
	case models.UserFileSource:
		return iav1beta1.TemplateSource_USER_FILE
	case models.UserAPISource:
		return iav1beta1.TemplateSource_USER_API
	default:
		return iav1beta1.TemplateSource_TEMPLATE_SOURCE_INVALID
	}
}

func convertParamType(t alert.Type) iav1beta1.ParamType {
	// TODO: add another types.
	switch t {
	case alert.Float:
		return iav1beta1.ParamType_FLOAT
	default:
		return iav1beta1.ParamType_PARAM_TYPE_INVALID
	}
}

// ListTemplates returns a list of all collected Alert Rule Templates.
func (s *TemplatesService) ListTemplates(ctx context.Context, req *iav1beta1.ListTemplatesRequest) (*iav1beta1.ListTemplatesResponse, error) {
	var pageIndex int
	var pageSize int
	if req.PageParams != nil {
		pageIndex = int(req.PageParams.Index)
		pageSize = int(req.PageParams.PageSize)
	}

	if req.Reload {
		s.CollectTemplates(ctx)
	}

	templates := s.getTemplates()
	res := &iav1beta1.ListTemplatesResponse{
		Templates: make([]*iav1beta1.Template, 0, len(templates)),
		Totals: &managementpb.PageTotals{
			TotalItems: int32(len(templates)),
			TotalPages: 1,
		},
	}

	if pageSize > 0 {
		res.Totals.TotalPages = int32(len(templates) / pageSize)
		if len(templates)%pageSize > 0 {
			res.Totals.TotalPages++
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
func (s *TemplatesService) CreateTemplate(ctx context.Context, req *iav1beta1.CreateTemplateRequest) (*iav1beta1.CreateTemplateResponse, error) {
	pParams := &alert.ParseParams{
		DisallowUnknownFields:    true,
		DisallowInvalidTemplates: true,
	}

	templates, err := alert.Parse(strings.NewReader(req.Yaml), pParams)
	if err != nil {
		s.l.Errorf("failed to parse rule template form request: +%v", err)
		return nil, status.Error(codes.InvalidArgument, "Failed to parse rule template.")
	}

	if len(templates) != 1 {
		return nil, status.Error(codes.InvalidArgument, "Request should contain exactly one rule template.")
	}

	for _, t := range templates {
		if err = validateUserTemplate(&t); err != nil { //nolint:gosec
			return nil, status.Errorf(codes.InvalidArgument, "%s.", err)
		}
	}

	params := &models.CreateTemplateParams{
		Template: &templates[0],
		Yaml:     req.Yaml,
		Source:   models.UserAPISource,
	}

	e := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		_, err = models.CreateTemplate(tx.Querier, params)
		return err
	})
	if e != nil {
		return nil, e
	}

	s.CollectTemplates(ctx)

	return &iav1beta1.CreateTemplateResponse{}, nil
}

// UpdateTemplate updates existing template, previously created via API.
func (s *TemplatesService) UpdateTemplate(ctx context.Context, req *iav1beta1.UpdateTemplateRequest) (*iav1beta1.UpdateTemplateResponse, error) {
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
		Yaml:     req.Yaml,
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

	return &iav1beta1.UpdateTemplateResponse{}, nil
}

// DeleteTemplate deletes existing, previously created via API.
func (s *TemplatesService) DeleteTemplate(ctx context.Context, req *iav1beta1.DeleteTemplateRequest) (*iav1beta1.DeleteTemplateResponse, error) {
	e := s.db.InTransaction(func(tx *reform.TX) error {
		return models.RemoveTemplate(tx.Querier, req.Name)
	})
	if e != nil {
		return nil, e
	}

	s.CollectTemplates(ctx)

	return &iav1beta1.DeleteTemplateResponse{}, nil
}

func convertTemplate(l *logrus.Entry, template templateInfo) (*iav1beta1.Template, error) {
	var err error
	t := &iav1beta1.Template{
		Name:        template.Name,
		Summary:     template.Summary,
		Expr:        template.Expr,
		Params:      make([]*iav1beta1.ParamDefinition, 0, len(template.Params)),
		For:         durationpb.New(time.Duration(template.For)),
		Severity:    managementpb.Severity(template.Severity),
		Labels:      template.Labels,
		Annotations: template.Annotations,
		Source:      template.Source,
		Yaml:        template.Yaml,
	}

	if template.CreatedAt != nil {
		t.CreatedAt = timestamppb.New(*template.CreatedAt)
		if err = t.CreatedAt.CheckValid(); err != nil {
			return nil, err
		}
	}

	t.Params, err = convertParamDefinitions(l, template.Params)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func convertParamDefinitions(l *logrus.Entry, params []alert.Parameter) ([]*iav1beta1.ParamDefinition, error) {
	res := make([]*iav1beta1.ParamDefinition, 0, len(params))
	for _, p := range params {
		pd := &iav1beta1.ParamDefinition{
			Name:    p.Name,
			Summary: p.Summary,
			Unit:    convertParamUnit(p.Unit),
			Type:    convertParamType(p.Type),
		}

		var err error
		switch p.Type {
		case alert.Float:
			var fp iav1beta1.FloatParamDefinition
			if p.Value != nil {
				fp.Default, err = p.GetValueForFloat()
				if err != nil {
					return nil, errors.Wrap(err, "failed to get value for float parameter")
				}
				fp.HasDefault = true
			}

			if len(p.Range) != 0 {
				fp.Min, fp.Max, err = p.GetRangeForFloat()
				if err != nil {
					return nil, errors.Wrap(err, "failed to get range for float parameter")
				}
				fp.HasMin, fp.HasMax = true, true
			}

			pd.Value = &iav1beta1.ParamDefinition_Float{Float: &fp}
			res = append(res, pd)

		case alert.Bool, alert.String:
			l.Warnf("Skipping unsupported parameter type %q.", p.Type)
		}

		// do not add `default:` to make exhaustive linter do its job
	}

	return res, nil
}

// Check interfaces.
var (
	_ iav1beta1.TemplatesServer = (*TemplatesService)(nil)
)
