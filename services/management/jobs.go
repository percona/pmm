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

package management

import (
	"context"

	jobsAPI "github.com/percona/pmm/api/managementpb/jobs"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// JobsAPIService provides methods for Jobs starting and management.
type JobsAPIService struct {
	l *logrus.Entry

	db          *reform.DB
	jobsService jobsService
}

// NewJobsAPIServer creates new jobs service.
func NewJobsAPIServer(db *reform.DB, service jobsService) *JobsAPIService {
	return &JobsAPIService{
		l: logrus.WithField("component", "management/jobs"),

		db:          db,
		jobsService: service,
	}
}

// GetJob returns job result.
func (s *JobsAPIService) GetJob(_ context.Context, req *jobsAPI.GetJobRequest) (*jobsAPI.GetJobResponse, error) {
	result, err := models.FindJobResultByID(s.db.Querier, req.JobId)
	if err != nil {
		return nil, err
	}

	resp := &jobsAPI.GetJobResponse{
		JobId:      result.ID,
		PmmAgentId: result.PMMAgentID,
		Done:       result.Done,
	}

	if !result.Done {
		return resp, nil
	}

	if result.Error != "" {
		resp.Result = &jobsAPI.GetJobResponse_Error_{
			Error: &jobsAPI.GetJobResponse_Error{
				Message: result.Error,
			},
		}

		return resp, nil
	}

	switch result.Type {
	case models.Echo:
		resp.Result = &jobsAPI.GetJobResponse_Echo_{
			Echo: &jobsAPI.GetJobResponse_Echo{
				Message: result.Result.Echo.Message,
			},
		}
	default:
		return nil, errors.Errorf("Unexpected job type: %s", result.Type)
	}

	return resp, nil
}

// StartEchoJob starts echo job. Its purpose is testing.
func (s *JobsAPIService) StartEchoJob(_ context.Context, req *jobsAPI.StartEchoJobRequest) (*jobsAPI.StartEchoJobResponse, error) {
	res, err := s.prepareAgentJob(req.PmmAgentId, models.Echo)
	if err != nil {
		return nil, err
	}

	err = s.jobsService.StartEchoJob(res.ID, res.PMMAgentID, req.Timeout.AsDuration(), req.Message, req.Delay.AsDuration())
	if err != nil {
		s.saveJobError(res.ID, err.Error())
		return nil, err
	}

	return &jobsAPI.StartEchoJobResponse{
		PmmAgentId: req.PmmAgentId,
		JobId:      res.ID,
	}, nil
}

// CancelJob terminates job.
func (s *JobsAPIService) CancelJob(_ context.Context, req *jobsAPI.CancelJobRequest) (*jobsAPI.CancelJobResponse, error) {
	if err := s.jobsService.StopJob(req.JobId); err != nil {
		return nil, err
	}

	return &jobsAPI.CancelJobResponse{}, nil
}

func (s *JobsAPIService) saveJobError(resultID string, message string) {
	if e := s.db.InTransaction(func(t *reform.TX) error {
		res, err := models.FindJobResultByID(t.Querier, resultID)
		if err != nil {
			return err
		}

		res.Error = message
		res.Done = true
		return t.Update(res)
	}); e != nil {
		s.l.Errorf("Failed to save job result: %+v", e)
	}
}

func (s *JobsAPIService) prepareAgentJob(pmmAgentID string, jobType models.JobType) (*models.JobResult, error) {
	var res *models.JobResult
	e := s.db.InTransaction(func(tx *reform.TX) error {
		_, err := models.FindAgentByID(tx.Querier, pmmAgentID)
		if err != nil {
			return err
		}

		res, err = models.CreateJobResult(tx.Querier, pmmAgentID, jobType)
		return err
	})
	if e != nil {
		return nil, e
	}
	return res, nil
}

func (s *JobsAPIService) prepareServiceJob(serviceID, pmmAgentID, database string, jobType models.JobType) (*models.JobResult, string, error) {
	var res *models.JobResult
	var dsn string
	e := s.db.InTransaction(func(tx *reform.TX) error {
		agents, err := models.FindPMMAgentsForService(tx.Querier, serviceID)
		if err != nil {
			return err
		}

		if pmmAgentID, err = models.FindPmmAgentIDToRunActionOrJob(pmmAgentID, agents); err != nil {
			return err
		}

		if dsn, _, err = models.FindDSNByServiceIDandPMMAgentID(tx.Querier, serviceID, pmmAgentID, database); err != nil {
			return err
		}

		res, err = models.CreateJobResult(tx.Querier, pmmAgentID, jobType)
		return err
	})
	if e != nil {
		return nil, "", e
	}
	return res, dsn, nil
}
