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

// Package encryption contains PMM encryption key rotation functions.
package encryption

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/encryption"
)

const (
	statusRetries  = 5
	statusInterval = 5 * time.Second
	sweepRetries   = 60
	sweepInterval  = 2 * time.Second
	statusRunning  = "RUNNING"

	codeOK             = 0
	codeRotationFailed = 2
	codeRestartFailed  = 3
	codeSweepFailed    = 4
	codePruneFailed    = 5
)

// RotationParams configure RotateEncryptionKey.
type RotationParams struct {
	// Prune removes the retired keys from the keyset once no stored value
	// references them.
	Prune bool
	// OtherHANodesStopped confirms that PMM Server is stopped on every other
	// node of an HA cluster; rotation is refused in HA without it.
	OtherHANodesStopped bool
}

// RotateEncryptionKey adds a new primary key to the encryption keyset — the
// previous keys stay in the keyset, so all stored values remain readable and
// the database is never held decrypted at rest. pmm-managed is then restarted
// to reload the keyset; its startup migration re-encrypts every stored secret
// with the new primary key, which this function waits for. With Prune, the
// retired keys are removed from the keyset once no stored value references
// them.
//
// In HA, every other node must be stopped first (OtherHANodesStopped): only
// this node's pmm-managed is restarted, and the other nodes cannot read values
// encrypted with the new key until the key file is copied to them.
func RotateEncryptionKey(sqlDB *sql.DB, params RotationParams) (int, error) {
	keyPath := encryption.DefaultKeyPath()
	ha, _ := strconv.ParseBool(os.Getenv("PMM_HA_ENABLE"))
	if ha && !params.OtherHANodesStopped {
		return codeRotationFailed, fmt.Errorf("in HA mode, stop PMM Server on every other node first: rotation restarts only "+
			"this node's pmm-managed, and the other nodes could not read values encrypted with the new key. "+
			"Then run pmm-encryption-rotation --ha-other-nodes-stopped on this node, copy %s to every other node and start them",
			keyPath)
	}

	provider := encryption.NewFileKeyProvider(keyPath)

	newKeyID, err := encryption.AddNewPrimaryKey(provider)
	if err != nil {
		return codeRotationFailed, fmt.Errorf("failed to add new encryption key: %w", err)
	}
	logrus.Infof("Added new primary encryption key %d", newKeyID)

	err = restartPMMServer()
	if err != nil {
		return codeRestartFailed, fmt.Errorf("failed to restart PMM Server: %w", err)
	}

	cipher, err := encryption.LoadCipher(provider)
	if err != nil {
		return codeRotationFailed, err
	}

	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	err = waitForReencryption(db.Querier, cipher)
	if err != nil {
		return codeSweepFailed, err
	}
	logrus.Infoln("All stored secrets are re-encrypted with the new key")

	if params.Prune {
		backups := models.MigrationBackupFiles()
		if len(backups) != 0 {
			logrus.Warnf("Pruning removes the keys the values in %s are encrypted with; "+
				"keep a copy of the key file from before this rotation to read them later.", strings.Join(backups, ", "))
		}
		retired, err := encryption.PruneRetiredKeys(provider)
		if err != nil {
			return codePruneFailed, fmt.Errorf("failed to prune retired encryption keys: %w", err)
		}
		logrus.Infof("Removed %d retired encryption key(s) from the keyset", len(retired))
	}

	if ha {
		logrus.Warnf("Copy %s to every other PMM Server node before starting them: they cannot read values encrypted with the new key.",
			keyPath)
	}

	return codeOK, nil
}

func restartPMMServer() error {
	logrus.Infoln("Restarting PMM Server")

	cmd := exec.CommandContext(context.Background(), "supervisorctl", "restart", "pmm-managed")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, output)
	}

	if !pmmServerStatusWithRetries(statusRunning) {
		return errors.New("pmm-managed did not reach RUNNING state")
	}

	return nil
}

func pmmServerStatus(status string) bool {
	cmd := exec.Command("supervisorctl", "status", "pmm-managed") //nolint:noctx
	output, _ := cmd.CombinedOutput()

	return strings.Contains(string(output), strings.ToUpper(status))
}

func pmmServerStatusWithRetries(status string) bool {
	for range statusRetries {
		if !pmmServerStatus(status) {
			logrus.Infoln("Retry...")
			time.Sleep(statusInterval)
			continue
		}

		return true
	}

	return false
}

// waitForReencryption polls until pmm-managed's startup migration has
// re-encrypted every stored secret with the current primary key.
func waitForReencryption(q *reform.Querier, cipher *encryption.Cipher) error {
	var lastCount int
	for range sweepRetries {
		agentIDs, err := models.AgentsNeedingReencryption(q, cipher)
		if err != nil {
			return err
		}
		locationIDs, err := models.LocationsNeedingReencryption(q, cipher)
		if err != nil {
			return err
		}
		checkStale, err := models.KeyCheckNeedsReencryption(q, cipher)
		if err != nil {
			return err
		}
		lastCount = len(agentIDs) + len(locationIDs)
		if checkStale {
			lastCount++
		}
		if lastCount == 0 {
			return nil
		}

		logrus.Infof("%d row(s) still need re-encryption, waiting...", lastCount)
		time.Sleep(sweepInterval)
	}

	return fmt.Errorf("timed out waiting for re-encryption: %d row(s) still reference retired keys", lastCount)
}
