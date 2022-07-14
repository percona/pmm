// Copyright (C) 2017 Percona LLC
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

// ErrSTTDisabled means that STT checks are disabled and can't be executed.
var ErrSTTDisabled = errors.New("STT is disabled")

// ErrAlertingDisabled means Integrated Alerting is disabled and IA APIs can't be executed.
var ErrAlertingDisabled = errors.New("Alerting is disabled")
