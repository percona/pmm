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

// Package deployment renders PMM-managed HolmesGPT/ADRE config (config.yaml, model_list.yaml,
// .env, skills) from the DB to the shared config directory, and provisions the PMM↔Holmes secrets.
package deployment

import (
	"embed"
	"io/fs"
	"strings"

	"github.com/pkg/errors" //nolint:depguard
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/models"
)

// builtinSkillsFS holds the SKILL.md tree shipped with PMM, seeded into adre_skills on first run.
//
//go:embed builtin_skills
var builtinSkillsFS embed.FS

const builtinSkillsDir = "builtin_skills"

// defaultConfigYAML is the shipped HolmesGPT config.yaml (PMM toolsets), seeded into
// adre_holmes_config on first run. The Grafana service-account token is stored as
// grafanaTokenPlaceholder and substituted by the renderer with the minted PMM token.
//
//go:embed default_config.yaml
var defaultConfigYAML string

// grafanaTokenPlaceholder marks where the minted PMM service-account token is injected into
// config.yaml at render time (the prometheus/metrics toolset).
const grafanaTokenPlaceholder = "__PMM_GRAFANA_TOKEN__" //nolint:gosec

// systemActor is the updated_by of rows PMM wrote itself; an admin's save replaces it with their login.
const systemActor = "system"

// SyncDefaultConfig stores the shipped config.yaml on first run, and again when it changed and no admin has
// edited the stored one. It reports whether it wrote.
func SyncDefaultConfig(db *reform.DB) (bool, error) {
	cfg, err := models.GetAdreHolmesConfig(db)
	if err != nil {
		return false, err
	}
	seeded := strings.TrimSpace(cfg.ConfigYAML) != ""
	if seeded && (cfg.UpdatedBy != systemActor || cfg.ConfigYAML == defaultConfigYAML) {
		return false, nil
	}
	return true, models.SaveAdreHolmesConfig(db, defaultConfigYAML, systemActor)
}

type builtinSkill struct {
	name        string
	description string
	body        string // full SKILL.md content (frontmatter + markdown), written verbatim by the renderer
}

// frontmatterDescription extracts the `description` field from a SKILL.md YAML frontmatter block.
func frontmatterDescription(content string) string {
	s := strings.TrimLeft(strings.TrimPrefix(content, "\ufeff"), " \n")
	if !strings.HasPrefix(s, "---") {
		return ""
	}
	rest := s[len("---"):]
	before, _, ok := strings.Cut(rest, "\n---")
	if !ok {
		return ""
	}
	var fm struct {
		Description string `yaml:"description"`
	}
	err := yaml.Unmarshal([]byte(before), &fm)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(fm.Description)
}

// loadBuiltinSkills reads the embedded SKILL.md tree.
func loadBuiltinSkills() ([]builtinSkill, error) {
	entries, err := fs.ReadDir(builtinSkillsFS, builtinSkillsDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read embedded builtin skills")
	}
	var out []builtinSkill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		b, err := builtinSkillsFS.ReadFile(builtinSkillsDir + "/" + e.Name() + "/SKILL.md")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read embedded skill %q", e.Name())
		}
		content := string(b)
		out = append(out, builtinSkill{
			name:        e.Name(),
			description: frontmatterDescription(content),
			body:        content,
		})
	}
	return out, nil
}

// SyncBuiltinSkills keeps PMM-owned skills (source builtin) equal to the shipped ones (see planSkillSync).
// It reports whether anything changed.
func SyncBuiltinSkills(db *reform.DB) (bool, error) {
	shipped, err := loadBuiltinSkills()
	if err != nil {
		return false, err
	}
	existing, err := models.ListAdreSkills(db, false)
	if err != nil {
		return false, err
	}
	deleted, err := models.AdreSkillsDeletedByAdmin(db)
	if err != nil {
		return false, err
	}
	upserts, removals := planSkillSync(shipped, existing, deleted)
	if len(upserts) == 0 && len(removals) == 0 {
		return false, nil
	}
	err = db.InTransaction(func(tx *reform.TX) error {
		for _, s := range upserts {
			err := models.UpsertAdreSkill(tx, s)
			if err != nil {
				return err
			}
		}
		for _, name := range removals {
			err := models.DeleteAdreSkill(tx, name)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err == nil, err
}

// planSkillSync adds shipped skills that are missing unless an admin deleted them, refreshes changed PMM-owned ones
// (keeping enabled), and removes PMM-owned ones no longer shipped. Skills an admin edited have source user and are
// left alone.
func planSkillSync(shipped []builtinSkill, existing []*models.AdreSkill, deletedByAdmin map[string]bool) ([]*models.AdreSkill, []string) {
	byName := make(map[string]*models.AdreSkill, len(existing))
	for _, s := range existing {
		byName[s.Name] = s
	}
	upserts := make([]*models.AdreSkill, 0, len(shipped))
	shippedNames := make(map[string]bool, len(shipped))
	for _, s := range shipped {
		shippedNames[s.name] = true
		enabled := true
		cur := byName[s.name]
		switch {
		case cur == nil && deletedByAdmin[s.name]:
			continue
		case cur == nil:
		case cur.Source != models.AdreSkillSourceBuiltin, cur.Body == s.body && cur.Description == s.description:
			continue
		default:
			enabled = cur.Enabled
		}
		upserts = append(upserts, &models.AdreSkill{
			Name:        s.name,
			Description: s.description,
			Body:        s.body,
			Source:      models.AdreSkillSourceBuiltin,
			Enabled:     enabled,
			UpdatedBy:   systemActor,
		})
	}
	var removals []string
	for _, cur := range existing {
		if !shippedNames[cur.Name] && cur.Source == models.AdreSkillSourceBuiltin {
			removals = append(removals, cur.Name)
		}
	}
	return upserts, removals
}
