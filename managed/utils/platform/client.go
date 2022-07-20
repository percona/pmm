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

// Package platform implements HTTP client for Percona Platform.
package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	api "github.com/percona-platform/saas/gen/check/retrieval"
	reporter "github.com/percona-platform/saas/gen/telemetry/reporter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/envvars"
	"github.com/percona/pmm/utils/tlsconfig"
)

// Client is HTTP Percona Platform client.
// TODO: Replace this client with generated one https://jira.percona.com/browse/SAAS-956
type Client struct {
	db *reform.DB

	address string
	l       *logrus.Entry
	client  http.Client
}

// NewClient creates new Percona Platform client.
func NewClient(db *reform.DB, address string) (*Client, error) {
	l := logrus.WithField("component", "portal client")

	tlsConfig := tlsconfig.Get()
	tlsConfig.InsecureSkipVerify = envvars.GetPlatformInsecure()

	return &Client{
		db:      db,
		l:       l,
		address: address,
		client: http.Client{
			Timeout: envvars.GetPlatformAPITimeout(l),
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
	}, nil
}

// GetChecks download checks from Percona Platform. It also validates content and checks signatures.
func (c *Client) GetChecks(ctx context.Context) (*api.GetAllChecksResponse, error) {
	const path = "/v1/check/GetAllChecks"

	var accessToken string
	if ssoDetails, err := models.GetPerconaSSODetails(ctx, c.db.Querier); err == nil {
		accessToken = ssoDetails.AccessToken.AccessToken
	}

	c.l.Infof("Downloading checks from %s ...", c.address)
	bodyBytes, err := c.makeRequest(ctx, accessToken, http.MethodPost, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download checks")
	}

	var resp api.GetAllChecksResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetTemplates download templates from Percona Platform. It also validates content and checks signatures.
func (c *Client) GetTemplates(ctx context.Context) (*api.GetAllAlertRuleTemplatesResponse, error) {
	const path = "/v1/check/GetAllAlertRuleTemplates"

	var accessToken string
	if ssoDetails, err := models.GetPerconaSSODetails(ctx, c.db.Querier); err == nil {
		accessToken = ssoDetails.AccessToken.AccessToken
	}

	c.l.Infof("Downloading templates from %s ...", c.address)
	bodyBytes, err := c.makeRequest(ctx, accessToken, http.MethodPost, path, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download checks")
	}

	var resp api.GetAllAlertRuleTemplatesResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SendTelemetry sends telemetry data to Percona Platform.
func (c *Client) SendTelemetry(ctx context.Context, report *reporter.ReportRequest) error {
	const path = "/v1/telemetry/Report"

	var accessToken string
	if ssoDetails, err := models.GetPerconaSSODetails(ctx, c.db.Querier); err == nil {
		accessToken = ssoDetails.AccessToken.AccessToken
	}

	body, err := protojson.Marshal(report)
	if err != nil {
		return err
	}

	_, err = c.makeRequest(ctx, accessToken, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return errors.Wrap(err, "failed to send telemetry data")
	}

	return nil
}

// Connect send connect request to Percona Platform.
func (c *Client) Connect(ctx context.Context, accessToken, pmmServerID, pmmServerName, pmmServerURL, pmmServerOAuthCallbackURL string) (*ConnectPMMResponse, error) {
	const path = "/v1/orgs/inventory"

	body, err := json.Marshal(struct {
		PMMServerID               string `json:"pmm_server_id"`
		PMMServerName             string `json:"pmm_server_name"`
		PMMServerURL              string `json:"pmm_server_url"`
		PMMServerOAuthCallbackURL string `json:"pmm_server_oauth_callback_url"`
	}{
		PMMServerID:               pmmServerID,
		PMMServerName:             pmmServerName,
		PMMServerURL:              pmmServerURL,
		PMMServerOAuthCallbackURL: pmmServerOAuthCallbackURL,
	})
	if err != nil {
		c.l.Errorf("Failed to marshal request data: %s", err)
		return nil, err
	}

	bodyBytes, err := c.makeRequest(ctx, accessToken, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		c.l.Errorf("Failed to build Connect to Platform request: %s", err)
		return nil, err
	}

	var resp ConnectPMMResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		c.l.Errorf("Failed to decode response into SSO details: %s", err)
		return nil, err
	}

	return &resp, nil
}

// Disconnect send disconnect request to Percona Platform.
func (c *Client) Disconnect(ctx context.Context, accessToken, pmmServerID string) error {
	const path = "/v1/orgs/inventory/%s:disconnect"

	_, err := c.makeRequest(ctx, accessToken, http.MethodPost, fmt.Sprintf(path, pmmServerID), nil)
	if err != nil {
		return err
	}

	return nil
}

// SearchOrgTickets searches tickets for given organization ID.
func (c *Client) SearchOrgTickets(ctx context.Context, accessToken, orgID string) (*SearchOrganizationTicketsResponse, error) {
	const path = "/v1/orgs/%s/tickets:search"

	resp, err := c.makeRequest(ctx, accessToken, http.MethodPost, fmt.Sprintf(path, orgID), nil)
	if err != nil {
		return nil, err
	}

	var res SearchOrganizationTicketsResponse
	if err := json.Unmarshal(resp, &res); err != nil {
		c.l.Errorf("Failed to decode response into OrganizationTickets: %s", err)
		return nil, err
	}

	return &res, nil
}

// SearchOrgEntitlements searches entitlements for given organization ID.
func (c *Client) SearchOrgEntitlements(ctx context.Context, accessToken, orgID string) (*SearchOrganizationEntitlementsResponse, error) {
	const path = "/v1/orgs/%s/entitlements:search"

	resp, err := c.makeRequest(ctx, accessToken, http.MethodPost, fmt.Sprintf(path, orgID), nil)
	if err != nil {
		return nil, err
	}

	var res SearchOrganizationEntitlementsResponse
	if err := json.Unmarshal(resp, &res); err != nil {
		c.l.Errorf("Failed to decode response into OrganizationTickets: %s", err)
		return nil, err
	}

	return &res, nil
}

// GetContactInformation returns contact information for given organization ID.
func (c *Client) GetContactInformation(ctx context.Context, accessToken, orgID string) (*ContactInformation, error) {
	const path = "/v1/orgs/%s"

	resp, err := c.makeRequest(ctx, accessToken, http.MethodGet, fmt.Sprintf(path, orgID), nil)
	if err != nil {
		return nil, err
	}

	var res ContactInformation
	if err := json.Unmarshal(resp, &res); err != nil {
		c.l.Errorf("Failed to decode response : %s", err)
		return nil, err
	}

	return &res, nil
}

// MakeRequest makes request to Percona Platform.
func (c *Client) makeRequest(ctx context.Context, accessToken, method, path string, body io.Reader) ([]byte, error) {
	endpoint := c.address + path
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}

	h := req.Header
	h.Add("Content-Type", "application/json")
	if accessToken != "" {
		h.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close() //nolint:errcheck

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var gwErr struct {
			Message string `json:"message"`
			Code    uint32 `json:"code"`
		}

		if err := json.Unmarshal(bodyBytes, &gwErr); err != nil {
			c.l.Errorf("Failed to dial Percona Portal and we failed to decode error message: %s", err)
			return nil, err
		}
		return nil, status.Error(codes.Code(gwErr.Code), gwErr.Message)
	}

	return bodyBytes, nil
}

type SsoDetails struct {
	GrafanaClientID        string `json:"grafana_client_id"`         //nolint:tagliatelle
	PMMManagedClientID     string `json:"pmm_managed_client_id"`     //nolint:tagliatelle
	PMMManagedClientSecret string `json:"pmm_managed_client_secret"` //nolint:tagliatelle
	Scope                  string `json:"scope"`
	IssuerURL              string `json:"issuer_url"` //nolint:tagliatelle
}

type ConnectPMMResponse struct {
	SSODetails     *SsoDetails `json:"sso_details"`
	OrganizationID string      `json:"org_id"`
}

type SearchOrganizationEntitlementsResponse struct {
	Entitlement []*EntitlementResponse `json:"entitlements"`
}

type EntitlementResponse struct {
	Number           string           `json:"number"`
	Name             string           `json:"name"`
	Summary          string           `json:"summary"`
	Tier             string           `json:"tier"`
	TotalUnits       string           `json:"total_units"`       //nolint:tagliatelle
	UnlimitedUnits   bool             `json:"unlimited_units"`   //nolint:tagliatelle
	SupportLevel     string           `json:"support_level"`     //nolint:tagliatelle
	SoftwareFamilies []string         `json:"software_families"` //nolint:tagliatelle
	StartDate        string           `json:"start_date"`        //nolint:tagliatelle
	EndDate          string           `json:"end_date"`          //nolint:tagliatelle
	Platform         PlatformResponse `json:"platform"`
}

type PlatformResponse struct {
	SecurityAdvisor string `json:"security_advisor"` //nolint:tagliatelle
	ConfigAdvisor   string `json:"config_advisor"`   //nolint:tagliatelle
}

type SearchOrganizationTicketsResponse struct {
	Tickets []*TicketResponse `json:"tickets"`
}

type TicketResponse struct {
	Number           string `json:"number"`
	ShortDescription string `json:"short_description"` //nolint:tagliatelle
	Priority         string `json:"priority"`
	State            string `json:"state"`
	CreateTime       string `json:"create_time"` //nolint:tagliatelle
	Department       string `json:"department"`
	Requester        string `json:"requestor"`
	TaskType         string `json:"task_type"` //nolint:tagliatelle
	URL              string `json:"url"`
}

type ContactInformation struct {
	Contacts struct {
		CustomerSuccess struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"customer_success"` //nolint:tagliatelle
		NewTicketURL string `json:"new_ticket_url"` //nolint:tagliatelle
	} `json:"contacts"`
}
