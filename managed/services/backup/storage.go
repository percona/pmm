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

import (
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/minio"
)

// GetStorageForLocation returns storage client depending on location type.
func GetStorageForLocation(location *models.BackupLocation) Storage {
	switch location.Type {
	case models.S3BackupLocationType:
		return minio.New()
	default:
		return nil
	}
}
