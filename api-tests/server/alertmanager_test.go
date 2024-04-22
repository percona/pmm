// Copyright (C) 2024 Percona LLC
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

package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	"github.com/percona/pmm/api/alertmanager/amclient"
	"github.com/percona/pmm/api/alertmanager/amclient/alert"
)

func TestAlertManager(t *testing.T) {
	t.Run("TestEndsAtForFailedChecksAlerts", func(t *testing.T) {
		if !pmmapitests.RunSTTTests {
			t.Skip("Skipping STT tests until we have environment: https://jira.percona.com/browse/PMM-5106")
		}

		defer restoreSettingsDefaults(t)

		// sync with pmm-managed
		const (
			resolveTimeoutFactor  = 3
			defaultResendInterval = 2 * time.Second
		)

		// 120 sec ping for failed checks alerts to appear in alertmanager
		for i := 0; i < 120; i++ {
			res, err := amclient.Default.Alert.GetAlerts(&alert.GetAlertsParams{
				Filter:  []string{"stt_check=1"},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			if len(res.Payload) == 0 {
				time.Sleep(1 * time.Second)
				continue
			}

			require.NotEmpty(t, res.Payload, "No alerts met")

			// TODO: Expand this test once we are silencing/removing alerts.
			alertTTL := resolveTimeoutFactor * defaultResendInterval
			for _, v := range res.Payload {
				// Since the `EndsAt` timestamp is always resolveTimeoutFactor times the
				// `resendInterval` in the future from `UpdatedAt`
				// we check whether they lie in that time alertTTL.
				assert.WithinDuration(t, time.Time(*v.EndsAt), time.Time(*v.UpdatedAt), alertTTL)
				assert.Greater(t, v.EndsAt, v.UpdatedAt)
			}
			break
		}
	})
}
