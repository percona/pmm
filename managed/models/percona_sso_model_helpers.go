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

package models

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

var (
	perconaSSOMtx           sync.Mutex
	ErrNotConnectedToPortal = errors.New("PMM Server is not connected to Portal")
)

// GetPerconaSSODetails returns PerconaSSODetails if there are any, error otherwise.
// Access token is automatically refreshed if it is expired.
// Get, check eventually refresh done in one tx.
func GetPerconaSSODetails(ctx context.Context, q *reform.Querier) (*PerconaSSODetails, error) {
	perconaSSOMtx.Lock()
	defer perconaSSOMtx.Unlock()

	ssoDetails, err := q.SelectOneFrom(PerconaSSODetailsTable, "")
	if err != nil {
		if err == reform.ErrNoRows {
			return nil, ErrNotConnectedToPortal
		}
		return nil, errors.Wrap(err, "failed to get Percona SSO Details")
	}

	details := ssoDetails.(*PerconaSSODetails)
	if details.isAccessTokenExpired() {
		refreshedToken, err := details.refreshAndGetAccessToken(ctx, q)
		if err != nil {
			return nil, errors.Wrap(err, "failed to insert Percona SSO Details")
		}
		details.AccessToken = refreshedToken
	}

	return details, nil
}

func (sso *PerconaSSODetails) refreshAndGetAccessToken(ctx context.Context, q *reform.Querier) (*PerconaSSOAccessToken, error) {
	values := url.Values{
		"grant_type": []string{"client_credentials"},
		"scope":      []string{sso.Scope},
	}
	requestURL := fmt.Sprintf("%s/token?%s", sso.IssuerURL, values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, nil)
	if err != nil {
		return nil, err
	}

	authHeader := base64.StdEncoding.EncodeToString([]byte(sso.PMMManagedClientID + ":" + sso.PMMManagedClientSecret))
	h := req.Header
	h.Add("Authorization", "Basic "+authHeader)
	h.Add("Accept", "application/json")
	h.Add("Content-Type", "application/x-www-form-urlencoded")

	timeBeforeRequest := time.Now()
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close() //nolint:errcheck

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get access token, response body: %s", bodyBytes)
	}

	var accessToken *PerconaSSOAccessToken
	if err := json.Unmarshal(bodyBytes, &accessToken); err != nil {
		return nil, err
	}
	accessToken.ExpiresAt = timeBeforeRequest.Add(time.Duration(accessToken.ExpiresIn) * time.Second)
	sso.AccessToken = accessToken

	if err := q.UpdateColumns(sso, "access_token"); err != nil {
		return nil, err
	}

	return accessToken, nil
}

func (sso *PerconaSSODetails) isAccessTokenExpired() bool {
	if sso == nil || sso.AccessToken == nil {
		return true
	}

	return time.Now().After(sso.AccessToken.ExpiresAt.Add(-time.Minute * 5))
}

// DeletePerconaSSODetails removes all stored DeletePerconaSSODetails.
func DeletePerconaSSODetails(q *reform.Querier) error {
	_, err := q.DeleteFrom(PerconaSSODetailsTable, "")
	if err != nil {
		return errors.Wrap(err, "failed to delete Percona SSO Details")
	}
	return nil
}

// InsertPerconaSSODetails inserts a new Percona SSO details.
func InsertPerconaSSODetails(q *reform.Querier, ssoDetails *PerconaSSODetailsInsert) error {
	details := &PerconaSSODetails{
		IssuerURL:              ssoDetails.IssuerURL,
		PMMManagedClientID:     ssoDetails.PMMManagedClientID,
		PMMManagedClientSecret: ssoDetails.PMMManagedClientSecret,
		GrafanaClientID:        ssoDetails.GrafanaClientID,
		Scope:                  ssoDetails.Scope,
		OrganizationID:         ssoDetails.OrganizationID,
		PMMServerName:          ssoDetails.PMMServerName,
	}

	if err := q.Save(details); err != nil {
		return errors.Wrap(err, "failed to insert Percona SSO Details")
	}

	return nil
}
