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

// Package encryption contains PMM encryption rotation functions.
package encryption

import (
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/encryption"
)

const (
	retries              = 5
	interval             = 5 * time.Second
	statusRunning        = "RUNNING"
	statusStopped        = "STOPPED"
	codeOK               = 0
	codePMMStopFailed    = 2
	codeEncryptionFailed = 3
	codePMMStartFailed   = 4
)

// RotateEncryptionKey will stop PMM server, decrypt data, create new encryption key and encrypt them and start PMM Server again.
func RotateEncryptionKey(sqlDB *sql.DB, dbName string) int {
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	err := stopPMMServer()
	if err != nil {
		logrus.Errorf("Failed to stop PMM Server: %+v", err)
		return codePMMStopFailed
	}

	err = rotateEncryptionKey(db, dbName)
	if err != nil {
		logrus.Errorf("Failed to rotate encryption key: %+v", err)
		return codeEncryptionFailed
	}

	err = startPMMServer()
	if err != nil {
		logrus.Errorf("Failed to start PMM Server: %+v", err)
		return codePMMStartFailed
	}

	return codeOK
}

func startPMMServer() error {
	logrus.Infoln("Starting PMM Server")
	if pmmServerStatus(statusRunning) {
		return nil
	}

	cmd := exec.Command("supervisorctl", "start pmm-managed")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, output)
	}

	if !pmmServerStatusWithRetries(statusRunning) {
		return errors.New("cannot start pmm-managed")
	}

	return nil
}

func stopPMMServer() error {
	logrus.Infoln("Stopping PMM Server")
	if pmmServerStatus(statusStopped) {
		return nil
	}

	cmd := exec.Command("supervisorctl", "stop pmm-managed")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, output)
	}

	if !pmmServerStatusWithRetries(statusStopped) {
		return errors.New("cannot stop pmm-managed")
	}

	return nil
}

func pmmServerStatus(status string) bool {
	cmd := exec.Command("supervisorctl", "status pmm-managed")
	output, _ := cmd.CombinedOutput()

	return strings.Contains(string(output), strings.ToUpper(status))
}

func pmmServerStatusWithRetries(status string) bool {
	for i := 0; i < retries; i++ {
		if !pmmServerStatus(status) {
			logrus.Infoln("Retry...")
			time.Sleep(interval)
			continue
		}

		return true
	}

	return false
}

func rotateEncryptionKey(db *reform.DB, dbName string) error {
	return db.InTransaction(func(tx *reform.TX) error {
		logrus.Infof("DB %s is being decrypted", dbName)
		err := models.DecryptDB(tx, dbName, models.DefaultAgentEncryptionColumnsV3)
		if err != nil {
			return err
		}
		logrus.Infof("DB %s is successfully decrypted", dbName)

		logrus.Infoln("Rotating encryption key")
		err = encryption.RotateEncryptionKey()
		if err != nil {
			return err
		}
		logrus.Infof("New encryption key generated")

		logrus.Infof("DB %s is being encrypted", dbName)
		err = models.EncryptDB(tx, dbName, models.DefaultAgentEncryptionColumnsV3)
		if err != nil {
			if e := encryption.RestoreOldEncryptionKey(); e != nil {
				return errors.Wrap(err, e.Error())
			}
			return err
		}
		logrus.Infof("DB %s is successfully encrypted", dbName)

		return nil
	})
}
