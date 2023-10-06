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

package backup

import "github.com/pkg/errors"

var (
	// ErrIncompatibleService is returned when the service is incompatible for making a backup or restore.
	ErrIncompatibleService = errors.New("incompatible service")
	// ErrXtrabackupNotInstalled is returned if some xtrabackup component is missing.
	ErrXtrabackupNotInstalled = errors.New("xtrabackup is not installed")
	// ErrInvalidXtrabackup is returned if xtrabackup components have different version.
	ErrInvalidXtrabackup = errors.New("invalid installation of the xtrabackup")
	// ErrIncompatibleXtrabackup is returned if xtrabackup is not compatible with the MySQL.
	ErrIncompatibleXtrabackup = errors.New("incompatible xtrabackup")
	// ErrIncompatiblePBM is returned if installed PBM version not fits to perform requested operation.
	ErrIncompatiblePBM = errors.New("incompatible pbm")
	// ErrIncompatibleTargetMySQL is returned if target version of MySQL is not compatible for restoring selected artifact.
	ErrIncompatibleTargetMySQL = errors.New("incompatible version of target mysql")
	// ErrIncompatibleTargetMongoDB is returned if target version of MongoDB is not compatible for restoring selected artifact.
	ErrIncompatibleTargetMongoDB = errors.New("incompatible version of target mongodb")
	// ErrComparisonImpossible is returned when comparison of versions is impossible for some reasons.
	ErrComparisonImpossible = errors.New("cannot compare software versions")
	// ErrIncompatibleDataModel is returned if the specified data model (logical or physical) is not compatible with other parameters.
	ErrIncompatibleDataModel = errors.New("the specified backup model is not compatible with other parameters")
	// ErrIncompatibleLocationType is returned if the specified location type (local or s3) is not compatible with other parameters.
	ErrIncompatibleLocationType = errors.New("the specified location type is not compatible with other parameters")
	// ErrIncompatibleArtifactMode is returned if artifact backup mode is incompatible with other parameters.
	ErrIncompatibleArtifactMode = errors.New("artifact backup mode is not compatible with other parameters")
	// ErrTimestampOutOfRange is returned if timestamp value is out of allowed range.
	ErrTimestampOutOfRange = errors.New("timestamp value is out of range")
	// ErrAnotherOperationInProgress is returned if there are other operations in progress that prevent running the requested one.
	ErrAnotherOperationInProgress = errors.New("another operation in progress")
	// ErrArtifactNotReady is returned when artifact not ready to be restored, i.e. not in success status.
	ErrArtifactNotReady = errors.New("artifact not in success status")
	// ErrIncorrectArtifactStatus is returned when artifact status doesn't fit to proceed with action.
	ErrIncorrectArtifactStatus = errors.New("incorrect artifact status")
)
