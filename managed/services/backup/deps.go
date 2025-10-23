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
	"context"
	"time"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents"
	"github.com/percona/pmm/managed/services/minio"
)

// jobsService is a subset of methods of agents.JobsService used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type jobsService interface {
	StopJob(jobID string) error
	StartMySQLBackupJob(
		jobID string,
		pmmAgentID string,
		timeout time.Duration,
		name string,
		dbConfig *models.DBConfig,
		locationConfig *models.BackupLocationConfig,
		folder string,
		compression models.BackupCompression,
	) error
	StartMySQLRestoreBackupJob(
		jobID string,
		pmmAgentID string,
		serviceID string,
		timeout time.Duration,
		name string,
		locationConfig *models.BackupLocationConfig,
		folder string,
		compression models.BackupCompression,
	) error
	StartMongoDBBackupJob(
		service *models.Service,
		jobID string,
		pmmAgentID string,
		timeout time.Duration,
		name string,
		dbConfig *models.DBConfig,
		mode models.BackupMode,
		dataModel models.DataModel,
		locationConfig *models.BackupLocationConfig,
		folder string,
		compression models.BackupCompression,
	) error
	StartMongoDBRestoreBackupJob(
		service *models.Service,
		jobID string,
		pmmAgentID string,
		timeout time.Duration,
		name string,
		pbmBackupName string,
		dbConfig *models.DBConfig,
		dataModel models.DataModel,
		locationConfig *models.BackupLocationConfig,
		pitrTimestamp time.Time,
		folder string,
		compression models.BackupCompression,
	) error
}

type removalService interface {
	// DeleteArtifact deletes specified artifact along with files if specified.
	DeleteArtifact(storage Storage, artifactID string, removeFiles bool) error
	// TrimPITRArtifact removes first N records from PITR artifact. Removes snapshots, PITR chunks and corresponding data from database.
	TrimPITRArtifact(storage Storage, artifactID string, firstN int) error
}

// agentService is a subset of methods of agents.AgentService used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type agentService interface {
	PBMSwitchPITR(pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair, enabled bool) error
}

// versioner contains method for retrieving versions of different software.
type versioner interface {
	GetVersions(pmmAgentID string, softwares []agents.Software) ([]agents.Version, error)
}

type compatibilityService interface {
	// CheckSoftwareCompatibilityForService checks if all the necessary backup tools are installed,
	// and they are compatible with the db version.
	// Returns db version.
	CheckSoftwareCompatibilityForService(ctx context.Context, serviceID string) (string, error)
	// CheckArtifactCompatibility check compatibility between artifact and target database.
	CheckArtifactCompatibility(artifactID, targetDBVersion string) error
}

// pbmPITRService provides methods that help us inspect and manage PITR oplog slices.
type pbmPITRService interface {
	// ListPITRTimeranges list the available PITR timeranges for the given artifact in the provided location
	ListPITRTimeranges(ctx context.Context, locationClient Storage, location *models.BackupLocation, artifact *models.Artifact) ([]Timeline, error)
	// GetPITRFiles returns list of PITR chunks. If 'until' specified, returns only chunks created before that date, otherwise returns all artifact chunks.
	GetPITRFiles(ctx context.Context, locationClient Storage, location *models.BackupLocation, artifact *models.Artifact, until *time.Time) ([]*oplogChunk, error)
}

// Storage represents the interface for interacting with storage.
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
