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
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/percona-platform/saas/pkg/alert"
	"github.com/percona-platform/saas/pkg/common"
	iav1beta1 "github.com/percona/pmm/api/managementpb/ia"
	"github.com/percona/promconfig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm-managed/data"
	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/dir"
)

const (
	templatesParentDir = "/srv/ia"
	templatesDir       = "/srv/ia/templates"
	rulesParentDir     = "/etc/ia"
	rulesDir           = "/etc/ia/rules"
	dirPerm            = os.FileMode(0o775)
)

// templateInfo represents alerting rule template information from various sources.
//
// TODO We already have models.Template, iav1beta1.Template, and alert.Template.
//      We probably can remove that type.
type templateInfo struct {
	alert.Template
	Yaml      string
	Source    iav1beta1.TemplateSource
	CreatedAt *time.Time
}

// TemplatesService is responsible for interactions with IA rule templates.
type TemplatesService struct {
	db                *reform.DB
	l                 *logrus.Entry
	userTemplatesPath string
	rulesPath         string // used for testing

	rw        sync.RWMutex
	templates map[string]templateInfo
}

// NewTemplatesService creates a new TemplatesService.
func NewTemplatesService(db *reform.DB) *TemplatesService {
	l := logrus.WithField("component", "management/ia/templates")

	err := dir.CreateDataDir(templatesParentDir, "pmm", "pmm", dirPerm)
	if err != nil {
		l.Error(err)
	}
	err = dir.CreateDataDir(templatesDir, "pmm", "pmm", dirPerm)
	if err != nil {
		l.Error(err)
	}

	err = dir.CreateDataDir(rulesParentDir, "pmm", "pmm", dirPerm)
	if err != nil {
		l.Error(err)
	}
	// TODO move to rules service
	err = dir.CreateDataDir(rulesDir, "pmm", "pmm", dirPerm)
	if err != nil {
		l.Error(err)
	}

	return &TemplatesService{
		db:                db,
		l:                 l,
		userTemplatesPath: templatesDir + "/*.yml",
		rulesPath:         rulesDir,
		templates:         make(map[string]templateInfo),
	}
}

func newParamTemplate() *template.Template {
	return template.New("").Option("missingkey=error").Delims("[[", "]]")
}

// getCollected return collected templates.
func (s *TemplatesService) getCollected(ctx context.Context) map[string]templateInfo {
	s.rw.RLock()
	defer s.rw.RUnlock()

	res := make(map[string]templateInfo)
	for n, r := range s.templates {
		res[n] = r
	}
	return res
}

// collect collects IA rule templates from various sources like:
// builtin templates: read from the generated code in bindata.go.
// user file templates: read from yaml files created by the user in `/srv/ia/templates`
// user API templates: in the DB created using the API.
func (s *TemplatesService) collect(ctx context.Context) {
	builtInTemplates, err := s.loadTemplatesFromAssets(ctx)
	if err != nil {
		s.l.Errorf("Failed to load built-in rule templates: %s.", err)
		return
	}

	userDefinedTemplates, err := s.loadTemplatesFromFiles(ctx, s.userTemplatesPath)
	if err != nil {
		s.l.Errorf("Failed to load user-defined rule templates: %s.", err)
		return
	}

	dbTemplates, err := s.loadTemplatesFromDB()
	if err != nil {
		s.l.Errorf("Failed to load rule templates from DB: %s.", err)
		return
	}

	templates := make([]templateInfo, 0, len(builtInTemplates)+len(userDefinedTemplates)+len(dbTemplates))

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

	templates = append(templates, dbTemplates...)

	// TODO download templates from SAAS.

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

func (s *TemplatesService) loadTemplatesFromAssets(ctx context.Context) ([]alert.Template, error) {
	paths := data.AssetNames()
	res := make([]alert.Template, 0, len(paths))
	for _, path := range paths {
		data, err := data.Asset(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load rule template file: %s", path)
		}

		// be strict about builtin templates.
		params := &alert.ParseParams{
			DisallowUnknownFields:    true,
			DisallowInvalidTemplates: true,
		}
		templates, err := alert.Parse(bytes.NewReader(data), params)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse rule template file")
		}

		res = append(res, templates...)
	}
	return res, nil
}

func (s *TemplatesService) loadTemplatesFromFiles(ctx context.Context, path string) ([]alert.Template, error) {
	paths, err := filepath.Glob(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get paths")
	}

	res := make([]alert.Template, 0, len(paths))
	for _, path := range paths {
		templates, err := s.loadFile(ctx, path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load rule template file: %s", path)
		}

		res = append(res, templates...)
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
		return nil, errors.Wrap(e, "failed to load rule templates form DB")
	}

	res := make([]templateInfo, 0, len(templates))
	for _, template := range templates {
		template := template
		params := make([]alert.Parameter, len(template.Params))
		for _, param := range template.Params {
			p := alert.Parameter{
				Name:    param.Name,
				Summary: param.Summary,
				Unit:    param.Unit,
				Type:    alert.Type(param.Type),
			}

			switch alert.Type(param.Type) {
			case alert.Float:
				f := param.FloatParam
				p.Value = f.Default
				p.Range = []interface{}{f.Min, f.Max}
			}

			params = append(params, p)
		}

		labels, err := template.GetLabels()
		if err != nil {
			return nil, errors.Wrap(err, "failed to load template labels")
		}

		annotations, err := template.GetAnnotations()
		if err != nil {
			return nil, errors.Wrap(err, "failed to load template annotations")
		}

		res = append(res,
			templateInfo{
				Template: alert.Template{
					Name:        template.Name,
					Version:     template.Version,
					Summary:     template.Summary,
					Tiers:       template.Tiers,
					Expr:        template.Expr,
					Params:      params,
					For:         promconfig.Duration(template.For),
					Severity:    common.Severity(convertSeverity(template.Severity)),
					Labels:      labels,
					Annotations: annotations,
				},
				Yaml:      template.Yaml,
				Source:    convertSource(template.Source),
				CreatedAt: &template.CreatedAt,
			},
		)
	}

	return res, nil
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

// loadFile parses IA rule template file.
func (s *TemplatesService) loadFile(ctx context.Context, file string) ([]alert.Template, error) {
	if ctx.Err() != nil {
		return nil, errors.WithStack(ctx.Err())
	}

	data, err := ioutil.ReadFile(file) //nolint:gosec
	if err != nil {
		return nil, errors.Wrap(err, "failed to read rule template file")
	}

	// be strict about local files
	params := &alert.ParseParams{
		DisallowUnknownFields:    true,
		DisallowInvalidTemplates: true,
	}
	templates, err := alert.Parse(bytes.NewReader(data), params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse rule template file")
	}

	return templates, nil
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
	Alert       string              `yaml:"alert"` // same as alert name in template file
	Expr        string              `yaml:"expr"`
	Duration    promconfig.Duration `yaml:"for"`
	Labels      map[string]string   `yaml:"labels,omitempty"`
	Annotations map[string]string   `yaml:"annotations,omitempty"`
}

// converts an alert template rule to a rule file. generates one file per rule.
func (s *TemplatesService) convertTemplates(ctx context.Context) error {
	templates := s.getCollected(ctx)
	for _, template := range templates {
		r := rule{
			Alert:       template.Name,
			Duration:    template.For,
			Labels:      make(map[string]string, len(template.Labels)),
			Annotations: make(map[string]string, len(template.Annotations)),
		}

		data := make(map[string]string, len(template.Params))
		for _, param := range template.Params {
			data[param.Name] = fmt.Sprint(param.Value)
		}

		var buf bytes.Buffer
		t, err := newParamTemplate().Parse(template.Expr)
		if err != nil {
			return errors.Wrap(err, "failed to convert rule template")
		}
		if err = t.Execute(&buf, data); err != nil {
			return errors.Wrap(err, "failed to convert rule template")
		}
		r.Expr = buf.String()

		err = transformMaps(template.Labels, r.Labels, data)
		if err != nil {
			return errors.Wrap(err, "failed to convert rule template")
		}

		// add parameters to labels
		for _, p := range template.Params {
			r.Labels[p.Name] = fmt.Sprint(p.Value)
		}

		// add special labels
		r.Labels["ia"] = "1"
		r.Labels["severity"] = template.Severity.String()

		err = transformMaps(template.Annotations, r.Annotations, data)
		if err != nil {
			return errors.Wrap(err, "failed to convert rule template")
		}

		rf := &ruleFile{
			Group: []ruleGroup{{
				Name:  "PMM Server Integrated Alerting",
				Rules: []rule{r},
			}},
		}

		err = s.dumpRule(rf)
		if err != nil {
			return errors.Wrap(err, "failed to dump alert rules")
		}
	}
	return nil
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
func (s *TemplatesService) dumpRule(rule *ruleFile) error {
	b, err := yaml.Marshal(rule)
	if err != nil {
		return errors.Errorf("failed to marshal rule %s", err)
	}
	b = append([]byte("---\n"), b...)

	alertRule := rule.Group[0].Rules[0]
	if alertRule.Alert == "" {
		return errors.New("alert rule not initialized")
	}
	path := s.rulesPath + alertRule.Alert + ".yml"
	if err = ioutil.WriteFile(path, b, 0o644); err != nil {
		return errors.Errorf("failed to dump rule to file %s: %s", s.rulesPath, err)
	}
	return nil
}

// ListTemplates returns a list of all collected Alert Rule Templates.
func (s *TemplatesService) ListTemplates(ctx context.Context, req *iav1beta1.ListTemplatesRequest) (*iav1beta1.ListTemplatesResponse, error) {
	if req.Reload {
		s.collect(ctx)
	}

	templates := s.getCollected(ctx)
	res := &iav1beta1.ListTemplatesResponse{
		Templates: make([]*iav1beta1.Template, 0, len(templates)),
	}
	for _, template := range templates {
		t, err := convertTemplate(s.l, template)
		if err != nil {
			return nil, err
		}

		res.Templates = append(res.Templates, t)
	}

	sort.Slice(res.Templates, func(i, j int) bool { return res.Templates[i].Name < res.Templates[j].Name })
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

	s.collect(ctx)

	return &iav1beta1.CreateTemplateResponse{}, nil
}

// UpdateTemplate updates existing template, previously created via API.
func (s *TemplatesService) UpdateTemplate(ctx context.Context, req *iav1beta1.UpdateTemplateRequest) (*iav1beta1.UpdateTemplateResponse, error) {
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

	params := &models.ChangeTemplateParams{
		Template: &templates[0],
		Yaml:     req.Yaml,
	}

	e := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		_, err = models.ChangeTemplate(tx.Querier, params)
		return err
	})
	if e != nil {
		return nil, e
	}

	s.collect(ctx)

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
	return &iav1beta1.DeleteTemplateResponse{}, nil
}

// Check interfaces.
var (
	_ iav1beta1.TemplatesServer = (*TemplatesService)(nil)
)
