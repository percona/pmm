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

import (
	"context"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/minio"
)

type Storage interface {
	// FileStat returns file info. It returns error if file is empty or not exists.
	FileStat(ctx context.Context, endpoint, accessKey, secretKey, bucketName, name string) (minio.FileInfo, error)

	// List scans path with prefix and returns all files with given suffix.
	// Both prefix and suffix can be omitted.
	List(ctx context.Context, endpoint, accessKey, secretKey, bucketName, prefix, suffix string) ([]minio.FileInfo, error)
	// Remove removes single objects from storage.
	Remove(ctx context.Context, endpoint, accessKey, secretKey, bucketName, objectName string) error
	// RemoveRecursive removes objects recursively from storage with given prefix.
	RemoveRecursive(ctx context.Context, endpoint, accessKey, secretKey, bucketName, prefix string) (rerr error)
}

// Location2Storage returns storage client depending on location type.
func Location2Storage(location *models.BackupLocation) Storage {
	switch location.Type {
	case models.S3BackupLocationType:
		return minio.New()
	default:
		return nil
	}
}
