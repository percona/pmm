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
	"io/ioutil"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona-platform/saas/pkg/alert"
	saas "github.com/percona-platform/saas/pkg/alert"
	"github.com/percona/pmm/api/managementpb"
	iav1beta1 "github.com/percona/pmm/api/managementpb/ia"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	builtinTemplatesPath = "/tmp/ia1/*.yml"
	userTemplatesPath    = "/tmp/ia2/*.yml"
)

// TemplatesService is responsible for interactions with IA rule templates.
type TemplatesService struct {
	l                    *logrus.Entry
	builtinTemplatesPath string
	userTemplatesPath    string

	rw    sync.RWMutex
	rules map[string]saas.Rule
}

// NewTemplatesService creates a new TemplatesService.
func NewTemplatesService() *TemplatesService {
	return &TemplatesService{
		l:                    logrus.WithField("component", "management/ia/templates"),
		builtinTemplatesPath: builtinTemplatesPath,
		userTemplatesPath:    userTemplatesPath,
		rules:                make(map[string]saas.Rule),
	}
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
