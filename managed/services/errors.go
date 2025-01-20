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

package services

import "github.com/pkg/errors"

var (
	// ErrAdvisorsDisabled means that advisors checks are disabled and can't be called.
	ErrAdvisorsDisabled = errors.New("Advisor checks are disabled")

	// ErrLocationFolderPairAlreadyUsed returned when location-folder pair already in use and cannot be used for backup.
	ErrLocationFolderPairAlreadyUsed = errors.New("location-folder pair already used")

	// ErrAlertingDisabled means Percona Alerting is disabled and its APIs can't be called.
	ErrAlertingDisabled = errors.New("Alerting is disabled")

	// ErrAzureDisabled means Azure Monitoring is disabled and its APIs can't be called.
	ErrAzureDisabled = errors.New("Azure monitoring is disabled")

	// ErrPMMUpdatesDisabled means PMM server updates are disabled and calls to query/start updates are not allowed.
	ErrPMMUpdatesDisabled = errors.New("PMM updates are disabled")
)
