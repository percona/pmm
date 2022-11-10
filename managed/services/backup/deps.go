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

package backup

import (
	"context"
	"time"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents"
	"github.com/percona/pmm/managed/services/minio"
)

//go:generate ../../../bin/mockery -name=jobsService -case=snake -inpkg -testonly
//go:generate ../../../bin/mockery -name=s3 -case=snake -inpkg -testonly
//go:generate ../../../bin/mockery -name=agentService -case=snake -inpkg -testonly
//go:generate ../../../bin/mockery -name=versioner -case=snake -inpkg -testonly
//go:generate ../../../bin/mockery -name=pitrLocationClient -case=snake -inpkg -testonly
//go:generate ../../../bin/mockery -name=compatibilityService -case=snake -inpkg -testonly
//go:generate ../../../bin/mockery -name=pitrLocationClient -case=snake -inpkg -testonly
//go:generate ../../../bin/mockery -name=pitrTimerangeService -case=snake -inpkg -testonly

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
	) error
	StartMySQLRestoreBackupJob(
		jobID string,
		pmmAgentID string,
		serviceID string,
		timeout time.Duration,
		name string,
		locationConfig *models.BackupLocationConfig,
	) error
	StartMongoDBBackupJob(
		jobID string,
		pmmAgentID string,
		timeout time.Duration,
		name string,
		dbConfig *models.DBConfig,
		mode models.BackupMode,
		dataModel models.DataModel,
		locationConfig *models.BackupLocationConfig,
	) error
	StartMongoDBRestoreBackupJob(
		jobID string,
		serviceID string,
		pmmAgentID string,
		timeout time.Duration,
		name string,
		dbConfig *models.DBConfig,
		dataModel models.DataModel,
		locationConfig *models.BackupLocationConfig,
		pitrTimestamp time.Time,
	) error
}

type s3 interface {
	RemoveRecursive(ctx context.Context, endpoint, accessKey, secretKey, bucketName, prefix string) error
}

type removalService interface {
	DeleteArtifact(ctx context.Context, artifactID string, removeFiles bool) error
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
}

type pitrLocationClient interface {
	// FileStat returns file info. It returns error if file is empty or not exists.
	FileStat(ctx context.Context, endpoint, accessKey, secretKey, bucketName, name string) (minio.FileInfo, error)

	// List scans path with prefix and returns all files with given suffix.
	// Both prefix and suffix can be omitted.
	List(ctx context.Context, endpoint, accessKey, secretKey, bucketName, prefix, suffix string) ([]minio.FileInfo, error)
}

// pitrTimerangeService provides methods that help us inspect PITR artifacts
type pitrTimerangeService interface {
	// ListPITRTimeranges list the available PITR timeranges for the given artifact in the provided location
	ListPITRTimeranges(ctx context.Context, artifactName string, location *models.BackupLocation) ([]Timeline, error)
}
