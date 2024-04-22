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

package services

import "github.com/pkg/errors"

var (
	// ErrAdvisorsDisabled means that advisors checks are disabled and can't be executed.
	ErrAdvisorsDisabled = errors.New("Advisor checks are disabled")
	// ErrLocationFolderPairAlreadyUsed returned when location-folder pair already in use and cannot be used for backup.
	ErrLocationFolderPairAlreadyUsed = errors.New("location-folder pair already used")

	// ErrAlertingDisabled means Integrated Alerting is disabled and IA APIs can't be executed.
	ErrAlertingDisabled = errors.New("Alerting is disabled") // TODO Looks like this error is unused.
)
