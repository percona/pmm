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

// Package platform provides authentication/authorization functionality.
package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/platformpb"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/grafana"
	"github.com/percona/pmm-managed/utils/envvars"
)

const rollbackFailed = "Failed to rollback:"

var internalServerError = status.Error(codes.Internal, "Internal server error")

// supervisordService is a subset of methods of supervisord.Service used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type supervisordService interface {
	UpdateConfiguration(settings *models.Settings, ssoDetails *models.PerconaSSODetails) error
}

// Service is responsible for interactions with Percona Platform.
type Service struct {
	db            *reform.DB
	host          string
	l             *logrus.Entry
	supervisord   supervisordService
	client        http.Client
	grafanaClient grafanaClient

	platformpb.UnimplementedPlatformServer
}

type grafanaClient interface {
	GetCurrentUserAccessToken(ctx context.Context) (string, error)
}

// New returns platform Service.
func New(db *reform.DB, supervisord supervisordService, grafanaClient grafanaClient) (*Service, error) {
	l := logrus.WithField("component", "platform")

	host, err := envvars.GetSAASHost()
	if err != nil {
		return nil, err
	}

	timeout := envvars.GetPlatformAPITimeout(l)

	s := Service{
		host:          host,
		db:            db,
		l:             l,
		supervisord:   supervisord,
		client:        http.Client{Timeout: timeout},
		grafanaClient: grafanaClient,
	}

	return &s, nil
}

// Connect connects a PMM server to the organization created on Percona Portal. That allows the user to sign in to the PMM server with their Percona Account.
func (s *Service) Connect(ctx context.Context, req *platformpb.ConnectRequest) (*platformpb.ConnectResponse, error) {
	_, err := models.GetPerconaSSODetails(ctx, s.db.Querier)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "PMM server is already connected to Portal")
	}
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.Errorf("Failed to fetch PMM server ID and address: %s", err)
		return nil, internalServerError
	}
	if settings.PMMPublicAddress == "" {
		return nil, status.Error(codes.FailedPrecondition, "The address of PMM server is not set")
	}
	pmmServerURL := fmt.Sprintf("https://%s/graph", settings.PMMPublicAddress)

	connectResp, err := s.connect(ctx, &connectPMMParams{
		serverName:                req.ServerName,
		email:                     req.Email,
		password:                  req.Password,
		pmmServerURL:              pmmServerURL,
		pmmServerOAuthCallbackURL: fmt.Sprintf("%s/login/generic_oauth", pmmServerURL),
		pmmServerID:               settings.PMMServerID,
	})
	if err != nil {
		return nil, err // this is already a status error
	}

	err = models.InsertPerconaSSODetails(s.db.Querier, &models.PerconaSSODetailsInsert{
		ClientID:       connectResp.SSODetails.ClientID,
		ClientSecret:   connectResp.SSODetails.ClientSecret,
		IssuerURL:      connectResp.SSODetails.IssuerURL,
		Scope:          connectResp.SSODetails.Scope,
		OrganizationID: connectResp.OrganizationID,
	})
	if err != nil {
		s.l.Errorf("Failed to insert SSO details: %s", err)
		return nil, internalServerError
	}

	if err := s.UpdateSupervisordConfigurations(ctx); err != nil {
		s.l.Errorf("Failed to update configuration of grafana after connecting PMM to Portal: %s", err)
		return nil, internalServerError
	}
	return &platformpb.ConnectResponse{}, nil
}

// Disconnect disconnects a PMM server from the organization created on Percona Portal.
func (s *Service) Disconnect(ctx context.Context, req *platformpb.DisconnectRequest) (*platformpb.DisconnectResponse, error) {
	ssoDetails, err := models.GetPerconaSSODetails(ctx, s.db.Querier)
	if err != nil {
		s.l.Errorf("failed to get SSO details: %s", err)
		return nil, status.Error(codes.Aborted, "PMM server is not connected to Portal")
	}

	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.Errorf("Failed to fetch PMM server ID and address: %s", err)
		return nil, internalServerError
	}

	err = models.DeletePerconaSSODetails(s.db.Querier)
	if err != nil {
		s.l.Errorf("Failed to delete SSO details: %s", err)
		if e := s.UpdateSupervisordConfigurations(ctx); e != nil {
			s.l.Errorf("%s %s", rollbackFailed, e)
		}
		return nil, internalServerError
	}

	err = s.disconnect(ctx, &disconnectPMMParams{
		PMMServerID: settings.PMMServerID,
	})
	if err != nil {
		if e := models.InsertPerconaSSODetails(s.db.Querier, &models.PerconaSSODetailsInsert{
			ClientID:     ssoDetails.ClientID,
			ClientSecret: ssoDetails.ClientSecret,
			IssuerURL:    ssoDetails.IssuerURL,
			Scope:        ssoDetails.Scope,
		}); e != nil {
			s.l.Errorf("%s %s", rollbackFailed, e)
		}
		if e := s.UpdateSupervisordConfigurations(ctx); e != nil {
			s.l.Errorf("%s %s", rollbackFailed, e)
		}

		return nil, err // this is already a status error
	}

	if err = s.UpdateSupervisordConfigurations(ctx); err != nil {
		s.l.Errorf("Failed to update configuration of grafana after disconnect from Platform: %s", err)
		return nil, internalServerError
	}

	return &platformpb.DisconnectResponse{}, nil
}

func (s *Service) UpdateSupervisordConfigurations(ctx context.Context) error {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return errors.Wrap(err, "failed to get settings")
	}
	ssoDetails, err := models.GetPerconaSSODetails(ctx, s.db.Querier)
	if err != nil {
		if !errors.Is(err, reform.ErrNoRows) {
			return errors.Wrap(err, "failed to get SSO details")
		}
	}
	if err := s.supervisord.UpdateConfiguration(settings, ssoDetails); err != nil {
		return errors.Wrap(err, "failed to update supervisord configuration")
	}
	return nil
}

type connectPMMParams struct {
	pmmServerURL, pmmServerOAuthCallbackURL, pmmServerID, serverName, email, password string
}

type connectPMMRequest struct {
	PMMServerID               string `json:"pmm_server_id"`
	PMMServerName             string `json:"pmm_server_name"`
	PMMServerURL              string `json:"pmm_server_url"`
	PMMServerOAuthCallbackURL string `json:"pmm_server_oauth_callback_url"`
}

type disconnectPMMParams struct {
	PMMServerID string
}

type ssoDetails struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Scope        string `json:"scope"`
	IssuerURL    string `json:"issuer_url"`
}

type connectPMMResponse struct {
	SSODetails     *ssoDetails `json:"sso_details"`
	OrganizationID string      `json:"org_id"`
}

type grpcGatewayError struct {
	Message string `json:"message"`
	Code    uint32 `json:"code"`
}

func (s *Service) connect(ctx context.Context, params *connectPMMParams) (*connectPMMResponse, error) {
	endpoint := fmt.Sprintf("https://%s/v1/orgs/inventory", s.host)
	marshaled, err := json.Marshal(connectPMMRequest{
		PMMServerID:               params.pmmServerID,
		PMMServerName:             params.serverName,
		PMMServerURL:              params.pmmServerURL,
		PMMServerOAuthCallbackURL: params.pmmServerOAuthCallbackURL,
	})
	if err != nil {
		s.l.Errorf("Failed to marshal request data: %s", err)
		return nil, internalServerError
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(marshaled))
	if err != nil {
		s.l.Errorf("Failed to build Connect to Platform request: %s", err)
		return nil, internalServerError
	}
	req.SetBasicAuth(params.email, params.password)
	resp, err := s.client.Do(req)
	if err != nil {
		s.l.Errorf("Connect to Platform request failed: %s", err)
		return nil, internalServerError
	}
	defer resp.Body.Close() //nolint:errcheck

	decoder := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var gwErr grpcGatewayError
		if err := decoder.Decode(&gwErr); err != nil {
			s.l.Errorf("Connect to Platform request failed and we failed to decode error message: %s", err)
			return nil, internalServerError
		}
		return nil, status.Error(codes.Code(gwErr.Code), gwErr.Message)
	}

	response := &connectPMMResponse{}
	if err := decoder.Decode(response); err != nil {
		s.l.Errorf("Failed to decode response into SSO details: %s", err)
		return nil, internalServerError
	}
	return response, nil
}

func (s *Service) disconnect(ctx context.Context, params *disconnectPMMParams) error {
	userAccessToken, err := s.grafanaClient.GetCurrentUserAccessToken(ctx)
	if err != nil {
		if errors.Is(err, grafana.ErrFailedToGetToken) {
			return status.Error(codes.FailedPrecondition, "Failed to get access token. Please sign in using your Percona Account.")
		}
		s.l.Errorf("Disconnect to Platform request failed: %s", err)
		return internalServerError
	}

	endpoint := fmt.Sprintf("https://%s/v1/orgs/inventory/%s:disconnect", s.host, params.PMMServerID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		s.l.Errorf("Failed to build Disconnect to Platform request: %s", err)
		return internalServerError
	}

	h := req.Header
	h.Add("Authorization", fmt.Sprintf("Bearer %s", userAccessToken))

	resp, err := s.client.Do(req)
	if err != nil {
		s.l.Errorf("Disconnect to Platform request failed: %s", err)
		return internalServerError
	}
	defer resp.Body.Close() //nolint:errcheck

	decoder := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var gwErr grpcGatewayError
		if err := decoder.Decode(&gwErr); err != nil {
			s.l.Errorf("Disconnect to Platform request failed and we failed to decode error message: %s", err)
			return internalServerError
		}
		return status.Error(codes.Code(gwErr.Code), gwErr.Message)
	}

	return nil
}

type searchOrganizationTicketsResponse struct {
	Tickets []*ticketResponse `json:"tickets"`
}

type ticketResponse struct {
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

// SearchOrganizationTickets fetches the list of ticket associated with the Portal organization this PMM server is registered with.
func (s *Service) SearchOrganizationTickets(ctx context.Context, req *platformpb.SearchOrganizationTicketsRequest) (*platformpb.SearchOrganizationTicketsResponse, error) {
	userAccessToken, err := s.grafanaClient.GetCurrentUserAccessToken(ctx)
	if err != nil {
		if errors.Is(err, grafana.ErrFailedToGetToken) {
			return nil, status.Error(codes.Unauthenticated, "Failed to get access token. Please sign in using your Percona Account.")
		}
		s.l.Errorf("SearchOrganizationTickets request failed: %s", err)
		return nil, internalServerError
	}

	ssoDetails, err := models.GetPerconaSSODetails(ctx, s.db.Querier)
	if err != nil {
		s.l.Errorf("failed to get SSO details: %s", err)
		return nil, status.Error(codes.Aborted, "PMM server is not connected to Portal")
	}

	endpoint := fmt.Sprintf("https://%s/v1/orgs/%s/tickets:search", s.host, ssoDetails.OrganizationID)

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		s.l.Errorf("Failed to build SearchOrganizationTickets request: %s", err)
		return nil, internalServerError
	}

	h := r.Header
	h.Add("Authorization", fmt.Sprintf("Bearer %s", userAccessToken))

	resp, err := s.client.Do(r)
	if err != nil {
		s.l.Errorf("SearchOrganizationTickets request failed: %s", err)
		return nil, internalServerError
	}
	defer resp.Body.Close() //nolint:errcheck

	decoder := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var gwErr grpcGatewayError
		if err := decoder.Decode(&gwErr); err != nil {
			s.l.Errorf("SearchOrganizationRequest failed to decode error message: %s", err)
			return nil, internalServerError
		}
		return nil, status.Error(codes.Code(gwErr.Code), gwErr.Message)
	}

	// the response from portal contains the timestamp as a string
	// so we first unmarshal the response to an internal type with a string
	// timestamp field and then convert it to the type used by the public API.
	platformResponse := &searchOrganizationTicketsResponse{}
	if err := decoder.Decode(platformResponse); err != nil {
		s.l.Errorf("Failed to decode response into OrganizationTickets: %s", err)
		return nil, internalServerError
	}

	response := &platformpb.SearchOrganizationTicketsResponse{}
	for _, t := range platformResponse.Tickets {
		ticket, err := convertTicket(t)
		if err != nil {
			s.l.Errorf("Failed to convert OrganizationTickets: %s", err)
			return nil, internalServerError
		}
		response.Tickets = append(response.Tickets, ticket)
	}

	return response, nil
}

func convertTicket(t *ticketResponse) (*platformpb.OrganizationTicket, error) {
	createTime, err := time.Parse(time.RFC3339, t.CreateTime)
	if err != nil {
		return nil, err
	}

	return &platformpb.OrganizationTicket{
		Number:           t.Number,
		ShortDescription: t.ShortDescription,
		Priority:         t.Priority,
		State:            t.State,
		CreateTime:       timestamppb.New(createTime),
		Department:       t.Department,
		Requester:        t.Requester,
		TaskType:         t.TaskType,
		Url:              t.URL,
	}, nil
}

type searchOrganizationEntitlementsResponse struct {
	Entitlement []*entitlementResponse `json:"entitlements"`
}

type entitlementResponse struct {
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
	Platform         platformResponse `json:"platform"`
}

type platformResponse struct {
	SecurityAdvisor string `json:"security_advisor"` //nolint:tagliatelle
	ConfigAdvisor   string `json:"config_advisor"`   //nolint:tagliatelle
}

// SearchOrganizationEntitlements fetches customer entitlements for a particular organization.
func (s *Service) SearchOrganizationEntitlements(ctx context.Context, req *platformpb.SearchOrganizationEntitlementsRequest) (*platformpb.SearchOrganizationEntitlementsResponse, error) {
	userAccessToken, err := s.grafanaClient.GetCurrentUserAccessToken(ctx)
	if err != nil {
		if errors.Is(err, grafana.ErrFailedToGetToken) {
			return nil, status.Error(codes.Unauthenticated, "Failed to get access token. Please sign in using your Percona Account.")
		}
		s.l.Errorf("SearchOrganizationEntitlements request failed: %s", err)
		return nil, internalServerError
	}

	ssoDetails, err := models.GetPerconaSSODetails(ctx, s.db.Querier)
	if err != nil {
		s.l.Errorf("failed to get SSO details: %s", err)
		return nil, status.Error(codes.Aborted, "PMM server is not connected to Portal")
	}

	endpoint := fmt.Sprintf("https://%s/v1/orgs/%s/entitlements:search", s.host, ssoDetails.OrganizationID)

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		s.l.Errorf("Failed to build SearchOrganizationEntitlements request: %s", err)
		return nil, internalServerError
	}

	h := r.Header
	h.Add("Authorization", fmt.Sprintf("Bearer %s", userAccessToken))

	resp, err := s.client.Do(r)
	if err != nil {
		s.l.Errorf("SearchOrganizationEntitlements request failed: %s", err)
		return nil, internalServerError
	}
	defer resp.Body.Close() //nolint:errcheck

	decoder := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var gwErr grpcGatewayError
		if err := decoder.Decode(&gwErr); err != nil {
			s.l.Errorf("Failed to decode error message: %s", err)
			return nil, internalServerError
		}
		return nil, status.Error(codes.Code(gwErr.Code), gwErr.Message)
	}

	// the response from portal contains the timestamp as a string
	// so we first unmarshal the response to an internal type with a string
	// timestamp field and then convert it to the type used by the public API.
	platformResp := &searchOrganizationEntitlementsResponse{}
	if err := decoder.Decode(platformResp); err != nil {
		s.l.Errorf("Failed to decode response into OrganizationTickets: %s", err)
		return nil, internalServerError
	}

	response := &platformpb.SearchOrganizationEntitlementsResponse{}
	for _, e := range platformResp.Entitlement {
		entitlement, err := convertEntitlement(e)
		if err != nil {
			s.l.Errorf("Failed to convert OrganizationEntitlements: %s", err)
			return nil, internalServerError
		}
		response.Entitlements = append(response.Entitlements, entitlement)
	}

	return response, nil
}

func convertEntitlement(ent *entitlementResponse) (*platformpb.OrganizationEntitlement, error) {
	startDate, err := time.Parse(time.RFC3339, ent.StartDate)
	if err != nil {
		return nil, err
	}

	endDate, err := time.Parse(time.RFC3339, ent.EndDate)
	if err != nil {
		return nil, err
	}

	return &platformpb.OrganizationEntitlement{
		Number:           ent.Number,
		Name:             ent.Name,
		Summary:          ent.Summary,
		Tier:             &wrapperspb.StringValue{Value: ent.Tier},
		TotalUnits:       &wrapperspb.StringValue{Value: ent.TotalUnits},
		UnlimitedUnits:   &wrapperspb.BoolValue{Value: ent.UnlimitedUnits},
		SupportLevel:     &wrapperspb.StringValue{Value: ent.SupportLevel},
		SoftwareFamilies: ent.SoftwareFamilies,
		StartDate:        timestamppb.New(startDate),
		EndDate:          timestamppb.New(endDate),
		Platform: &platformpb.OrganizationEntitlement_Platform{
			ConfigAdvisor:   &wrapperspb.StringValue{Value: ent.Platform.ConfigAdvisor},
			SecurityAdvisor: &wrapperspb.StringValue{Value: ent.Platform.SecurityAdvisor},
		},
	}, nil
}
