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
	"context"
	"fmt"
	"net/http"
)

// Scan profiles pmm-managed requests; the monitored sweep keeps its own path and is not one of these.
const (
	scanProfileManaged        = "managed"
	scanProfileImportCheck    = "import_check"
	scanProfileInstallCheck   = "install_check"
	scanProfileUninstallCheck = "uninstall_check"
)

// scanRequest is the body pmm-managed POSTs to request a scan: the profile that picks the fact set,
// an idempotency key so a retry fulfils once, and the subject as a driver operation or a cluster.
type scanRequest struct {
	Profile        string `json:"profile"`
	IdempotencyKey string `json:"idempotency_key"`
	OperationID    string `json:"operation_id,omitempty"`
	ClusterID      string `json:"cluster_id,omitempty"`
}

// inventoryScanRequester requests scans from PMM Extensions' om_inventory app, fulfilling the driver's
// ScanRequester seam and the scheduler's per-cluster requests.
type inventoryScanRequester struct {
	app extensionsApp
}

// newInventoryScanRequester builds a requester against om_inventory on the configured Extensions client.
func newInventoryScanRequester(client *extensionsClient) *inventoryScanRequester {
	return &inventoryScanRequester{app: client.app(probeAppModule)}
}

// RequestScan fulfils the driver's rescan seam: it asks om_inventory to scan an operation under the
// profile the check set named, keyed so a retried request fulfils once.
func (r *inventoryScanRequester) RequestScan(operationID, profile, idempotencyKey string) error {
	return r.post(context.Background(), scanRequest{
		Profile: profile, IdempotencyKey: idempotencyKey, OperationID: operationID,
	})
}

// RequestClusterScan asks om_inventory for a scan of one cluster, as the scheduler does on the settings
// cadence; the key folds repeat requests in the same window into one fulfilment.
func (r *inventoryScanRequester) RequestClusterScan(ctx context.Context, clusterID, profile, idempotencyKey string) error {
	return r.post(ctx, scanRequest{
		Profile: profile, IdempotencyKey: idempotencyKey, ClusterID: clusterID,
	})
}

// post sends one scan request as pmm-managed's own work and maps a failure to a profile-named error.
func (r *inventoryScanRequester) post(ctx context.Context, body scanRequest) error {
	err := r.app.call(asService(ctx), inventoryCall{method: http.MethodPost, path: "scans", body: body}, nil)
	if err != nil {
		return fmt.Errorf("failed to request %s scan: %w", body.Profile, err)
	}
	return nil
}
