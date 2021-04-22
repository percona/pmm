// pmm-managed
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
	"time"

	"github.com/percona/pmm-managed/models"
)

//go:generate mockery -name=jobsService -case=snake -inpkg -testonly
//go:generate mockery -name=awsS3 -case=snake -inpkg -testonly

// jobsService is a subset of methods of agents.JobsService used by this package.
// We use it instead of real type for testing and to avoid dependency cycle.
type jobsService interface {
	StopJob(jobID string) error
	StartMySQLBackupJob(id, pmmAgentID string, timeout time.Duration, name string, dbConfig models.DBConfig, locationConfig models.BackupLocationConfig) error
}

type awsS3 interface {
	GetBucketLocation(host string, accessKey, secretKey, name string) (string, error)
	BucketExists(host string, accessKey, secretKey, name string) (bool, error)
}
