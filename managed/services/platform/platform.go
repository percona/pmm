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

// Package platform provides authentication/authorization functionality.
package platform

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/platformpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/grafana"
	"github.com/percona/pmm/managed/utils/platform"
)

const rollbackFailed = "Failed to rollback:"

var (
	errProxyNotSupported         = status.Error(codes.Aborted, "PMM Platform connection does not support proxy.")
	errGetSSODetailsFailed       = status.Error(codes.Aborted, "Failed to fetch SSO details.")
	errGrafanaAccessTokenFailed  = status.Error(codes.Unauthenticated, "Failed to get user access token. Please sign in using your Percona Portal account.")
	errPMMServerAlreadyConnected = status.Error(codes.AlreadyExists, "PMM server is already connected to Portal")
	errPMMServerAddressNotSet    = status.Error(codes.FailedPrecondition, "The address of PMM server is not set")
	errNotConnectedToPortal      = status.Error(codes.Aborted, "PMM server is not connected to Portal")
	errNotACustomer              = status.Error(codes.FailedPrecondition, "User is not a Percona customer.")
)

// Service is responsible for interactions with Percona Platform.
type Service struct {
	db            *reform.DB
	l             *logrus.Entry
	client        *platform.Client
	grafanaClient grafanaClient
	supervisord   supervisordService
	checksService checksService

	platformpb.UnimplementedPlatformServer
}

// New returns platform Service.
func New(client *platform.Client, db *reform.DB, supervisord supervisordService, checksService checksService, grafanaClient grafanaClient) *Service {
	l := logrus.WithField("component", "platform")

	s := Service{
		db:            db,
		client:        client,
		l:             l,
		supervisord:   supervisord,
		checksService: checksService,

		grafanaClient: grafanaClient,
	}

	return &s
}

// Connect connects a PMM server to the organization created on Percona Portal. That allows the user to sign in to the PMM server with their Percona Account.
func (s *Service) Connect(ctx context.Context, req *platformpb.ConnectRequest) (*platformpb.ConnectResponse, error) {
	_, err := models.GetPerconaSSODetails(ctx, s.db.Querier)
	if err == nil {
		return nil, errPMMServerAlreadyConnected
	}
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.Errorf("Failed to fetch PMM server ID and address: %s", err)
		return nil, err
	}
	if settings.PMMPublicAddress == "" {
		return nil, errPMMServerAddressNotSet
	}

	pmmServerURL := fmt.Sprintf("https://%s/graph", settings.PMMPublicAddress)
	pmmServerOAuthCallbackURL := fmt.Sprintf("%s/login/generic_oauth", pmmServerURL)

	resp, err := s.client.Connect(ctx, req.PersonalAccessToken, settings.PMMServerID, req.ServerName, pmmServerURL, pmmServerOAuthCallbackURL)
	if err != nil {
		if strings.Contains(err.Error(), "proxyconnect tcp") {
			return nil, errProxyNotSupported
		}
		return nil, err
	}

	err = models.InsertPerconaSSODetails(s.db.Querier, &models.PerconaSSODetailsInsert{
		PMMManagedClientID:     resp.SSODetails.PMMManagedClientID,
		PMMManagedClientSecret: resp.SSODetails.PMMManagedClientSecret,
		GrafanaClientID:        resp.SSODetails.GrafanaClientID,
		IssuerURL:              resp.SSODetails.IssuerURL,
		Scope:                  resp.SSODetails.Scope,
		OrganizationID:         resp.OrganizationID,
		PMMServerName:          req.ServerName,
	})
	if err != nil {
		s.l.Errorf("Failed to insert SSO details: %s", err)
		return nil, err
	}

	if !settings.SaaS.STTDisabled {
		s.checksService.CollectAdvisors(ctx)
	}

	if err := s.UpdateSupervisordConfigurations(ctx); err != nil {
		s.l.Errorf("Failed to update configuration of grafana after connecting PMM to Portal: %s", err)
		return nil, err
	}
	return &platformpb.ConnectResponse{}, nil
}

// Disconnect disconnects a PMM server from the organization created on Percona Portal.
func (s *Service) Disconnect(ctx context.Context, req *platformpb.DisconnectRequest) (*platformpb.DisconnectResponse, error) {
	_, err := models.GetPerconaSSODetails(ctx, s.db.Querier)
	if err != nil {
		s.l.Errorf("Failed to get SSO details: %s", err)
		return nil, errGetSSODetailsFailed
	}

	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.Errorf("Failed to fetch PMM server ID and address: %s", err)
		return nil, err
	}

	if req.Force {
		err = s.forceDisconnect(ctx)
		if err != nil {
			s.l.Errorf("Force disconnect failed: %s", err)
			return nil, err
		}
		return &platformpb.DisconnectResponse{}, nil
	}

	userAccessToken, err := s.grafanaClient.GetCurrentUserAccessToken(ctx)
	if err != nil {
		if errors.Is(err, grafana.ErrFailedToGetToken) {
			return nil, errGrafanaAccessTokenFailed
		}
		s.l.Errorf("Disconnect to Platform request failed: %s", err)
		return nil, err
	}

	err = s.client.Disconnect(ctx, userAccessToken, settings.PMMServerID)
	if err != nil {
		s.l.Errorf("Disconnect request to Platform failed: %s", err)
		return nil, err
	}

	err = models.DeletePerconaSSODetails(s.db.Querier)
	if err != nil {
		s.l.Errorf("Failed to delete SSO details: %s", err)
		if e := s.UpdateSupervisordConfigurations(ctx); e != nil {
			s.l.Errorf("%s %s", rollbackFailed, e)
		}
		return nil, err
	}

	if !settings.SaaS.STTDisabled {
		s.checksService.CollectAdvisors(ctx)
	}

	if err = s.UpdateSupervisordConfigurations(ctx); err != nil {
		s.l.Errorf("Failed to update configuration of grafana after disconnect from Platform: %s", err)
		return nil, err
	}

	return &platformpb.DisconnectResponse{}, nil
}

// forceDisconnect cleans up records of platform connection only from PMM side.
// This should only be used in case a user with admin credentials tries to disconnect.
// The SSO details should be removed from both the DB and grafana config.
func (s *Service) forceDisconnect(ctx context.Context) error {
	err := models.DeletePerconaSSODetails(s.db.Querier)
	if err != nil {
		s.l.Errorf("Failed to delete SSO details: %s", err)
		return err
	}
	if err = s.UpdateSupervisordConfigurations(ctx); err != nil {
		s.l.Errorf("Failed to update configuration of grafana after disconnect from Platform: %s", err)
		return err
	}
	return nil
}

// UpdateSupervisordConfigurations updates supervisord configurations for the service.
func (s *Service) UpdateSupervisordConfigurations(ctx context.Context) error {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return errors.Wrap(err, "failed to get settings")
	}
	ssoDetails, err := models.GetPerconaSSODetails(ctx, s.db.Querier)
	if err != nil {
		if !errors.Is(err, models.ErrNotConnectedToPortal) {
			return errors.Wrap(err, "failed to get SSO details")
		}
	}
	if err := s.supervisord.UpdateConfiguration(settings, ssoDetails); err != nil {
		return errors.Wrap(err, "failed to update supervisord configuration")
	}
	return nil
}

// SearchOrganizationTickets fetches the list of ticket associated with the Portal organization this PMM server is registered with.
func (s *Service) SearchOrganizationTickets(ctx context.Context, _ *platformpb.SearchOrganizationTicketsRequest) (*platformpb.SearchOrganizationTicketsResponse, error) { //nolint:lll
	accessToken, err := s.grafanaClient.GetCurrentUserAccessToken(ctx)
	if err != nil {
		if errors.Is(err, grafana.ErrFailedToGetToken) {
			return nil, errGrafanaAccessTokenFailed
		}
		s.l.Errorf("SearchOrganizationTickets request failed: %s", err)
		return nil, err
	}

	ssoDetails, err := models.GetPerconaSSODetails(ctx, s.db.Querier)
	if err != nil {
		s.l.Errorf("Failed to get SSO details: %s", err)
		return nil, errGetSSODetailsFailed
	}

	resp, err := s.client.SearchOrgTickets(ctx, accessToken, ssoDetails.OrganizationID)
	if err != nil {
		return nil, err
	}

	response := &platformpb.SearchOrganizationTicketsResponse{}
	for _, t := range resp.Tickets {
		ticket, err := convertTicket(t)
		if err != nil {
			s.l.Errorf("Failed to convert OrganizationTickets: %s", err)
			return nil, err
		}
		response.Tickets = append(response.Tickets, ticket)
	}

	return response, nil
}

func convertTicket(t *platform.TicketResponse) (*platformpb.OrganizationTicket, error) {
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

// SearchOrganizationEntitlements fetches customer entitlements for a particular organization.
func (s *Service) SearchOrganizationEntitlements(ctx context.Context, _ *platformpb.SearchOrganizationEntitlementsRequest) (*platformpb.SearchOrganizationEntitlementsResponse, error) { //nolint:lll
	accessToken, err := s.grafanaClient.GetCurrentUserAccessToken(ctx)
	if err != nil {
		if errors.Is(err, grafana.ErrFailedToGetToken) {
			return nil, errGrafanaAccessTokenFailed
		}
		s.l.Errorf("SearchOrganizationEntitlements request failed: %s", err)
		return nil, err
	}

	ssoDetails, err := models.GetPerconaSSODetails(ctx, s.db.Querier)
	if err != nil {
		s.l.Errorf("Failed to get SSO details: %s", err)
		return nil, errGetSSODetailsFailed
	}

	resp, err := s.client.SearchOrgEntitlements(ctx, accessToken, ssoDetails.OrganizationID)
	if err != nil {
		return nil, err
	}

	response := &platformpb.SearchOrganizationEntitlementsResponse{}
	for _, e := range resp.Entitlement {
		entitlement, err := convertEntitlement(e)
		if err != nil {
			s.l.Errorf("Failed to convert OrganizationEntitlements: %s", err)
			return nil, err
		}
		response.Entitlements = append(response.Entitlements, entitlement)
	}

	return response, nil
}

func convertEntitlement(ent *platform.EntitlementResponse) (*platformpb.OrganizationEntitlement, error) {
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

// GetContactInformation fetches contact information of the Customer Success employee assigned to the Percona customer from Percona Portal.
func (s *Service) GetContactInformation(ctx context.Context, _ *platformpb.GetContactInformationRequest) (*platformpb.GetContactInformationResponse, error) {
	accessToken, err := s.grafanaClient.GetCurrentUserAccessToken(ctx)
	if err != nil {
		if errors.Is(err, grafana.ErrFailedToGetToken) {
			s.l.Error("Failed to get access token.")
			return nil, errGrafanaAccessTokenFailed
		}
		s.l.Errorf("GetContactInformation request failed: %s", err)
		return nil, err
	}

	ssoDetails, err := models.GetPerconaSSODetails(ctx, s.db.Querier)
	if err != nil {
		s.l.Errorf("Failed to get SSO details: %s", err)
		return nil, errNotConnectedToPortal
	}

	resp, err := s.client.GetContactInformation(ctx, accessToken, ssoDetails.OrganizationID)
	if err != nil {
		return nil, err
	}

	response := &platformpb.GetContactInformationResponse{
		CustomerSuccess: &platformpb.GetContactInformationResponse_CustomerSuccess{
			Name:  resp.Contacts.CustomerSuccess.Name,
			Email: resp.Contacts.CustomerSuccess.Email,
		},
		NewTicketUrl: resp.Contacts.NewTicketURL,
	}

	// Platform account is not linked to ServiceNow.
	if response.CustomerSuccess.Email == "" {
		s.l.Error("Failed to find contact information, non-customer account.")
		return nil, errNotACustomer
	}

	return response, nil
}

// ServerInfo retrieves server information based on the provided context and request.
func (s *Service) ServerInfo(ctx context.Context, _ *platformpb.ServerInfoRequest) (*platformpb.ServerInfoResponse, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.Errorf("Failed to fetch PMM server ID: %s", err)
		return nil, err
	}

	serverName := ""
	connectedToPortal := false
	ssoDetails, err := models.GetPerconaSSODetails(ctx, s.db.Querier)
	if err != nil {
		s.l.Errorf("Failed to get SSO details: %s", err)
	}

	if ssoDetails != nil {
		serverName = ssoDetails.PMMServerName
		connectedToPortal = true
	}

	return &platformpb.ServerInfoResponse{
		PmmServerName:        serverName,
		PmmServerId:          settings.PMMServerID,
		PmmServerTelemetryId: settings.Telemetry.UUID,
		ConnectedToPortal:    connectedToPortal,
	}, nil
}

// UserStatus API tells whether the logged-in user is a Platform organization member or not.
func (s *Service) UserStatus(ctx context.Context, _ *platformpb.UserStatusRequest) (*platformpb.UserStatusResponse, error) {
	// We use the access token instead of `models.GetPerconaSSODetails()`.
	// The reason for that is Frontend needs to use this API to know whether they can
	// show certain menu items to users "logged in with their Percona Accounts" after PMM
	// server has been connected to Platform. If we use the presence of SSO details in
	// the DB as the deciding factor for this it will also return true for the admin user
	// who connected the PMM server to Platform but wasn't logged into PMM with Platform creds.
	_, err := s.grafanaClient.GetCurrentUserAccessToken(ctx)
	if err != nil {
		if errors.Is(err, grafana.ErrFailedToGetToken) {
			return nil, errGrafanaAccessTokenFailed
		}
		s.l.Errorf("UserStatus request failed: %s", err)
		return nil, err
	}

	return &platformpb.UserStatusResponse{
		IsPlatformUser: true,
	}, nil
}
