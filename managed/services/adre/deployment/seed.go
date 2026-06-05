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

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/models"
)

// builtinSkillsFS holds the SKILL.md tree shipped with PMM, seeded into adre_skills on first run.
//
//go:embed builtin_skills
var builtinSkillsFS embed.FS

const builtinSkillsDir = "builtin_skills"

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
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return ""
	}
	var fm struct {
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
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

// SeedBuiltinSkills inserts the shipped skills into adre_skills when the table is empty (first run).
func SeedBuiltinSkills(db *reform.DB) error {
	n, err := models.CountAdreSkills(db)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	skills, err := loadBuiltinSkills()
	if err != nil {
		return err
	}
	return db.InTransaction(func(tx *reform.TX) error {
		for _, s := range skills {
			if err := models.UpsertAdreSkill(tx, &models.AdreSkill{ //nolint:noinlineerr
				Name:        s.name,
				Description: s.description,
				Body:        s.body,
				Source:      models.AdreSkillSourceBuiltin,
				Enabled:     true,
				UpdatedBy:   "system",
			}); err != nil {
				return err
			}
		}
		return nil
	})
}
