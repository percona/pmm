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

package adre

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/adre/deployment"
	"github.com/percona/pmm/managed/utils/validators"
)

// incomingAuthContext bridges an HTTP request's auth into the gRPC-metadata context that
// grafana.Client.CreateServiceAccount expects (auth.GetHeadersFromContext reads incoming metadata).
func incomingAuthContext(r *http.Request) context.Context {
	md := metadata.MD{}
	if v := r.Header.Get("Authorization"); v != "" {
		md.Append("authorization", v)
	}
	if v := r.Header.Get("Cookie"); v != "" {
		md.Append("grpcgateway-cookie", v)
	}
	return metadata.NewIncomingContext(r.Context(), md)
}

// adreConfigDir returns the dedicated config dir PMM renders into (mirrors Holmes /config mount).
func adreConfigDir() string {
	if d := strings.TrimSpace(os.Getenv("ADRE_CONFIG_DIR")); d != "" {
		return d
	}
	return deployment.DefaultConfigDir
}

func (h *Handlers) renderer() *deployment.Renderer {
	return deployment.NewRenderer(h.db, adreConfigDir(), h.l)
}

func (h *Handlers) provisioner() *deployment.Provisioner {
	return deployment.NewProvisioner(h.db, h.grafana, h.l)
}

// requireAdmin enforces admin-only access server-side (the UI gate is secondary). Returns the
// caller login on success.
func (h *Handlers) requireAdmin(w http.ResponseWriter, r *http.Request) (string, bool) {
	headers := grafanaAuthHeadersFromRequest(r)
	isAdmin, err := h.grafana.IsCurrentUserAdmin(r.Context(), headers)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "Authentication required")
		return "", false
	}
	if !isAdmin {
		writeJSONError(w, http.StatusForbidden, "Admin privileges required")
		return "", false
	}
	login, _ := h.grafana.GetCurrentUserLogin(r.Context(), headers)
	return login, true
}

// --- response shapes (secrets masked) ---.

type deploymentModelView struct {
	Name          string `json:"name"`
	LitellmModel  string `json:"litellm_model"`
	APIBase       string `json:"api_base"`
	KeyConfigured bool   `json:"key_configured"`
	ExtraParams   string `json:"extra_params"`
}

type deploymentSkillView struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Body        string `json:"body"`
	Source      string `json:"source"`
	Enabled     bool   `json:"enabled"`
}

type deploymentProvisioningView struct {
	PMMURL             string     `json:"pmm_url"`
	TokenConfigured    bool       `json:"token_configured"`
	HolmesKeyConfig    bool       `json:"holmes_api_key_configured"`
	RestartRequired    bool       `json:"restart_required"`
	EnvRestartRequired bool       `json:"env_restart_required"`
	LastRenderAt       *time.Time `json:"last_render_at,omitempty"`
	RenderStatus       string     `json:"render_status"`
	ConfigDir          string     `json:"config_dir"`
}

type deploymentResponse struct {
	ConfigYAML   string                     `json:"config_yaml"`
	Models       []deploymentModelView      `json:"models"`
	Skills       []deploymentSkillView      `json:"skills"`
	Provisioning deploymentProvisioningView `json:"provisioning"`
}

// GetDeployment handles GET /v1/adre/deployment.
func (h *Handlers) GetDeployment(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	resp, err := h.buildDeploymentResponse()
	if err != nil {
		h.l.Errorf("GetDeployment: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load deployment config")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handlers) buildDeploymentResponse() (*deploymentResponse, error) {
	cfg, err := models.GetAdreHolmesConfig(h.db)
	if err != nil {
		return nil, err
	}
	mdls, err := models.ListAdreModels(h.db)
	if err != nil {
		return nil, err
	}
	skills, err := models.ListAdreSkills(h.db, false)
	if err != nil {
		return nil, err
	}
	prov, err := models.GetAdreProvisioning(h.db)
	if err != nil {
		return nil, err
	}

	// Non-nil slices so JSON renders [] (not null) for clients that map() over them.
	resp := &deploymentResponse{
		ConfigYAML: cfg.ConfigYAML,
		Models:     []deploymentModelView{},
		Skills:     []deploymentSkillView{},
	}
	for _, m := range mdls {
		resp.Models = append(resp.Models, deploymentModelView{
			Name: m.Name, LitellmModel: m.LitellmModel, APIBase: m.APIBase,
			KeyConfigured: m.APIKey != "", ExtraParams: m.ExtraParams,
		})
	}
	for _, s := range skills {
		resp.Skills = append(resp.Skills, deploymentSkillView{
			Name: s.Name, Description: s.Description, Body: s.Body, Source: s.Source, Enabled: s.Enabled,
		})
	}
	resp.Provisioning = deploymentProvisioningView{
		PMMURL: prov.PMMURL, TokenConfigured: prov.PMMSAToken != "", HolmesKeyConfig: prov.HolmesAPIKey != "",
		RestartRequired: prov.RestartRequired, EnvRestartRequired: prov.EnvRestartRequired,
		LastRenderAt: prov.LastRenderAt, RenderStatus: prov.RenderStatus, ConfigDir: adreConfigDir(),
	}
	return resp, nil
}

// PutDeploymentConfig handles PUT /v1/adre/deployment/config (raw config.yaml).
func (h *Handlers) PutDeploymentConfig(w http.ResponseWriter, r *http.Request) {
	login, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		ConfigYAML string `json:"config_yaml"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	// Validate it parses as YAML before persisting.
	var probe any
	err := yaml.Unmarshal([]byte(body.ConfigYAML), &probe)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "config.yaml is not valid YAML: "+err.Error())
		return
	}
	err = models.SaveAdreHolmesConfig(h.db, body.ConfigYAML, login)
	if err != nil {
		h.l.Errorf("SaveAdreHolmesConfig: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to save config.yaml")
		return
	}
	_ = models.InsertAdreConfigAudit(h.db, login, "update", "config.yaml", "")
	reload, _ := h.renderAndReload(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"saved": true, "reload": reload})
}

// PutDeploymentModels handles PUT /v1/adre/deployment/models (upsert; empty api_key keeps existing).
func (h *Handlers) PutDeploymentModels(w http.ResponseWriter, r *http.Request) {
	login, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		Models []struct {
			Name         string `json:"name"`
			LitellmModel string `json:"litellm_model"`
			APIBase      string `json:"api_base"`
			APIKey       string `json:"api_key"`
			ExtraParams  string `json:"extra_params"`
		} `json:"models"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	for _, m := range body.Models {
		name := strings.TrimSpace(m.Name)
		if name == "" || strings.TrimSpace(m.LitellmModel) == "" {
			writeJSONError(w, http.StatusBadRequest, "each model requires name and litellm_model")
			return
		}
		if !validModelName(name) {
			writeJSONError(w, http.StatusBadRequest, "model name may contain only letters, digits, '.', '-' or '_'")
			return
		}
		// Extra params must be a YAML mapping (e.g. "temperature: 1") so the renderer can merge them.
		if strings.TrimSpace(m.ExtraParams) != "" {
			probe := map[string]any{}
			err := yaml.Unmarshal([]byte(m.ExtraParams), &probe)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "model "+name+": extra params must be valid YAML key: value pairs: "+err.Error())
				return
			}
		}
		if err := models.UpsertAdreModel(h.db, &models.AdreModel{ //nolint:noinlineerr
			Name: name, LitellmModel: m.LitellmModel, APIBase: m.APIBase, APIKey: m.APIKey, ExtraParams: m.ExtraParams,
		}); err != nil {
			h.l.Errorf("UpsertAdreModel: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "Failed to save model")
			return
		}
	}
	_ = models.InsertAdreConfigAudit(h.db, login, "update", "models", "")
	reload, _ := h.renderAndReload(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"saved": len(body.Models), "reload": reload})
}

// DeleteDeploymentModel handles DELETE /v1/adre/deployment/models/{name}.
func (h *Handlers) DeleteDeploymentModel(w http.ResponseWriter, r *http.Request, name string) {
	login, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	err := models.DeleteAdreModel(h.db, name)
	if err != nil {
		h.l.Errorf("DeleteAdreModel: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to delete model")
		return
	}
	_ = models.InsertAdreConfigAudit(h.db, login, "delete", "model:"+name, "")
	reload, _ := h.renderAndReload(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "reload": reload})
}

// PutDeploymentProvisioning handles PUT /v1/adre/deployment/provisioning (set the PMM URL Holmes uses).
func (h *Handlers) PutDeploymentProvisioning(w http.ResponseWriter, r *http.Request) {
	login, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		PMMURL string `json:"pmm_url"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	prov, err := models.GetAdreProvisioning(h.db)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load provisioning")
		return
	}
	pmmURL := strings.TrimSpace(body.PMMURL)
	if pmmURL != "" {
		if _, err := validators.RequireSecureServiceURL(pmmURL, allowInsecureADREURLs()); err != nil { //nolint:noinlineerr
			writeJSONError(w, http.StatusBadRequest, "pmm_url: "+err.Error())
			return
		}
	}
	if prov.PMMURL == pmmURL {
		// Unchanged — don't force a spurious restart requirement for a no-op save.
		writeJSON(w, http.StatusOK, map[string]any{"saved": true})
		return
	}
	prov.PMMURL = pmmURL
	// Persist the restart flags with the URL so they survive even if the render below fails — PMM_URL
	// lives in .env, which a hot-reload cannot pick up.
	prov.RestartRequired = true
	prov.EnvRestartRequired = true
	if err := models.SaveAdreProvisioning(h.db, prov); err != nil { //nolint:noinlineerr
		h.l.Errorf("SaveAdreProvisioning: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to save PMM URL")
		return
	}
	// Render the new PMM_URL into .env now, so recreating the container actually applies it — the
	// banner tells the operator to restart, and .env must already hold the new value.
	if err := h.applyRender("provisioned"); err != nil { //nolint:noinlineerr
		writeJSONError(w, http.StatusInternalServerError, "Render failed: "+err.Error())
		return
	}
	_ = models.InsertAdreConfigAudit(h.db, login, "update", "pmm_url", "")
	writeJSON(w, http.StatusOK, map[string]any{"saved": true, "restart_required": true})
}

// PutDeploymentSkill handles POST/PUT /v1/adre/deployment/skills[/{name}].
func (h *Handlers) PutDeploymentSkill(w http.ResponseWriter, r *http.Request, name string) {
	login, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Body        string `json:"body"`
		Enabled     *bool  `json:"enabled"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if name == "" {
		name = strings.TrimSpace(body.Name)
	}
	if !validSkillNameAPI(name) {
		writeJSONError(w, http.StatusBadRequest, "invalid skill name (use letters, digits, '-' or '_')")
		return
	}
	existing, err := models.GetAdreSkill(h.db, name)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load skill")
		return
	}
	enabled := true
	source := models.AdreSkillSourceUser
	if existing != nil {
		enabled = existing.Enabled
		source = existing.Source
	}
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	if err := models.UpsertAdreSkill(h.db, &models.AdreSkill{ //nolint:noinlineerr
		Name: name, Description: body.Description, Body: body.Body, Source: source, Enabled: enabled, UpdatedBy: login,
	}); err != nil {
		h.l.Errorf("UpsertAdreSkill: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to save skill")
		return
	}
	_ = models.InsertAdreConfigAudit(h.db, login, "update", "skill:"+name, "")
	reload, _ := h.renderAndReload(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"saved": true, "reload": reload})
}

// DeleteDeploymentSkill handles DELETE /v1/adre/deployment/skills/{name}.
func (h *Handlers) DeleteDeploymentSkill(w http.ResponseWriter, r *http.Request, name string) {
	login, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	err := models.DeleteAdreSkill(h.db, name)
	if err != nil {
		h.l.Errorf("DeleteAdreSkill: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to delete skill")
		return
	}
	_ = models.InsertAdreConfigAudit(h.db, login, "delete", "skill:"+name, "")
	reload, _ := h.renderAndReload(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "reload": reload})
}

// PostDeploymentProvision handles POST /v1/adre/deployment/provision (mint token/key, render .env).
func (h *Handlers) PostDeploymentProvision(w http.ResponseWriter, r *http.Request) {
	login, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	if _, err := h.provisioner().EnsureProvisioned(incomingAuthContext(r), h.resolvePMMURL()); err != nil { //nolint:contextcheck,noinlineerr
		h.l.Errorf("EnsureProvisioned: %v", err)
		writeJSONError(w, http.StatusBadGateway, "Provisioning failed: "+err.Error())
		return
	}
	err := h.applyRender("provisioned")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Render failed: "+err.Error())
		return
	}
	_ = models.InsertAdreConfigAudit(h.db, login, "provision", "deployment", "")
	writeJSON(w, http.StatusOK, map[string]any{"provisioned": true, "restart_required": true})
}

// PostDeploymentApply handles POST /v1/adre/deployment/apply — render everything to disk and hot-reload
// Holmes. A successful reload applies config/model/skill changes with no restart; a pending .env change
// still needs a container recreate.
func (h *Handlers) PostDeploymentApply(w http.ResponseWriter, r *http.Request) {
	login, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	// Ensure secrets exist so the rendered .env is complete.
	if _, err := h.provisioner().EnsureProvisioned(incomingAuthContext(r), h.resolvePMMURL()); err != nil { //nolint:contextcheck,noinlineerr
		h.l.Warnf("EnsureProvisioned during apply: %v", err)
	}
	// Render to disk and hot-reload Holmes. For Apply the render IS the action, so a render failure
	// is fatal (nothing reached disk); a failed hot-reload is reported in the outcome, not the error.
	reload, err := h.renderAndReload(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Render failed: "+err.Error())
		return
	}
	_ = models.InsertAdreConfigAudit(h.db, login, "apply", "deployment", "")
	writeJSON(w, http.StatusOK, map[string]any{"applied": true, "reload": reload})
}

// PostDeploymentRestarted handles POST /v1/adre/deployment/restarted — the operator's acknowledgement
// that the HolmesGPT container was recreated. It clears the restart requirement, including the env one
// (PMM_URL / secrets) that a hot-reload cannot satisfy and that config saves therefore preserve.
func (h *Handlers) PostDeploymentRestarted(w http.ResponseWriter, r *http.Request) {
	login, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	if err := h.updateProvisioning(func(prov *models.AdreProvisioning) {
		prov.RestartRequired = false
		prov.EnvRestartRequired = false
	}); err != nil {
		h.l.Errorf("PostDeploymentRestarted: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to clear restart flag")
		return
	}
	_ = models.InsertAdreConfigAudit(h.db, login, "restarted", "deployment", "")
	writeJSON(w, http.StatusOK, map[string]any{"restart_required": false})
}

// applyRender renders to disk and records render status, keeping restart_required set. Used by the
// provisioning path, whose .env changes (PMM_URL / minted secrets) cannot be hot-reloaded and so
// always require a container restart — hence env_restart_required is set too.
func (h *Handlers) applyRender(status string) error {
	if err := h.renderer().Render(); err != nil { //nolint:noinlineerr
		_ = h.saveProvStatus("error: "+err.Error(), true)
		return err
	}
	now := time.Now()
	return h.updateProvisioning(func(prov *models.AdreProvisioning) {
		prov.LastRenderAt = &now
		prov.RenderStatus = status
		prov.RestartRequired = true
		prov.EnvRestartRequired = true
	})
}

func (h *Handlers) saveProvStatus(status string, restartRequired bool) error {
	return h.updateProvisioning(func(prov *models.AdreProvisioning) {
		prov.RenderStatus = status
		prov.RestartRequired = restartRequired
	})
}

// updateProvisioning loads the singleton provisioning row, applies mutate, and saves it — the common
// load/mutate/save shared by the render and restart-flag bookkeeping.
func (h *Handlers) updateProvisioning(mutate func(*models.AdreProvisioning)) error {
	prov, err := models.GetAdreProvisioning(h.db)
	if err != nil {
		return err
	}
	mutate(prov)
	return models.SaveAdreProvisioning(h.db, prov)
}

// reloadTimeout bounds the best-effort HolmesGPT hot-reload. It runs on a context detached from the
// request (see reloadHolmes) so a client disconnect can't abort it, but is still capped so a slow or
// unreachable Holmes never blocks the response for long.
const reloadTimeout = 30 * time.Second

// reloadOutcome summarizes a render + HolmesGPT hot-reload attempt, returned to the UI so it can
// show "reloaded" versus "restart still needed".
type reloadOutcome struct {
	Reloaded bool   `json:"reloaded"`
	Detail   string `json:"detail,omitempty"`
	Error    string `json:"error,omitempty"`
	// RestartRequired is the post-operation restart state: a hot-reload can succeed while a pending
	// .env change still needs a restart, so the UI keys its toast off this, not off Reloaded alone.
	RestartRequired bool `json:"restart_required"`
}

// renderAndReload renders the current DB config to disk and asks HolmesGPT to hot-reload it, so
// config/model/skill changes apply without recreating the container (POST /api/admin/reload).
//
// It returns a non-nil error only when the render to disk fails (the hard failure — nothing reached
// disk); a failed hot-reload is reported via the reloadOutcome, not the error. Callers whose real
// action is a DB write (config/model/skill saves) can ignore the error and surface the outcome;
// Apply, whose only action is the render, should treat the error as fatal.
//
// restart_required is set when the hot-reload did not apply the change, and is otherwise reduced to
// env_restart_required: a pending .env change (PMM_URL / secrets) survives config saves because a
// hot-reload cannot pick up env-delivered values — only a container restart can.
func (h *Handlers) renderAndReload(ctx context.Context) (reloadOutcome, error) {
	if err := h.renderer().Render(); err != nil { //nolint:noinlineerr
		h.l.Errorf("renderAndReload: render failed: %v", err)
		_ = h.saveProvStatus("error: "+err.Error(), true)
		return reloadOutcome{Error: "render failed: " + err.Error()}, err
	}

	outcome, status, reloaded := h.reloadHolmes(ctx)

	now := time.Now()
	restartRequired := true
	if err := h.updateProvisioning(func(prov *models.AdreProvisioning) {
		prov.LastRenderAt = &now
		prov.RenderStatus = status
		if reloaded {
			// Change is live; keep only a pending env restart, which a hot-reload cannot satisfy.
			prov.RestartRequired = prov.EnvRestartRequired
		} else {
			prov.RestartRequired = true
		}
		restartRequired = prov.RestartRequired
	}); err != nil {
		h.l.Errorf("renderAndReload: persist provisioning: %v", err)
	}
	outcome.RestartRequired = restartRequired
	return outcome, nil
}

// reloadHolmes performs the best-effort hot-reload and returns the API outcome, the render status to
// persist, and whether the reload actually succeeded. It runs on a context detached from the request
// so a client disconnect (e.g. the browser navigating away) cannot cancel the reload.
func (h *Handlers) reloadHolmes(ctx context.Context) (reloadOutcome, string, bool) {
	settings, err := models.GetSettings(h.db)
	if err != nil {
		h.l.Errorf("reloadHolmes: get settings: %v", err)
		return reloadOutcome{Error: "could not load PMM settings to reach HolmesGPT: " + err.Error()}, "rendered", false
	}
	if strings.TrimSpace(settings.GetAdreURL()) == "" {
		h.l.Warn("HolmesGPT hot-reload skipped: HolmesGPT URL is not configured")
		return reloadOutcome{Error: "HolmesGPT URL is not configured — set it in AI Assistant settings, then restart to apply"}, "rendered", false
	}

	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), reloadTimeout)
	defer cancel()
	result, err := h.holmesClient(settings).Reload(rctx)
	if err != nil {
		if errors.Is(err, ErrReloadUnsupported) {
			h.l.Warn("HolmesGPT hot-reload unavailable (admin API disabled or older Holmes build); restart required to apply")
		} else {
			h.l.Warnf("HolmesGPT hot-reload failed; restart required to apply: %v", err)
		}
		return reloadOutcome{Error: err.Error()}, "rendered", false
	}
	return reloadOutcome{Reloaded: true, Detail: result.Detail}, "reloaded", true
}

// resolvePMMURL returns the PMM URL Holmes uses to call back into PMM: the stored value, else the
// configured public address, else empty (admin can set it explicitly).
func (h *Handlers) resolvePMMURL() string {
	prov, err := models.GetAdreProvisioning(h.db)
	if err == nil && prov.PMMURL != "" {
		return prov.PMMURL
	}
	settings, err := models.GetSettings(h.db)
	if err == nil {
		if u := models.NormalizePMMPublicAddressOrigin(settings.PMMPublicAddress); u != "" {
			return u
		}
	}
	return ""
}

func validSkillNameAPI(name string) bool {
	if name == "" {
		return false
	}
	for _, c := range name {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '-' && c != '_' {
			return false
		}
	}
	return true
}

// validModelName allows model_list keys like "gpt-5.4" while rejecting "/", whitespace and other
// characters that would break URL routing for DELETE /models/{name}.
func validModelName(name string) bool {
	if name == "" {
		return false
	}
	for _, c := range name {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '.' && c != '-' && c != '_' {
			return false
		}
	}
	return true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20)) //nolint:mnd
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read request body")
		return false
	}
	if err := json.Unmarshal(body, v); err != nil { //nolint:noinlineerr
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// ServeDeploymentSubroutes dispatches /v1/adre/deployment[/...] (admin-only inside each handler).
func (h *Handlers) ServeDeploymentSubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/adre/deployment")
	path = strings.Trim(path, "/")

	switch {
	case path == "" && r.Method == http.MethodGet:
		h.GetDeployment(w, r)
	case path == "config" && r.Method == http.MethodPut:
		h.PutDeploymentConfig(w, r)
	case path == "models" && r.Method == http.MethodPut:
		h.PutDeploymentModels(w, r)
	case strings.HasPrefix(path, "models/") && r.Method == http.MethodDelete:
		h.DeleteDeploymentModel(w, r, strings.TrimPrefix(path, "models/"))
	case path == "provisioning" && r.Method == http.MethodPut:
		h.PutDeploymentProvisioning(w, r)
	case path == "apply" && r.Method == http.MethodPost:
		h.PostDeploymentApply(w, r)
	case path == "restarted" && r.Method == http.MethodPost:
		h.PostDeploymentRestarted(w, r)
	case path == "provision" && r.Method == http.MethodPost:
		h.PostDeploymentProvision(w, r)
	case path == "skills" && (r.Method == http.MethodPost || r.Method == http.MethodPut):
		h.PutDeploymentSkill(w, r, "")
	case strings.HasPrefix(path, "skills/"):
		name := strings.TrimPrefix(path, "skills/")
		switch r.Method {
		case http.MethodPut, http.MethodPost:
			h.PutDeploymentSkill(w, r, name)
		case http.MethodDelete:
			h.DeleteDeploymentSkill(w, r, name)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	default:
		writeJSONError(w, http.StatusNotFound, "not found")
	}
}
