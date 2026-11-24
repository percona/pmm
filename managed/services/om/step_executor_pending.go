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

	"github.com/percona/pmm/managed/models"
)

// pendingStepExecutorProvider stands in for E0-T3's on-host step execution; no registered kind ships
// steps yet, so the executor it returns is resolved but never asked to run one.
type pendingStepExecutorProvider struct{}

// Executor returns an executor that refuses to run, which no step-less kind reaches.
func (pendingStepExecutorProvider) Executor(*models.OmOperation, OperationKind) (StepExecutor, error) {
	return pendingStepExecutor{}, nil
}

// pendingStepExecutor refuses every dispatch until real on-host execution is wired.
type pendingStepExecutor struct{}

// Run reports that on-host step execution is not wired yet.
func (pendingStepExecutor) Run(context.Context, StepPlan, string) (StepResultKind, []byte, error) {
	return "", nil, errors.New("on-host step execution is not wired yet")
}
