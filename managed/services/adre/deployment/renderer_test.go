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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func testRenderer(t *testing.T) *Renderer {
	t.Helper()
	return NewRenderer(nil, t.TempDir(), logrus.WithField("test", "renderer"))
}

func TestLoadBuiltinSkills(t *testing.T) {
	t.Parallel()
	skills, err := loadBuiltinSkills()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(skills), 15, "expected the shipped skills to be embedded")

	byName := map[string]builtinSkill{}
	for _, s := range skills {
		assert.NotEmpty(t, s.name)
		assert.NotEmpty(t, s.body, "skill %q has empty body", s.name)
		byName[s.name] = s
	}
	g, ok := byName["general"]
	require.True(t, ok, "general skill must be embedded")
	assert.Contains(t, g.description, "workload", "general description parsed from frontmatter")
	assert.Contains(t, g.body, "## ", "general body kept verbatim")

	_, hasPG := byName["postgresql-query-analysis"]
	_, hasMongo := byName["mongodb-query-analysis"]
	assert.True(t, hasPG && hasMongo, "new query-analysis skills must ship")
}

func TestFrontmatterDescription(t *testing.T) {
	t.Parallel()
	content := "---\nname: x\ndescription: hello there\n---\n\n# Body\n"
	assert.Equal(t, "hello there", frontmatterDescription(content))
	assert.Empty(t, frontmatterDescription("# no frontmatter\n"))
}

func TestRenderModelList(t *testing.T) {
	t.Parallel()
	r := testRenderer(t)
	mdls := []*models.AdreModel{
		{Name: "openai/gpt-4.1", LitellmModel: "openai/gpt-4.1", APIKey: "sk-literal"},
		{Name: "anthropic-opus", LitellmModel: "anthropic/claude-opus-4-5", APIBase: "https://api.anthropic.com"},
		{Name: "local-llama", LitellmModel: "ollama_chat/llama3", APIBase: "http://ollama:11434", ExtraParams: "temperature: 1\nnum_ctx: 8192"},
	}
	require.NoError(t, r.renderModelList(mdls))

	b, err := os.ReadFile(filepath.Join(r.dir, "model_list.yaml"))
	require.NoError(t, err)
	out := string(b)
	assert.Contains(t, out, "api_key: sk-literal", "provider key rendered literally (no env var)")
	assert.Contains(t, out, "model: openai/gpt-4.1")
	assert.Contains(t, out, "api_base: https://api.anthropic.com")
	// Local-model extra params merge into the model entry.
	assert.Contains(t, out, "model: ollama_chat/llama3")
	assert.Contains(t, out, "api_base: http://ollama:11434")
	assert.Contains(t, out, "temperature: 1")
	assert.Contains(t, out, "num_ctx: 8192")

	info, err := os.Stat(filepath.Join(r.dir, "model_list.yaml"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "model_list.yaml holds secrets → 0600")
}

func TestDefaultConfigEmbedAndPlaceholder(t *testing.T) {
	t.Parallel()
	// The shipped default config.yaml must be embedded, carry the token placeholder (not a real
	// glsa_ token), and define the PMM toolsets.
	assert.NotEmpty(t, defaultConfigYAML, "default config.yaml must be embedded")
	assert.Contains(t, defaultConfigYAML, grafanaTokenPlaceholder)
	assert.NotContains(t, defaultConfigYAML, "glsa_", "no real Grafana token may ship in the template")
	assert.Contains(t, defaultConfigYAML, "pmm-inventory", "default config must define the PMM toolsets")
}

func TestRenderConfigYAMLSubstitutesToken(t *testing.T) {
	t.Parallel()
	r := testRenderer(t)
	// Mimic Render's substitution + write for a config holding the placeholder.
	const cfg = "model: openai/gpt-4.1\ntoolsets:\n  grafana/dashboards:\n    config:\n      api_key: __PMM_GRAFANA_TOKEN__\n"
	out := strings.ReplaceAll(cfg, grafanaTokenPlaceholder, "glsa_minted_123")
	require.NoError(t, writeFileAtomic(filepath.Join(r.dir, "config.yaml"), []byte(out), configFileMode))

	b, err := os.ReadFile(filepath.Join(r.dir, "config.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(b), "api_key: glsa_minted_123")
	assert.NotContains(t, string(b), grafanaTokenPlaceholder)
}

func TestRenderEnv(t *testing.T) {
	t.Parallel()
	r := testRenderer(t)
	require.NoError(t, r.renderEnv(&models.AdreProvisioning{
		PMMURL: "https://pmm-server:8443", PMMSAToken: "tok123", HolmesAPIKey: "key456",
	}))
	b, err := os.ReadFile(filepath.Join(r.dir, ".env"))
	require.NoError(t, err)
	out := string(b)
	assert.Contains(t, out, "PMM_URL=https://pmm-server:8443")
	assert.Contains(t, out, "PMM_API_TOKEN=tok123")
	assert.Contains(t, out, "HOLMES_API_KEY=key456")

	info, err := os.Stat(filepath.Join(r.dir, ".env"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestRenderSkills(t *testing.T) {
	t.Parallel()
	r := testRenderer(t)
	skillsDir := filepath.Join(r.dir, "skills")

	// Pre-existing stale skill that should be removed.
	require.NoError(t, os.MkdirAll(filepath.Join(skillsDir, "stale"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "stale", "SKILL.md"), []byte("old"), 0o644))

	skills := []*models.AdreSkill{
		{Name: "general", Body: "---\nname: general\n---\n\n# General\n", Enabled: true},
		{Name: "../escape", Body: "evil", Enabled: true}, // must be skipped (path traversal)
	}
	require.NoError(t, r.renderSkills(skills))

	b, err := os.ReadFile(filepath.Join(skillsDir, "general", "SKILL.md"))
	require.NoError(t, err)
	assert.Contains(t, string(b), "# General")

	_, err = os.Stat(filepath.Join(skillsDir, "stale"))
	assert.True(t, os.IsNotExist(err), "stale skill dir must be removed")
}

func TestValidSkillName(t *testing.T) {
	t.Parallel()
	assert.True(t, validSkillName("mysql-availability"))
	assert.True(t, validSkillName("postgresql_query"))
	assert.False(t, validSkillName("../escape"))
	assert.False(t, validSkillName("a/b"))
	assert.False(t, validSkillName(""))
}

func TestWriteFileAtomic(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	require.NoError(t, writeFileAtomic(path, []byte("hello"), 0o600))
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(b))
	_, err = os.Stat(path + ".tmp")
	assert.True(t, os.IsNotExist(err), "temp file must be renamed away")
}
