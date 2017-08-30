// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package handlers

import (
	"golang.org/x/net/context"

	"github.com/percona/pmm-managed/api"
	"github.com/percona/pmm-managed/services/prometheus"
)

type ScrapeJobsServer struct {
	Prometheus *prometheus.Service
}

// List returns all scrape jobs.
func (s *ScrapeJobsServer) List(ctx context.Context, req *api.ScrapeJobsListRequest) (*api.ScrapeJobsListResponse, error) {
	jobs, err := s.Prometheus.ListScrapeJobs(ctx)
	if err != nil {
		return nil, err
	}
	res := &api.ScrapeJobsListResponse{
		ScrapeJobs: make([]*api.ScrapeJob, len(jobs)),
	}
	for i, job := range jobs {
		j := api.ScrapeJob(job)
		res.ScrapeJobs[i] = &j
	}
	return res, nil
}

// Get returns a scrape job by name.
func (s *ScrapeJobsServer) Get(ctx context.Context, req *api.ScrapeJobsGetRequest) (*api.ScrapeJobsGetResponse, error) {
	job, err := s.Prometheus.GetScrapeJob(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	j := api.ScrapeJob(*job)
	return &api.ScrapeJobsGetResponse{
		ScrapeJob: &j,
	}, nil
}

// Create creates a new scrape job.
func (s *ScrapeJobsServer) Create(ctx context.Context, req *api.ScrapeJobsCreateRequest) (*api.ScrapeJobsCreateResponse, error) {
	j := prometheus.ScrapeJob(*req.ScrapeJob)
	if err := s.Prometheus.CreateScrapeJob(ctx, &j); err != nil {
		return nil, err
	}
	return &api.ScrapeJobsCreateResponse{}, nil
}

// Delete removes a scrape job by name.
func (s *ScrapeJobsServer) Delete(ctx context.Context, req *api.ScrapeJobsDeleteRequest) (*api.ScrapeJobsDeleteResponse, error) {
	if err := s.Prometheus.DeleteScrapeJob(ctx, req.Name); err != nil {
		return nil, err
	}
	return &api.ScrapeJobsDeleteResponse{}, nil
}

// check interface
var _ api.ScrapeJobsServer = (*ScrapeJobsServer)(nil)
