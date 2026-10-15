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

package annotations

import (
	"context"
	"time"
)

// syncStore is a subset of methods of Store used by the sync job.
// We use it instead of real type for testing.
type syncStore interface {
	Write(ctx context.Context, events []Event) error
	Delete(ctx context.Context, ids []int64) error
	Keys(ctx context.Context, window time.Duration) ([]Key, error)
}
