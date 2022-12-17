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

package agents

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/logger"
)

// DefaultsFileParser requests from agent to parse defaultsFile.
type DefaultsFileParser struct {
	r *Registry
}

// NewDefaultsFileParser creates new ParseDefaultsFile request.
func NewDefaultsFileParser(r *Registry) *DefaultsFileParser {
	return &DefaultsFileParser{
		r: r,
	}
}

// ParseDefaultsFile sends request (with file path) to pmm-agent to parse defaults file.
func (p *DefaultsFileParser) ParseDefaultsFile(ctx context.Context, pmmAgentID, filePath string, serviceType models.ServiceType) (*models.ParseDefaultsFileResult, error) { //nolint:lll
	l := logger.Get(ctx)

	pmmAgent, err := p.r.get(pmmAgentID)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	defer func() {
		if dur := time.Since(start); dur > 5*time.Second {
			l.Warnf("ParseDefaultsFile took %s.", dur)
		}
	}()

	request, err := createRequest(filePath, serviceType)
	if err != nil {
		l.Debugf("can't create ParseDefaultsFileRequest %s", err)
		return nil, err
	}

	resp, err := pmmAgent.channel.SendAndWaitResponse(request)
	if err != nil {
		return nil, err
	}

	l.Infof("ParseDefaultsFile response from agent: %+v.", resp)
	parserResponse, ok := resp.(*agentpb.ParseDefaultsFileResponse)
	if !ok {
		return nil, errors.New("wrong response from agent (not ParseDefaultsFileResponse model)")
	}
	if parserResponse.Error != "" {
		return nil, errors.New(parserResponse.Error)
	}

	return &models.ParseDefaultsFileResult{
		Username: parserResponse.Username,
		Password: parserResponse.Password,
		Host:     parserResponse.Host,
		Port:     parserResponse.Port,
		Socket:   parserResponse.Socket,
	}, nil
}

func createRequest(configPath string, serviceType models.ServiceType) (*agentpb.ParseDefaultsFileRequest, error) {
	if serviceType == models.MySQLServiceType {
		request := &agentpb.ParseDefaultsFileRequest{
			ServiceType: inventorypb.ServiceType_MYSQL_SERVICE,
			ConfigPath:  configPath,
		}
		return request, nil
	} else {
		return nil, errors.Errorf("unhandled service type %s", serviceType)
	}
}
