// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package deployment

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/models"
)

const (
	// DefaultConfigDir is the dedicated bind dir inside pmm-server that mirrors the Holmes /config mount.
	// It is NOT PMM's /srv data volume.
	DefaultConfigDir = "/holmes-config"

	envFileMode    os.FileMode = 0o600 // secrets
	configFileMode os.FileMode = 0o644
	dirMode        os.FileMode = 0o755
)

// Renderer materializes the DB-stored ADRE deployment config to the shared config directory.
type Renderer struct {
	db  *reform.DB
	dir string
	l   *logrus.Entry
}

// NewRenderer returns a Renderer writing to dir (defaults to DefaultConfigDir when empty).
func NewRenderer(db *reform.DB, dir string, l *logrus.Entry) *Renderer {
	if dir == "" {
		dir = DefaultConfigDir
	}
	return &Renderer{db: db, dir: dir, l: l}
}

// Render writes .env, model_list.yaml, config.yaml and skills/ from the DB. Writes are atomic
// (temp file + rename) so Holmes never reads a half-written file.
func (r *Renderer) Render() error {
	if err := os.MkdirAll(r.dir, dirMode); err != nil {
		return errors.Wrapf(err, "failed to create config dir %q", r.dir)
	}

	prov, err := models.GetAdreProvisioning(r.db)
	if err != nil {
		return err
	}
	mdls, err := models.ListAdreModels(r.db)
	if err != nil {
		return err
	}
	cfg, err := models.GetAdreHolmesConfig(r.db)
	if err != nil {
		return err
	}
	skills, err := models.ListAdreSkills(r.db, true)
	if err != nil {
		return err
	}

	if err := r.renderEnv(prov); err != nil {
		return err
	}
	if err := r.renderModelList(mdls); err != nil {
		return err
	}
	// Inject the minted PMM service-account token where config.yaml references it (the Grafana-token
	// placeholder used by the prometheus/metrics and grafana/dashboards toolsets).
	configYAML := strings.ReplaceAll(cfg.ConfigYAML, grafanaTokenPlaceholder, prov.PMMSAToken)
	// Never overwrite an existing config.yaml with an empty one — an empty config.yaml strips all
	// PMM toolsets from Holmes. A fresh setup must populate config.yaml before it is rendered.
	if strings.TrimSpace(configYAML) != "" {
		if err := writeFileAtomic(filepath.Join(r.dir, "config.yaml"), []byte(configYAML), configFileMode); err != nil {
			return errors.Wrap(err, "failed to render config.yaml")
		}
	} else {
		r.l.Warn("ADRE config.yaml is empty in DB; leaving any existing config.yaml on disk untouched")
	}
	if err := r.renderSkills(skills); err != nil {
		return err
	}
	return nil
}

func (r *Renderer) renderEnv(p *models.AdreProvisioning) error {
	// Bootstrap env consumed as the holmesgpt compose env_file at container (re)start.
	var sb strings.Builder
	sb.WriteString("# Rendered by PMM — bootstrap env for HolmesGPT (do not edit by hand).\n")
	sb.WriteString(fmt.Sprintf("PMM_URL=%s\n", p.PMMURL))
	sb.WriteString(fmt.Sprintf("PMM_API_TOKEN=%s\n", p.PMMSAToken))
	sb.WriteString(fmt.Sprintf("HOLMES_API_KEY=%s\n", p.HolmesAPIKey))
	return errors.Wrap(writeFileAtomic(filepath.Join(r.dir, ".env"), []byte(sb.String()), envFileMode), "failed to render .env")
}

type modelEntry struct {
	Model   string `yaml:"model"`
	APIBase string `yaml:"api_base,omitempty"`
	APIKey  string `yaml:"api_key,omitempty"`
}

func (r *Renderer) renderModelList(mdls []*models.AdreModel) error {
	ml := make(map[string]modelEntry, len(mdls))
	for _, m := range mdls {
		ml[m.Name] = modelEntry{Model: m.LitellmModel, APIBase: m.APIBase, APIKey: m.APIKey}
	}
	b, err := yaml.Marshal(ml) // yaml.v3 sorts map keys → deterministic output
	if err != nil {
		return errors.Wrap(err, "failed to marshal model_list.yaml")
	}
	return errors.Wrap(writeFileAtomic(filepath.Join(r.dir, "model_list.yaml"), b, envFileMode), "failed to render model_list.yaml")
}

func (r *Renderer) renderSkills(skills []*models.AdreSkill) error {
	skillsDir := filepath.Join(r.dir, "skills")
	if err := os.MkdirAll(skillsDir, dirMode); err != nil {
		return errors.Wrap(err, "failed to create skills dir")
	}

	want := make(map[string]struct{}, len(skills))
	for _, s := range skills {
		if !validSkillName(s.Name) {
			r.l.Warnf("skipping skill with unsafe name %q", s.Name)
			continue
		}
		want[s.Name] = struct{}{}
		dir := filepath.Join(skillsDir, s.Name)
		if err := os.MkdirAll(dir, dirMode); err != nil {
			return errors.Wrapf(err, "failed to create skill dir %q", s.Name)
		}
		if err := writeFileAtomic(filepath.Join(dir, "SKILL.md"), []byte(s.Body), configFileMode); err != nil { //nolint:noinlineerr
			return errors.Wrapf(err, "failed to render skill %q", s.Name)
		}
	}

	// Remove skill dirs that are no longer wanted (deleted/disabled).
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return errors.Wrap(err, "failed to read skills dir")
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, ok := want[e.Name()]; !ok {
			if err := os.RemoveAll(filepath.Join(skillsDir, e.Name())); err != nil { //nolint:noinlineerr
				return errors.Wrapf(err, "failed to remove stale skill %q", e.Name())
			}
		}
	}
	return nil
}

// validSkillName guards against path traversal in the skill directory name.
func validSkillName(name string) bool {
	if name == "" || strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return false
	}
	return name == filepath.Base(name)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
