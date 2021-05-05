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

package saasdial

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

const platformAuthType = "PP-1"

type platformAuth struct {
	sessionID string
}

// GetRequestMetadata implements credentials.PerRPCCredentials interface.
func (b *platformAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": platformAuthType + " " + b.sessionID,
	}, nil
}

// RequireTransportSecurity implements credentials.PerRPCCredentials interface.
func (*platformAuth) RequireTransportSecurity() bool {
	return true
}

// LogoutIfInvalidAuth will force a log out if SaaS credentials become invalid.
// Note: This is a special case that occurs if a user's password is reset from
// the Okta dashboard but the PMM server is left logged in and is not able to log out
// after the password reset.
func LogoutIfInvalidAuth(db *reform.DB, l *logrus.Entry, platformErr error) error {
	l.Warn("Platform session invalid, forcing a logout.")
	if st, _ := status.FromError(platformErr); st.Code() == codes.Unauthenticated {
		e := db.InTransaction(func(tx *reform.TX) error {
			params := models.ChangeSettingsParams{LogOut: true}
			_, err := models.UpdateSettings(tx.Querier, &params)
			return err
		})
		if e != nil {
			return errors.Wrap(e, "failed to remove session id")
		}
	}
	return nil
}

// check interfaces
var (
	_ credentials.PerRPCCredentials = (*platformAuth)(nil)
)
