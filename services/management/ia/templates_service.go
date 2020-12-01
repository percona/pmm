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
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona-platform/saas/pkg/alert"
	saas "github.com/percona-platform/saas/pkg/alert"
	"github.com/percona/pmm/api/managementpb"
	iav1beta1 "github.com/percona/pmm/api/managementpb/ia"
	"github.com/percona/promconfig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v3"
)

const (
	builtinTemplatesPath = "/tmp/ia1/*.yml"
	userTemplatesPath    = "/tmp/ia2/*.yml"

	ruleFileDir = "/tmp/ia1/"
)

// TemplatesService is responsible for interactions with IA rule templates.
type TemplatesService struct {
	db                   *reform.DB
	l                    *logrus.Entry
	builtinTemplatesPath string
	userTemplatesPath    string

	rw    sync.RWMutex
	rules map[string]saas.Rule
}

// NewTemplatesService creates a new TemplatesService.
func NewTemplatesService(db *reform.DB) *TemplatesService {
	return &TemplatesService{
		db:                   db,
		l:                    logrus.WithField("component", "management/ia/templates"),
		builtinTemplatesPath: builtinTemplatesPath,
		userTemplatesPath:    userTemplatesPath,
		rules:                make(map[string]saas.Rule),
	}
}

func newParamTemplate() *template.Template {
	return template.New("").Option("missingkey=error").Delims("[[", "]]")
}

// getCollected return collected templates.
func (svc *TemplatesService) getCollected(ctx context.Context) map[string]saas.Rule {
	svc.rw.RLock()
	defer svc.rw.RUnlock()

	res := make(map[string]saas.Rule)
	for n, r := range svc.rules {
		res[n] = r
	}
	return res
}

// collect collects IA rule templates from various sources like
// built-in templates shipped with PMM and defined by the users.
func (svc *TemplatesService) collect(ctx context.Context) {
	builtinFilePaths, err := filepath.Glob(svc.builtinTemplatesPath)
	if err != nil {
		svc.l.Errorf("Failed to get paths of built-in templates files shipped with PMM: %s.", err)
		return
	}

	userFilePaths, err := filepath.Glob(svc.userTemplatesPath)
	if err != nil {
		svc.l.Errorf("Failed to get paths of user-defined template files: %s.", err)
		return
	}

	rules := make([]saas.Rule, 0, len(builtinFilePaths)+len(userFilePaths))

	for _, path := range builtinFilePaths {
		r, err := svc.loadFile(ctx, path)
		if err != nil {
			svc.l.Errorf("Failed to load shipped rule template file: %s, reason: %s.", path, err)
			return
		}

		rules = append(rules, r...)
	}

	for _, path := range userFilePaths {
		r, err := svc.loadFile(ctx, path)
		if err != nil {
			svc.l.Errorf("Failed to load user-defined rule template file: %s, reason: %s.", path, err)
			return
		}
		rules = append(rules, r...)
	}

	// TODO download templates from SAAS.

	// replace previously stored rules with newly collected ones.
	svc.rw.Lock()
	defer svc.rw.Unlock()
	svc.rules = make(map[string]saas.Rule, len(rules))
	for _, r := range rules {
		// TODO Check for name clashes? Allow users to re-define built-in rules?
		// Reserve prefix for built-in or user-defined rules?
		// https://jira.percona.com/browse/PMM-7023

		svc.rules[r.Name] = r
	}
}

// loadFile parses IA rule template file.
func (svc *TemplatesService) loadFile(ctx context.Context, file string) ([]saas.Rule, error) {
	if ctx.Err() != nil {
		return nil, errors.WithStack(ctx.Err())
	}

	data, err := ioutil.ReadFile(file) //nolint:gosec
	if err != nil {
		return nil, errors.Wrap(err, "failed to read rule template file")
	}

	// be strict about local files
	params := &saas.ParseParams{
		DisallowUnknownFields: true,
		DisallowInvalidRules:  true,
	}
	rules, err := saas.Parse(bytes.NewReader(data), params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse rule template file")
	}

	return rules, nil
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
func (svc *TemplatesService) convertTemplates(ctx context.Context) error {
	templates := svc.getCollected(ctx)
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

		err = dumpRule(rf)
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

// dump the transformed IA rules to a file.
func dumpRule(rule *ruleFile) error {
	b, err := yaml.Marshal(rule)
	if err != nil {
		return errors.Errorf("failed to marshal rule %s", err)
	}
	b = append([]byte("---\n"), b...)

	alertRule := rule.Group[0].Rules[0]
	if alertRule.Alert == "" {
		return errors.New("alert rule not initialized")
	}
	path := ruleFileDir + alertRule.Alert + ".yml"

	_, err = os.Stat(ruleFileDir)
	if os.IsNotExist(err) {
		err = os.Mkdir(ruleFileDir, 0750) // TODO move to https://jira.percona.com/browse/PMM-7024
		if err != nil {
			return err
		}
	}
	if err = ioutil.WriteFile(path, b, 0644); err != nil {
		return errors.Errorf("failed to dump rule to file %s: %s", ruleFileDir, err)

	}
	return nil
}

// ListTemplates returns a list of all collected Alert Rule Templates.
func (svc *TemplatesService) ListTemplates(ctx context.Context, req *iav1beta1.ListTemplatesRequest) (*iav1beta1.ListTemplatesResponse, error) {
	if req.Reload {
		svc.collect(ctx)
	}

	templates := svc.getCollected(ctx)
	res := &iav1beta1.ListTemplatesResponse{
		Templates: make([]*iav1beta1.Template, 0, len(templates)),
	}
	for _, r := range templates {
		t := &iav1beta1.Template{
			Name:        r.Name,
			Summary:     r.Summary,
			Expr:        r.Expr,
			Params:      make([]*iav1beta1.TemplateParam, 0, len(r.Params)),
			For:         ptypes.DurationProto(time.Duration(r.For)),
			Severity:    managementpb.Severity(r.Severity),
			Labels:      r.Labels,
			Annotations: r.Annotations,
			Source:      iav1beta1.TemplateSource_TEMPLATE_SOURCE_INVALID, // TODO
		}

		for _, p := range r.Params {
			var tp *iav1beta1.TemplateParam
			switch p.Type {
			case alert.Float:
				tp = &iav1beta1.TemplateParam{
					Name:    p.Name,
					Summary: p.Summary,
					Unit:    iav1beta1.ParamUnit_PARAM_UNIT_INVALID, // TODO
					Type:    iav1beta1.ParamType_FLOAT,
					Value:   nil, // TODO
				}
			default:
				svc.l.Warnf("Skipping unexpected parameter type %q for %q.", p.Type, r.Name)
			}

			if tp != nil {
				t.Params = append(t.Params, tp)
			}
		}

		res.Templates = append(res.Templates, t)
	}

	sort.Slice(res.Templates, func(i, j int) bool { return res.Templates[i].Name < res.Templates[j].Name })
	return res, nil
}

// CreateTemplate creates a new template.
func (svc *TemplatesService) CreateTemplate(ctx context.Context, req *iav1beta1.CreateTemplateRequest) (*iav1beta1.CreateTemplateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateTemplate not implemented")
}

// UpdateTemplate updates existing template, previously created via API.
func (svc *TemplatesService) UpdateTemplate(ctx context.Context, req *iav1beta1.UpdateTemplateRequest) (*iav1beta1.UpdateTemplateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateTemplate not implemented")
}

// DeleteTemplate deletes existing, previously created via API.
func (svc *TemplatesService) DeleteTemplate(ctx context.Context, req *iav1beta1.DeleteTemplateRequest) (*iav1beta1.DeleteTemplateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteTemplate not implemented")
}

// Check interfaces.
var (
	_ iav1beta1.TemplatesServer = (*TemplatesService)(nil)
)
