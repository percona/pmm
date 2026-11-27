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
	"errors"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// Scan profiles pmm-managed requests; the monitored sweep keeps its own path and is not one of these.
const (
	scanProfileManaged        = "managed"
	scanProfileImportCheck    = "import_check"
	scanProfileInstallCheck   = "install_check"
	scanProfileUninstallCheck = "uninstall_check"
)

// subjectKindMember labels a managed member as a scan subject, matching the kind the start gate reads
// its observation back under.
const subjectKindMember = "member"

// scanSubject is one host a scan collects facts for, carrying the key pmm-managed stores and reads the
// observation under so the write and the read agree.
type scanSubject struct {
	SubjectKind string  `json:"subject_kind"`
	SubjectKey  string  `json:"subject_key"`
	Host        string  `json:"host"`
	ServiceID   string  `json:"service_id,omitempty"`
	Port        *uint16 `json:"port,omitempty"`
}

// scanRequest is the body pmm-managed POSTs to request a scan: the profile that picks the fact set,
// an idempotency key so a retry fulfils once, the subject as a driver operation or a cluster, and the
// per-host subjects to collect facts for.
type scanRequest struct {
	Profile        string        `json:"profile"`
	IdempotencyKey string        `json:"idempotency_key"`
	OperationID    string        `json:"operation_id,omitempty"`
	ClusterID      string        `json:"cluster_id,omitempty"`
	Subjects       []scanSubject `json:"subjects"`
}

// inventoryScanRequester requests scans from PMM Extensions' om_inventory app, fulfilling the driver's
// ScanRequester seam and the scheduler's per-cluster requests.
type inventoryScanRequester struct {
	db  *reform.DB
	l   *logrus.Entry
	app extensionsApp
}

// newInventoryScanRequester builds a requester against om_inventory on the configured Extensions client.
func newInventoryScanRequester(db *reform.DB, l *logrus.Entry, client *extensionsClient) *inventoryScanRequester {
	return &inventoryScanRequester{db: db, l: l, app: client.app(probeAppModule)}
}

// RequestScan fulfils the driver's rescan seam: it asks om_inventory to scan an operation under the
// profile the check set named, keyed so a retried request fulfils once. An install names its target
// nodes and a cluster operation its members, so the request always carries a non-null subject list.
func (r *inventoryScanRequester) RequestScan(operationID, profile, idempotencyKey string) error {
	subjects := []scanSubject{}
	if profile == scanProfileInstallCheck {
		nodes, err := r.installSubjects(operationID)
		if err != nil {
			return err
		}
		subjects = append(subjects, nodes...)
	} else {
		members, err := r.operationMemberSubjects(operationID)
		if err != nil {
			return err
		}
		subjects = append(subjects, members...)
	}
	return r.post(context.Background(), scanRequest{
		Profile: profile, IdempotencyKey: idempotencyKey, OperationID: operationID, Subjects: subjects,
	})
}

// operationMemberSubjects lists a cluster operation's members as the per-host subjects its scan collects
// facts for; an operation with no cluster carries none, and om_inventory keys the scan on the operation.
func (r *inventoryScanRequester) operationMemberSubjects(operationID string) ([]scanSubject, error) {
	row, err := r.db.FindByPrimaryKeyFrom(models.OmOperationTable, operationID)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read operation '%s': %w", operationID, err)
	}
	op := row.(*models.OmOperation) //nolint:forcetypeassert
	if op.ClusterID == nil {
		return nil, nil
	}
	return r.memberSubjects(*op.ClusterID)
}

// installSubjects lists an install operation's target nodes as install-node scan subjects; an operation
// of another kind carries none here, its subjects coming from operationMemberSubjects instead.
func (r *inventoryScanRequester) installSubjects(operationID string) ([]scanSubject, error) {
	row, err := r.db.FindByPrimaryKeyFrom(models.OmOperationTable, operationID)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read operation '%s': %w", operationID, err)
	}
	op := row.(*models.OmOperation) //nolint:forcetypeassert
	if op.Kind != installKind {
		return nil, nil
	}
	params := installParamsOf(op)
	subjects := make([]scanSubject, 0, len(params.NodeIDs))
	for _, nodeID := range params.NodeIDs {
		subjects = append(subjects, scanSubject{
			SubjectKind: subjectKindInstallNode, SubjectKey: nodeID, Host: params.Hosts[nodeID],
		})
	}
	return subjects, nil
}

// RequestClusterScan asks om_inventory for a scan of one cluster's members, as the scheduler does on
// the settings cadence; the key folds repeat requests in the same window into one fulfilment.
func (r *inventoryScanRequester) RequestClusterScan(ctx context.Context, clusterID, profile, idempotencyKey string) error {
	subjects, err := r.memberSubjects(clusterID)
	if err != nil {
		return err
	}
	return r.post(ctx, scanRequest{
		Profile: profile, IdempotencyKey: idempotencyKey, ClusterID: clusterID, Subjects: subjects,
	})
}

// memberSubjects lists a cluster's live members as the per-host subjects a scan collects facts for,
// each keyed by its member id so the stored observation matches the read.
func (r *inventoryScanRequester) memberSubjects(clusterID string) ([]scanSubject, error) {
	rows, err := r.db.SelectAllFrom(models.OmClusterMemberTable, "WHERE cluster_id = $1 AND removed_at IS NULL ORDER BY id", clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to list members of cluster '%s': %w", clusterID, err)
	}
	subjects := make([]scanSubject, 0, len(rows))
	for _, row := range rows {
		member := row.(*models.OmClusterMember) //nolint:forcetypeassert
		subject := scanSubject{SubjectKind: subjectKindMember, SubjectKey: member.ID, Host: member.Host}
		if member.ServiceID != nil {
			subject.ServiceID = *member.ServiceID
			subject.Port = r.memberPort(*member.ServiceID)
		}
		subjects = append(subjects, subject)
	}
	return subjects, nil
}

// memberPort returns the port the member's registered service listens on, or nil when the service
// cannot be resolved, so the scan request's own port fallback applies.
func (r *inventoryScanRequester) memberPort(serviceID string) *uint16 {
	service, err := models.FindServiceByID(r.db.Querier, serviceID)
	if err != nil {
		r.l.WithField("service_id", serviceID).WithError(err).Warn("failed to resolve the member service port; the scan falls back to the default")
		return nil
	}
	return service.Port
}

// post sends one scan request as pmm-managed's own work and maps a failure to a profile-named error.
func (r *inventoryScanRequester) post(ctx context.Context, body scanRequest) error {
	err := r.app.call(asService(ctx), inventoryCall{method: http.MethodPost, path: "scans", body: body}, nil)
	if err != nil {
		return fmt.Errorf("failed to request %s scan: %w", body.Profile, err)
	}
	return nil
}
