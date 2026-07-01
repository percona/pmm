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
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/pkg/errors" //nolint:depguard
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// holmesServiceAccountName is the Grafana service account PMM mints for HolmesGPT's PMM_API_TOKEN.
const holmesServiceAccountName = "holmesgpt"

// ServiceAccountCreator mints a Grafana service-account token and provisions the auto-investigate
// alert webhook contact point. *grafana.Client satisfies it.
type ServiceAccountCreator interface {
	CreateServiceAccount(ctx context.Context, nodeName string, reregister bool) (int, string, error)
	EnsureAlertWebhookContactPoint(ctx context.Context, webhookURL, secret string) error
}

// Provisioner ensures the PMM↔Holmes bootstrap secrets exist: a minted Grafana service-account
// token (PMM_API_TOKEN) and a generated HOLMES_API_KEY.
type Provisioner struct {
	db *reform.DB
	sa ServiceAccountCreator
	l  *logrus.Entry
}

// NewProvisioner returns a Provisioner.
func NewProvisioner(db *reform.DB, sa ServiceAccountCreator, l *logrus.Entry) *Provisioner {
	return &Provisioner{db: db, sa: sa, l: l}
}

// EnsureProvisioned mints/generates any missing bootstrap secrets and records pmmURL, persisting
// changes. It is idempotent: existing token/key are kept. Returns the current provisioning row.
func (p *Provisioner) EnsureProvisioned(ctx context.Context, pmmURL string) (*models.AdreProvisioning, error) {
	prov, err := models.GetAdreProvisioning(p.db)
	if err != nil {
		return nil, err
	}

	changed := false

	if prov.PMMSAToken == "" {
		// reregister=true so a retry recovers cleanly if the service account already exists from a
		// prior partial provision (e.g. SA created but token never persisted). Only runs when no
		// token is stored, so it does not rotate an existing working token.
		id, token, err := p.sa.CreateServiceAccount(ctx, holmesServiceAccountName, true)
		if err != nil {
			return nil, errors.Wrap(err, "failed to mint HolmesGPT service-account token")
		}
		prov.PMMSAID = id
		prov.PMMSAToken = token
		changed = true
		p.l.Infof("Minted HolmesGPT service account (id=%d)", id)
	}

	if prov.HolmesAPIKey == "" {
		key, err := generateAPIKey()
		if err != nil {
			return nil, err
		}
		prov.HolmesAPIKey = key
		changed = true
		p.l.Info("Generated HOLMES_API_KEY")
	}

	if pmmURL != "" && prov.PMMURL != pmmURL {
		prov.PMMURL = pmmURL
		changed = true
	}

	if changed {
		err := models.SaveAdreProvisioning(p.db, prov)
		if err != nil {
			return nil, err
		}
	}

	// Best-effort: provision the auto-investigate webhook contact point. Failures are non-fatal —
	// auto-investigate still runs via the reconciliation poll.
	if err := p.EnsureAlertWebhook(ctx); err != nil { //nolint:noinlineerr
		p.l.Warnf("%v (auto-investigate still runs via the reconciliation poll)", err)
	}

	return prov, nil
}

// internalWebhookBaseURL is how Grafana (co-located with pmm-managed) reaches PMM's alert webhook. It
// is loopback — the address PMM's default TLS cert is valid for — so Grafana's webhook sender passes
// certificate verification. It is deliberately independent of the public/Holmes PMM URL, which may be
// an external address the local cert does not cover (the cause of x509 "valid for 127.0.0.1, not <ip>").
const internalWebhookBaseURL = "https://127.0.0.1"

// EnsureAlertWebhook idempotently provisions the auto-investigate webhook secret plus the Grafana
// contact point + catch-all route that deliver firing alerts to PMM's authenticated alert webhook.
// Best-effort: callers log failures and continue (auto-investigate still runs via the reconciliation
// poll). ctx must carry the admin auth headers — the Grafana provisioning API requires them.
func (p *Provisioner) EnsureAlertWebhook(ctx context.Context) error {
	secret, err := models.EnsureAlertWebhookSecret(p.db)
	if err != nil {
		return errors.Wrap(err, "ensure alert webhook secret")
	}
	if secret == "" {
		return nil
	}
	webhookURL := internalWebhookBaseURL + "/v1/adre/alert-webhook"
	return errors.Wrap(p.sa.EnsureAlertWebhookContactPoint(ctx, webhookURL, secret), "auto-provision alert webhook contact point")
}

// generateAPIKey returns a 256-bit base64 random key.
func generateAPIKey() (string, error) {
	b := make([]byte, 32)                   //nolint:mnd
	if _, err := rand.Read(b); err != nil { //nolint:noinlineerr
		return "", errors.Wrap(err, "failed to generate API key")
	}
	return base64.RawStdEncoding.EncodeToString(b), nil
}
