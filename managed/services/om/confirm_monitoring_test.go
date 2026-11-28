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

package om

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfirmMonitoringStep(t *testing.T) {
	t.Parallel()

	withService := map[string]extensionsHost{"pmm-client-node02": {Services: []extensionsService{{ServiceID: "svc-1"}}}}
	withoutService := map[string]extensionsHost{"pmm-client-node02": {}}

	cases := []struct {
		name       string
		runStatus  string
		registered bool
		hosts      map[string]extensionsHost
		want       string
	}{
		{
			name:       "a succeeded run waits while the inventory has no service on the host",
			runStatus:  bootstrapRunSucceeded,
			registered: false,
			hosts:      withoutService,
			want:       bootstrapStepRunning,
		},
		{
			name:       "a succeeded run is confirmed once the inventory sees the service",
			runStatus:  bootstrapRunSucceeded,
			registered: false,
			hosts:      withService,
			want:       bootstrapStepSucceeded,
		},
		{
			// Uninstall removes the service, so the live inventory no longer shows it. The registered
			// run must not go back to waiting, or its node stays "Installing" forever.
			name:       "a registered run stays confirmed after its service is removed",
			runStatus:  bootstrapRunSucceeded,
			registered: true,
			hosts:      withoutService,
			want:       bootstrapStepSucceeded,
		},
		{
			name:       "a running run is never confirmed",
			runStatus:  bootstrapRunRunning,
			registered: true,
			hosts:      withService,
			want:       bootstrapStepPending,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			step := confirmMonitoringStep(tc.runStatus, "pmm-client-node02", tc.registered, tc.hosts)
			assert.Equal(t, confirmMonitoringStepName, step.GetName())
			assert.Equal(t, tc.want, step.GetStatus())
		})
	}
}
