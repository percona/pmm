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

package supervisor

import (
	"context"

	"github.com/percona/kardianos-service"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm-managed/utils/logger"
)

// Supervisor starts and stops external processes (typically Agents) using system process supervisor
// (systemd or supervisord).
// It does not tracks them itself.
type Supervisor struct {
}

func New(l *logrus.Entry) *Supervisor {
	l.WithField("component", "supervisor").Infof("Using %s", service.Platform())
	return &Supervisor{}
}

func (s *Supervisor) Start(ctx context.Context, config *service.Config) error {
	config.Option = adjustOption(config.Option)

	svc, err := service.New(new(program), config)
	if err != nil {
		return err
	}

	logger.Get(ctx).WithField("component", "supervisor").Infof("Installing %s", config.Name)
	if err := svc.Install(); err != nil {
		return errors.Wrapf(err, "failed to install %s", config.Name)
	}

	logger.Get(ctx).WithField("component", "supervisor").Infof("Starting %s", config.Name)
	return errors.Wrapf(svc.Start(), "failed to start %s", config.Name)
}

func (s *Supervisor) Stop(ctx context.Context, name string) error {
	config := &service.Config{Name: name}
	config.Option = adjustOption(config.Option)

	svc, err := service.New(new(program), config)
	if err != nil {
		return err
	}

	logger.Get(ctx).WithField("component", "supervisor").Infof("Stopping %s", config.Name)
	if err := svc.Stop(); err != nil {
		logger.Get(ctx).WithField("component", "supervisor").Errorf("Failed to stop %s: %s", config.Name, err)
	}

	logger.Get(ctx).WithField("component", "supervisor").Infof("Uninstalling %s", config.Name)
	return errors.Wrapf(svc.Uninstall(), "failed to uninstall %s", config.Name)
}
