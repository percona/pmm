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
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors" //nolint:depguard
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
	if err := os.MkdirAll(r.dir, dirMode); err != nil { //nolint:noinlineerr
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

	if err := r.renderEnv(prov); err != nil { //nolint:noinlineerr
		return err
	}
	if err := r.renderModelList(mdls); err != nil { //nolint:noinlineerr
		return err
	}
	// Inject the minted PMM service-account token where config.yaml references it (the Grafana-token
	// placeholder used by the prometheus/metrics and grafana/dashboards toolsets).
	configYAML := strings.ReplaceAll(cfg.ConfigYAML, grafanaTokenPlaceholder, prov.PMMSAToken)
	// Never overwrite an existing config.yaml with an empty one — an empty config.yaml strips all
	// PMM toolsets from Holmes. A fresh setup must populate config.yaml before it is rendered.
	if strings.TrimSpace(configYAML) != "" {
		err := writeFileAtomic(filepath.Join(r.dir, "config.yaml"), []byte(configYAML), configFileMode)
		if err != nil {
			return errors.Wrap(err, "failed to render config.yaml")
		}
	} else {
		r.l.Warn("ADRE config.yaml is empty in DB; leaving any existing config.yaml on disk untouched")
	}
	if err := r.renderSkills(skills); err != nil { //nolint:noinlineerr
		return err
	}
	return nil
}

func (r *Renderer) renderEnv(p *models.AdreProvisioning) error {
	// Bootstrap env consumed as the holmesgpt compose env_file at container (re)start.
	var sb strings.Builder
	sb.WriteString("# Rendered by PMM — bootstrap env for HolmesGPT (do not edit by hand).\n")
	fmt.Fprintf(&sb, "PMM_URL=%s\n", p.PMMURL)
	fmt.Fprintf(&sb, "PMM_API_TOKEN=%s\n", p.PMMSAToken)
	fmt.Fprintf(&sb, "HOLMES_API_KEY=%s\n", p.HolmesAPIKey)
	return errors.Wrap(writeFileAtomic(filepath.Join(r.dir, ".env"), []byte(sb.String()), envFileMode), "failed to render .env")
}

func (r *Renderer) renderModelList(mdls []*models.AdreModel) error {
	ml := make(map[string]map[string]any, len(mdls))
	for _, m := range mdls {
		entry := map[string]any{"model": m.LitellmModel}
		if m.APIBase != "" {
			entry["api_base"] = m.APIBase
		}
		if m.APIKey != "" {
			entry["api_key"] = m.APIKey
		}
		// Merge optional per-model LiteLLM params (temperature, num_ctx, api_version, …) for
		// local/self-hosted models. Extra keys win over the base fields if the user repeats them.
		if strings.TrimSpace(m.ExtraParams) != "" {
			extra := map[string]any{}
			if err := yaml.Unmarshal([]byte(m.ExtraParams), &extra); err != nil { //nolint:noinlineerr
				return errors.Wrapf(err, "model %q: invalid extra params YAML", m.Name)
			}
			maps.Copy(entry, extra)
		}
		ml[m.Name] = entry
	}
	b, err := yaml.Marshal(ml) // yaml.v3 sorts map keys → deterministic output
	if err != nil {
		return errors.Wrap(err, "failed to marshal model_list.yaml")
	}
	return errors.Wrap(writeFileAtomic(filepath.Join(r.dir, "model_list.yaml"), b, envFileMode), "failed to render model_list.yaml")
}

func (r *Renderer) renderSkills(skills []*models.AdreSkill) error {
	skillsDir := filepath.Join(r.dir, "skills")
	if err := os.MkdirAll(skillsDir, dirMode); err != nil { //nolint:noinlineerr
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
		err := os.MkdirAll(dir, dirMode)
		if err != nil {
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
	err := os.WriteFile(tmp, data, perm)
	if err != nil {
		return err
	}
	err = os.Rename(tmp, path)
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
