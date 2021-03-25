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
	"context"
	"os"
	"time"

	api "github.com/percona-platform/saas/gen/auth"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/envvars"
	"github.com/percona/pmm-managed/utils/saasdial"
)

const (
	defaultSessionRefreshInterval = 24 * time.Hour

	envSessionRefreshInterval = "PERCONA_TEST_SESSION_REFRESH_INTERVAL"
)

var errNoActiveSessions = status.Error(codes.FailedPrecondition, "No active sessions.")

// Service is responsible for interactions with Percona Platform.
type Service struct {
	db                     *reform.DB
	host                   string
	sessionRefreshInterval time.Duration
	l                      *logrus.Entry
}

// New returns platform Service.
func New(db *reform.DB) (*Service, error) {
	l := logrus.WithField("component", "auth")

	host, err := envvars.GetSAASHost()
	if err != nil {
		return nil, err
	}

	s := Service{
		host:                   host,
		sessionRefreshInterval: defaultSessionRefreshInterval,
		db:                     db,
		l:                      l,
	}

	if d, err := time.ParseDuration(os.Getenv(envSessionRefreshInterval)); err == nil && d > 0 {
		l.Warnf("Session refresh interval changed to %s.", d)
		s.sessionRefreshInterval = d
	}

	return &s, nil
}

// Run refreshes Percona Platform session every interval until context is canceled.
func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(s.sessionRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// continue with next loop iteration
		case <-ctx.Done():
			return
		}

		nCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := s.refreshSession(nCtx)
		if err != nil && err != errNoActiveSessions {
			s.l.Warnf("Failed to refresh session, reason: %+v.", err)
		}
		cancel()
	}
}

// SignUp creates new Percona Platform user with given email and password.
func (s *Service) SignUp(ctx context.Context, email, firstName, lastName string) error {
	cc, err := saasdial.Dial(ctx, "", s.host)
	if err != nil {
		return errors.Wrap(err, "failed establish connection with Percona")
	}
	defer cc.Close() //nolint:errcheck

	_, err = api.NewAuthAPIClient(cc).SignUp(ctx, &api.SignUpRequest{Email: email, FirstName: firstName, LastName: lastName})
	if err != nil {
		return err
	}

	return nil
}

// SignIn checks Percona Platform user authentication and creates session.
func (s *Service) SignIn(ctx context.Context, email, password string) error {
	cc, err := saasdial.Dial(ctx, "", s.host)
	if err != nil {
		return errors.Wrap(err, "failed establish connection with Percona")
	}
	defer cc.Close() //nolint:errcheck

	resp, err := api.NewAuthAPIClient(cc).SignIn(ctx, &api.SignInRequest{Email: email, Password: password})
	if err != nil {
		return err
	}

	err = s.db.InTransaction(func(tx *reform.TX) error {
		params := models.ChangeSettingsParams{SessionID: resp.SessionId, Email: email}
		_, err := models.UpdateSettings(tx.Querier, &params)
		return err
	})
	if err != nil {
		return errors.Wrap(err, "failed to save session id")
	}

	return nil
}

// SignOut logouts that instance from Percona Platform account and removes session id.
func (s *Service) SignOut(ctx context.Context) error {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return err
	}

	if settings.SaaS.SessionID == "" {
		return errNoActiveSessions
	}

	cc, err := saasdial.Dial(ctx, settings.SaaS.SessionID, s.host)
	if err != nil {
		return errors.Wrap(err, "failed establish connection with Percona")
	}
	defer cc.Close() //nolint:errcheck

	_, err = api.NewAuthAPIClient(cc).SignOut(ctx, &api.SignOutRequest{})
	if err != nil {
		if st, ok := status.FromError(err); !ok || st.Code() != codes.InvalidArgument {
			return err
		}
	}

	err = s.db.InTransaction(func(tx *reform.TX) error {
		params := models.ChangeSettingsParams{LogOut: true}
		_, err := models.UpdateSettings(tx.Querier, &params)
		return err
	})
	if err != nil {
		return errors.Wrap(err, "failed to remove session id")
	}

	return nil
}

// refreshSession resets session timeout.
func (s *Service) refreshSession(ctx context.Context) error {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return err
	}

	if settings.SaaS.SessionID == "" {
		return errNoActiveSessions
	}

	cc, err := saasdial.Dial(ctx, settings.SaaS.SessionID, s.host)
	if err != nil {
		return errors.Wrap(err, "failed establish connection with Percona")
	}
	defer cc.Close() //nolint:errcheck

	_, err = api.NewAuthAPIClient(cc).RefreshSession(ctx, &api.RefreshSessionRequest{})
	if err != nil {
		return errors.Wrap(err, "failed to refresh session")
	}

	return nil
}
