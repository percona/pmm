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

package models

import (
	"database/sql"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/utils/encryption"
)

func RotateEncryptionKey(sqlDB *sql.DB, dbName string) int {
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	err := stopPMMServer()
	if err != nil {
		logrus.Errorf("Failed to stop PMM Server: %+v", err)
		return 2
	}

	err = rotateEncryptionKey(db, dbName)
	if err != nil {
		logrus.Errorf("Failed to rotate encryption key: %+v", err)
		return 3
	}

	err = startPMMServer()
	if err != nil {
		logrus.Errorf("Failed to start PMM Server: %+v", err)
		return 4
	}

	return 0
}

func startPMMServer() error {
	if isPMMServerStatus("RUNNING") {
		return nil
	}

	cmd := exec.Command("supervisorctl", "start pmm-managed")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, output)
	}

	if !isPMMServerStatus("RUNNING") {
		return errors.New("cannot start pmm-managed")
	}

	return nil
}

func stopPMMServer() error {
	if isPMMServerStatus("STOPPED") {
		return nil
	}

	cmd := exec.Command("supervisorctl", "stop pmm-managed")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, output)
	}

	if !isPMMServerStatus("STOPPED") {
		return errors.New("cannot stop pmm-managed")
	}

	return nil
}

func isPMMServerStatus(status string) bool {
	cmd := exec.Command("supervisorctl", "status pmm-managed")
	output, _ := cmd.CombinedOutput()

	return strings.Contains(string(output), strings.ToUpper(status))
}

func rotateEncryptionKey(db *reform.DB, dbName string) error {
	return db.InTransaction(func(tx *reform.TX) error {
		logrus.Infof("DB %s is being decrypted", dbName)
		err := DecryptDB(tx, dbName, DefaultAgentEncryptionColumns)
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
		err = EncryptDB(tx, dbName, DefaultAgentEncryptionColumns)
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
