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

package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/client/folders"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
)

// TestRecordingRuleTargetNeedsDataSourceWrite checks that only a user who can edit the Metrics data source can make a
// recording rule write to it, see PMM-15941.
func TestRecordingRuleTargetNeedsDataSourceWrite(t *testing.T) {
	gClient := pmmapitests.GetGrafanaClient(t)

	ds, err := gClient.Datasources.GetDataSourceByName("Metrics")
	require.NoError(t, err)
	metricsUID := ds.Payload.UID

	forceDelete := true
	createdFolder, err := gClient.Folders.CreateFolder(&models.CreateFolderCommand{Title: pmmapitests.TestString(t, "recording-rules")})
	require.NoError(t, err)
	folderUID := createdFolder.Payload.UID
	t.Cleanup(func() {
		_, _ = gClient.Folders.DeleteFolder(folders.NewDeleteFolderParams().WithFolderUID(folderUID).WithForceDeleteRules(&forceDelete))
	})

	editor := pmmapitests.TestString(t, "recording-editor")
	created, err := gClient.AdminUsers.AdminCreateUser(&models.AdminCreateUserForm{
		Login:    editor,
		Name:     editor,
		Email:    editor + "@percona.invalid",
		Password: models.Password(editor),
	})
	require.NoError(t, err)
	editorID := created.Payload.ID
	t.Cleanup(func() {
		_, _ = gClient.AdminUsers.AdminDeleteUser(editorID)
	})
	_, err = gClient.Org.UpdateOrgUserForCurrentOrg(editorID, &models.UpdateOrgUserCommand{Role: "Editor"})
	require.NoError(t, err)

	admin := pmmapitests.BaseURL.User.Username()
	adminPassword, _ := pmmapitests.BaseURL.User.Password()

	t.Run("an editor cannot create a recording rule that writes to Metrics", func(t *testing.T) {
		status, body := postRecordingRule(t, editor, editor, folderUID, "", metricsUID)
		assert.Equalf(t, http.StatusForbidden, status, "response: %s", body)
		assert.Contains(t, body, "target data source")
	})

	t.Run("an admin can", func(t *testing.T) {
		status, body := postRecordingRule(t, admin, adminPassword, folderUID, "", metricsUID)
		require.Equalf(t, http.StatusAccepted, status, "response: %s", body)
	})

	t.Run("an editor cannot change the admin's recording rule", func(t *testing.T) {
		ruleUID := recordingRuleUID(t, admin, adminPassword, folderUID)
		status, body := postRecordingRule(t, editor, editor, folderUID, ruleUID, metricsUID)
		assert.Equalf(t, http.StatusForbidden, status, "response: %s", body)
	})
}

const recordingRuleGroup = "recording"

func postRecordingRule(t *testing.T, login, password, folderUID, ruleUID, targetUID string) (int, string) {
	t.Helper()

	rule := map[string]any{
		"grafana_alert": map[string]any{
			"uid":   ruleUID,
			"title": "recording rule",
			"data": []map[string]any{{
				"refId":             "A",
				"datasourceUid":     targetUID,
				"relativeTimeRange": map[string]any{"from": 600, "to": 0},
				"model":             map[string]any{"refId": "A", "expr": "up"},
			}},
			// The name a reviewer would worry about: it is also a series PMM collects.
			"record": map[string]any{"metric": "up", "from": "A", "target_datasource_uid": targetUID},
		},
	}
	body, err := json.Marshal(map[string]any{"name": recordingRuleGroup, "interval": "1m", "rules": []any{rule}})
	require.NoError(t, err)

	return rulerRequest(t, http.MethodPost, login, password, "/graph/api/ruler/grafana/api/v1/rules/"+folderUID, body)
}

func recordingRuleUID(t *testing.T, login, password, folderUID string) string {
	t.Helper()

	status, body := rulerRequest(t, http.MethodGet, login, password,
		fmt.Sprintf("/graph/api/ruler/grafana/api/v1/rules/%s/%s", folderUID, recordingRuleGroup), nil)
	require.Equalf(t, http.StatusAccepted, status, "response: %s", body)

	var group struct {
		Rules []struct {
			GrafanaAlert struct {
				UID string `json:"uid"`
			} `json:"grafana_alert"`
		} `json:"rules"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &group))
	require.Len(t, group.Rules, 1)

	return group.Rules[0].GrafanaAlert.UID
}

func rulerRequest(t *testing.T, method, login, password, path string, body []byte) (int, string) {
	t.Helper()

	u := *pmmapitests.BaseURL
	u.User = nil
	u.Path = path

	req, err := http.NewRequestWithContext(pmmapitests.Context, method, u.String(), bytes.NewReader(body))
	require.NoError(t, err)
	req.SetBasicAuth(login, password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := pmmapitests.HTTPClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck

	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, string(b)
}
