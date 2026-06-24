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

package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"github.com/pkg/errors" //nolint:depguard
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/utils/encryption"
)

// ADRE/HolmesGPT deployment config is the source of truth that PMM renders to the shared config
// directory (config.yaml, model_list.yaml, .env, skills/). These helpers use raw SQL (like
// settings_helpers.go) rather than reform models to avoid codegen for config-store tables.

// Skill source values.
const (
	AdreSkillSourceBuiltin = "builtin"
	AdreSkillSourceUser    = "user"
)

// AdreHolmesConfig is the singleton config.yaml store.
type AdreHolmesConfig struct {
	ConfigYAML string
	UpdatedAt  time.Time
	UpdatedBy  string
}

// AdreModel is one entry rendered into model_list.yaml. APIKey is a secret (masked on the API).
// The default chat/fast model is config.yaml's model:/fast_model:, not a flag here.
// ExtraParams is optional YAML of extra LiteLLM params (e.g. temperature, num_ctx) merged into the entry.
type AdreModel struct {
	ID           int64
	Name         string
	LitellmModel string
	APIBase      string
	APIKey       string
	ExtraParams  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AdreSkill is one SKILL.md rendered under skills/<name>/SKILL.md.
type AdreSkill struct {
	ID          int64
	Name        string
	Description string
	Body        string
	Source      string // builtin | user
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	UpdatedBy   string
}

// AdreProvisioning is the singleton holding generated/minted secrets, integration tokens, and render
// status. All secret fields are encrypted at rest (see DefaultAgentEncryptionColumnsV3) and masked on
// the API.
type AdreProvisioning struct {
	HolmesAPIKey          string
	PMMSAToken            string
	ServiceNowAPIKey      string
	ServiceNowClientToken string
	SlackBotToken         string
	SlackAppToken         string
	AlertWebhookSecret    string
	PMMSAID               int
	PMMURL                string
	LastRenderAt          *time.Time
	RenderStatus          string
	RestartRequired       bool
}

// AdreConfigAudit is one audit-log row for a deployment-config mutation.
type AdreConfigAudit struct {
	ID     int64
	Actor  string
	Action string
	Target string
	At     time.Time
	Diff   string
}

// GetAdreHolmesConfig returns the singleton config.yaml store (zero value when not yet set).
func GetAdreHolmesConfig(q reform.DBTX) (*AdreHolmesConfig, error) {
	var c AdreHolmesConfig
	err := q.QueryRow("SELECT config_yaml, updated_at, updated_by FROM adre_holmes_config WHERE id = TRUE").
		Scan(&c.ConfigYAML, &c.UpdatedAt, &c.UpdatedBy)
	if errors.Is(err, sql.ErrNoRows) {
		return &AdreHolmesConfig{}, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to select adre_holmes_config")
	}
	return &c, nil
}

// SaveAdreHolmesConfig upserts the singleton config.yaml store.
func SaveAdreHolmesConfig(q reform.DBTX, configYAML, updatedBy string) error {
	_, err := q.Exec(
		`INSERT INTO adre_holmes_config (id, config_yaml, updated_at, updated_by)
		 VALUES (TRUE, $1, NOW(), $2)
		 ON CONFLICT (id) DO UPDATE SET config_yaml = EXCLUDED.config_yaml, updated_at = NOW(), updated_by = EXCLUDED.updated_by`,
		configYAML, updatedBy,
	)
	return errors.Wrap(err, "failed to save adre_holmes_config")
}

// ListAdreModels returns all configured models ordered by name.
func ListAdreModels(q reform.DBTX) ([]*AdreModel, error) {
	rows, err := q.Query(
		`SELECT id, name, litellm_model, api_base, api_key, extra_params, created_at, updated_at
		 FROM adre_models ORDER BY name`,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select adre_models")
	}
	defer rows.Close() //nolint:errcheck

	var out []*AdreModel
	for rows.Next() {
		var m AdreModel
		if err := rows.Scan(&m.ID, &m.Name, &m.LitellmModel, &m.APIBase, &m.APIKey, &m.ExtraParams, &m.CreatedAt, &m.UpdatedAt); err != nil { //nolint:noinlineerr
			return nil, errors.Wrap(err, "failed to scan adre_models")
		}
		m.APIKey = decryptField("api_key", m.APIKey)
		out = append(out, &m)
	}
	return out, errors.Wrap(rows.Err(), "failed to iterate adre_models")
}

// UpsertAdreModel inserts or updates a model by name. An empty APIKey keeps the existing key.
func UpsertAdreModel(q reform.DBTX, m *AdreModel) error {
	if m.APIKey == "" {
		_, err := q.Exec(
			`INSERT INTO adre_models (name, litellm_model, api_base, extra_params, updated_at)
			 VALUES ($1, $2, $3, $4, NOW())
			 ON CONFLICT (name) DO UPDATE SET litellm_model = EXCLUDED.litellm_model, api_base = EXCLUDED.api_base,
			   extra_params = EXCLUDED.extra_params, updated_at = NOW()`,
			m.Name, m.LitellmModel, m.APIBase, m.ExtraParams,
		)
		return errors.Wrap(err, "failed to upsert adre_models")
	}
	apiKey, err := encryption.Encrypt(m.APIKey)
	if err != nil {
		return errors.Wrap(err, "failed to encrypt adre_models api_key")
	}
	_, err = q.Exec(
		`INSERT INTO adre_models (name, litellm_model, api_base, api_key, extra_params, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 ON CONFLICT (name) DO UPDATE SET litellm_model = EXCLUDED.litellm_model, api_base = EXCLUDED.api_base,
		   api_key = EXCLUDED.api_key, extra_params = EXCLUDED.extra_params, updated_at = NOW()`,
		m.Name, m.LitellmModel, m.APIBase, apiKey, m.ExtraParams,
	)
	return errors.Wrap(err, "failed to upsert adre_models")
}

// DeleteAdreModel removes a model by name.
func DeleteAdreModel(q reform.DBTX, name string) error {
	_, err := q.Exec("DELETE FROM adre_models WHERE name = $1", name)
	return errors.Wrap(err, "failed to delete adre_models")
}

// ListAdreSkills returns skills (optionally only enabled), ordered by name.
func ListAdreSkills(q reform.DBTX, onlyEnabled bool) ([]*AdreSkill, error) {
	query := `SELECT id, name, description, body, source, enabled, created_at, updated_at, updated_by FROM adre_skills`
	if onlyEnabled {
		query += " WHERE enabled = TRUE"
	}
	query += " ORDER BY name"
	rows, err := q.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select adre_skills")
	}
	defer rows.Close() //nolint:errcheck

	var out []*AdreSkill
	for rows.Next() {
		var s AdreSkill
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Body, &s.Source, &s.Enabled, &s.CreatedAt, &s.UpdatedAt, &s.UpdatedBy); err != nil { //nolint:noinlineerr
			return nil, errors.Wrap(err, "failed to scan adre_skills")
		}
		out = append(out, &s)
	}
	return out, errors.Wrap(rows.Err(), "failed to iterate adre_skills")
}

// GetAdreSkill returns one skill by name, or nil when absent.
func GetAdreSkill(q reform.DBTX, name string) (*AdreSkill, error) {
	var s AdreSkill
	err := q.QueryRow(
		`SELECT id, name, description, body, source, enabled, created_at, updated_at, updated_by
		 FROM adre_skills WHERE name = $1`, name,
	).
		Scan(&s.ID, &s.Name, &s.Description, &s.Body, &s.Source, &s.Enabled, &s.CreatedAt, &s.UpdatedAt, &s.UpdatedBy)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil //nolint:nilnil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to select adre_skills")
	}
	return &s, nil
}

// UpsertAdreSkill inserts or updates a skill by name.
func UpsertAdreSkill(q reform.DBTX, s *AdreSkill) error {
	source := s.Source
	if source == "" {
		source = AdreSkillSourceUser
	}
	_, err := q.Exec(
		`INSERT INTO adre_skills (name, description, body, source, enabled, updated_at, updated_by)
		 VALUES ($1, $2, $3, $4, $5, NOW(), $6)
		 ON CONFLICT (name) DO UPDATE SET description = EXCLUDED.description, body = EXCLUDED.body,
		   enabled = EXCLUDED.enabled, updated_at = NOW(), updated_by = EXCLUDED.updated_by`,
		s.Name, s.Description, s.Body, source, s.Enabled, s.UpdatedBy,
	)
	return errors.Wrap(err, "failed to upsert adre_skills")
}

// DeleteAdreSkill removes a skill by name.
func DeleteAdreSkill(q reform.DBTX, name string) error {
	_, err := q.Exec("DELETE FROM adre_skills WHERE name = $1", name)
	return errors.Wrap(err, "failed to delete adre_skills")
}

// CountAdreSkills returns the number of skill rows (used to decide first-run seeding).
func CountAdreSkills(q reform.DBTX) (int, error) {
	var n int
	err := q.QueryRow("SELECT COUNT(*) FROM adre_skills").Scan(&n)
	return n, errors.Wrap(err, "failed to count adre_skills")
}

// GetAdreProvisioning returns the singleton provisioning row (zero value when not yet set).
func GetAdreProvisioning(q reform.DBTX) (*AdreProvisioning, error) {
	var p AdreProvisioning
	var lastRender sql.NullTime
	err := q.QueryRow(
		`SELECT holmes_api_key, pmm_sa_token, servicenow_api_key, servicenow_client_token, slack_bot_token, slack_app_token,
		        alert_webhook_secret, pmm_sa_id, pmm_url, last_render_at, render_status, restart_required
		 FROM adre_provisioning WHERE id = TRUE`,
	).
		Scan(&p.HolmesAPIKey, &p.PMMSAToken, &p.ServiceNowAPIKey, &p.ServiceNowClientToken, &p.SlackBotToken, &p.SlackAppToken,
			&p.AlertWebhookSecret, &p.PMMSAID, &p.PMMURL, &lastRender, &p.RenderStatus, &p.RestartRequired)
	if errors.Is(err, sql.ErrNoRows) {
		return &AdreProvisioning{}, nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to select adre_provisioning")
	}
	if lastRender.Valid {
		p.LastRenderAt = &lastRender.Time
	}
	p.HolmesAPIKey = decryptField("holmes_api_key", p.HolmesAPIKey)
	p.PMMSAToken = decryptField("pmm_sa_token", p.PMMSAToken)
	p.ServiceNowAPIKey = decryptField("servicenow_api_key", p.ServiceNowAPIKey)
	p.ServiceNowClientToken = decryptField("servicenow_client_token", p.ServiceNowClientToken)
	p.SlackBotToken = decryptField("slack_bot_token", p.SlackBotToken)
	p.SlackAppToken = decryptField("slack_app_token", p.SlackAppToken)
	p.AlertWebhookSecret = decryptField("alert_webhook_secret", p.AlertWebhookSecret)
	return &p, nil
}

// EnsureAlertWebhookSecret returns the alert-webhook shared secret, generating and persisting one
// (encrypted) on first use. The secret authenticates Grafana's alert webhook to PMM.
func EnsureAlertWebhookSecret(q reform.DBTX) (string, error) {
	p, err := GetAdreProvisioning(q)
	if err != nil {
		return "", err
	}
	if p.AlertWebhookSecret != "" {
		return p.AlertWebhookSecret, nil
	}
	secret, err := generateSecretHex(32)
	if err != nil {
		return "", err
	}
	p.AlertWebhookSecret = secret
	if err := SaveAdreProvisioning(q, p); err != nil { //nolint:noinlineerr
		return "", err
	}
	return secret, nil
}

// generateSecretHex returns a cryptographically random hex string of n bytes (2n hex chars).
func generateSecretHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil { //nolint:noinlineerr
		return "", errors.Wrap(err, "failed to generate random secret")
	}
	return hex.EncodeToString(b), nil
}

// SaveAdreProvisioning upserts the singleton provisioning row.
func SaveAdreProvisioning(q reform.DBTX, p *AdreProvisioning) error {
	var lastRender any
	if p.LastRenderAt != nil {
		lastRender = *p.LastRenderAt
	}
	// Order matches the placeholders $1..$7 below.
	enc, err := encryptSecrets(
		p.HolmesAPIKey, p.PMMSAToken, p.ServiceNowAPIKey, p.ServiceNowClientToken, p.SlackBotToken, p.SlackAppToken,
		p.AlertWebhookSecret,
	)
	if err != nil {
		return errors.Wrap(err, "failed to encrypt adre_provisioning secrets")
	}
	_, err = q.Exec(
		`INSERT INTO adre_provisioning (id, holmes_api_key, pmm_sa_token, servicenow_api_key, servicenow_client_token,
		        slack_bot_token, slack_app_token, alert_webhook_secret, pmm_sa_id, pmm_url, last_render_at, render_status, restart_required)
		 VALUES (TRUE, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 ON CONFLICT (id) DO UPDATE SET holmes_api_key = EXCLUDED.holmes_api_key, pmm_sa_token = EXCLUDED.pmm_sa_token,
		   servicenow_api_key = EXCLUDED.servicenow_api_key, servicenow_client_token = EXCLUDED.servicenow_client_token,
		   slack_bot_token = EXCLUDED.slack_bot_token, slack_app_token = EXCLUDED.slack_app_token,
		   alert_webhook_secret = EXCLUDED.alert_webhook_secret,
		   pmm_sa_id = EXCLUDED.pmm_sa_id, pmm_url = EXCLUDED.pmm_url, last_render_at = EXCLUDED.last_render_at,
		   render_status = EXCLUDED.render_status, restart_required = EXCLUDED.restart_required`,
		enc[0], enc[1], enc[2], enc[3], enc[4], enc[5], enc[6], p.PMMSAID, p.PMMURL, lastRender, p.RenderStatus, p.RestartRequired,
	)
	return errors.Wrap(err, "failed to save adre_provisioning")
}

// encryptSecrets encrypts each value for at-rest storage, aborting on the first error so a secret is
// never persisted in plaintext. The returned slice is in the same order as the arguments.
// encryption.Encrypt is a no-op on the empty string.
func encryptSecrets(values ...string) ([]string, error) {
	out := make([]string, len(values))
	for i, v := range values {
		enc, err := encryption.Encrypt(v)
		if err != nil {
			return nil, err
		}
		out[i] = enc
	}
	return out, nil
}

// decryptField decrypts an at-rest secret. On error (e.g. a legacy plaintext value) it returns the
// original value so reads degrade gracefully, logging only the field name — never the encryption.Decrypt
// error, which embeds the raw input. encryption.Decrypt is a no-op on the empty string.
func decryptField(field, s string) string {
	out, err := encryption.Decrypt(s)
	if err != nil {
		logrus.Warnf("adre: failed to decrypt %s", field)
	}
	return out
}

// AdreSecretsConfigured reports whether the ServiceNow tokens and the Slack tokens are set, checking
// the encrypted columns for non-emptiness without decrypting them (Encrypt maps "" to "" and non-empty
// to non-empty ciphertext, so emptiness is preserved). Used for the masked settings API.
func AdreSecretsConfigured(q reform.DBTX) (serviceNow, slack bool, err error) {
	err = q.QueryRow(
		`SELECT servicenow_api_key <> '' AND servicenow_client_token <> '',
		        slack_bot_token <> '' AND slack_app_token <> ''
		 FROM adre_provisioning WHERE id = TRUE`,
	).Scan(&serviceNow, &slack)
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}
	return serviceNow, slack, errors.Wrap(err, "failed to read adre_provisioning secret flags")
}

// InsertAdreConfigAudit appends one audit-log row.
func InsertAdreConfigAudit(q reform.DBTX, actor, action, target, diff string) error {
	_, err := q.Exec(
		`INSERT INTO adre_config_audit (actor, action, target, diff) VALUES ($1, $2, $3, $4)`,
		actor, action, target, diff,
	)
	return errors.Wrap(err, "failed to insert adre_config_audit")
}
