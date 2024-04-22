// Copyright (C) 2024 Percona LLC
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

package server

import (
	"crypto/subtle"
	"sync"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/serverpb"
	"github.com/percona/pmm/managed/models"
)

// AWSInstanceChecker checks AWS EC2 instance ID for AMI.
type AWSInstanceChecker struct {
	db               *reform.DB
	telemetryService telemetryService
	l                *logrus.Entry

	rw      sync.RWMutex
	checked bool
}

// NewAWSInstanceChecker creates a new AWSInstanceChecker.
func NewAWSInstanceChecker(db *reform.DB, telemetryService telemetryService) *AWSInstanceChecker {
	return &AWSInstanceChecker{
		db:               db,
		telemetryService: telemetryService,
		l:                logrus.WithField("component", "server/awsInstanceChecker"),
	}
}

// MustCheck returns true if instance ID must be checked: this is AMI, and it wasn't checked already.
func (c *AWSInstanceChecker) MustCheck() bool {
	// fast-path without hitting database
	c.rw.RLock()
	checked := c.checked
	c.rw.RUnlock()
	if checked {
		return false
	}

	c.rw.Lock()
	defer c.rw.Unlock()

	if c.telemetryService.DistributionMethod() != serverpb.DistributionMethod_AMI {
		c.checked = true
		return false
	}

	settings, err := models.GetSettings(c.db.Querier)
	if err != nil {
		c.l.Error(err)
		return true
	}
	if settings.AWSInstanceChecked {
		c.checked = true
		return false
	}

	return true
}

// check performs instance ID check and stores successful result flag in settings.
func (c *AWSInstanceChecker) check(instanceID string) error {
	// do not allow more AWS API calls if instance is already checked
	if !c.MustCheck() {
		return nil
	}

	sess, err := session.NewSession()
	if err != nil {
		return errors.Wrap(err, "cannot create AWS session")
	}
	doc, err := ec2metadata.New(sess).GetInstanceIdentityDocument()
	if err != nil {
		c.l.Error(err)
		return status.Error(codes.Unavailable, "cannot get instance metadata")
	}
	if subtle.ConstantTimeCompare([]byte(instanceID), []byte(doc.InstanceID)) == 0 {
		return status.Error(codes.InvalidArgument, "invalid instance ID")
	}

	if e := c.db.InTransaction(func(tx *reform.TX) error {
		settings, err := models.GetSettings(tx.Querier)
		if err != nil {
			return err
		}

		settings.AWSInstanceChecked = true
		return models.SaveSettings(tx.Querier, settings)
	}); e != nil {
		return e
	}

	c.rw.Lock()
	c.checked = true
	c.rw.Unlock()

	return nil
}
