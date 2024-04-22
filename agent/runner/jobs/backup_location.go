// Copyright (C) 2024 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jobs

// BackupLocationType represents BackupLocation type as stored in database.
type BackupLocationType string

// BackupLocation types. Same as in managed/models/location_model.go.
const (
	S3BackupLocationType         BackupLocationType = "s3"
	FilesystemBackupLocationType BackupLocationType = "filesystem"
)

// S3LocationConfig contains required properties for accessing S3 Bucket.
type S3LocationConfig struct {
	Endpoint     string
	AccessKey    string
	SecretKey    string
	BucketName   string
	BucketRegion string
}

// FilesystemBackupLocationConfig contains config for local storage.
type FilesystemBackupLocationConfig struct {
	Path string
}

// BackupLocationConfig groups all backup locations configs.
type BackupLocationConfig struct {
	Type                    BackupLocationType
	S3Config                *S3LocationConfig
	FilesystemStorageConfig *FilesystemBackupLocationConfig
}
