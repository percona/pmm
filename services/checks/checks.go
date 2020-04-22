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

// Package checks provides security checks functionality.
package checks

import (
	"bytes"
	"context"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	api "github.com/percona-platform/saas/gen/checked"
	"github.com/percona-platform/saas/pkg/check"
	"github.com/percona/pmm/utils/tlsconfig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	defaultHost     = "check.percona.com"
	defaultInterval = 24 * time.Hour

	// Environment variables that affect checks service; only for testing.
	envHost      = "PERCONA_TEST_CHECKS_HOST"
	envPublicKey = "PERCONA_TEST_CHECKS_PUBLIC_KEY"
	envInterval  = "PERCONA_TEST_CHECKS_INTERVAL"
	envCheckFile = "PERCONA_TEST_CHECKS_FILE"

	timeout = 5 * time.Second
)

var defaultPublicKeys = []string{
	"RWSKCHyoLDYxJ1k0qeayKu3/fsXVS1z8M+0deAClryiHWP99Sr4R/gPP", // PMM 2.6
}

// Service is responsible for interactions with Percona Check service.
type Service struct {
	l          *logrus.Entry
	pmmVersion string
	host       string
	publicKeys []string
	interval   time.Duration

	m      sync.RWMutex
	checks []check.Check
}

// New returns Service with given PMM version.
func New(pmmVersion string) *Service {
	l := logrus.WithField("component", "check")
	s := &Service{
		l:          l,
		pmmVersion: pmmVersion,
		host:       defaultHost,
		publicKeys: defaultPublicKeys,
		interval:   defaultInterval,
	}

	if h := os.Getenv(envHost); h != "" {
		l.Warnf("Host changed to %s.", h)
		s.host = h
	}
	if k := os.Getenv(envPublicKey); k != "" {
		s.publicKeys = strings.Split(k, ",")
		l.Warnf("Public keys changed to %q.", k)
	}
	if d, err := time.ParseDuration(os.Getenv(envInterval)); err == nil && d > 0 {
		l.Warnf("Interval changed to %s.", d)
		s.interval = d
	}

	return s
}

// Run runs checks service that grabs checks from Percona Checks service every interval until context is canceled.
func (s *Service) Run(ctx context.Context) {
	if f := os.Getenv(envCheckFile); f != "" {
		s.l.Warnf("Using local test checks file: %s", f)

		data, err := ioutil.ReadFile(f) //nolint:gosec
		if err != nil {
			s.l.Errorf("Failed to read test checks file: %s.", err)
			return
		}
		checks, err := check.Parse(bytes.NewReader(data))
		if err != nil {
			s.l.Errorf("Failed to parse test checks file: %s.", err)
			return
		}

		s.updateChecks(checks)
		return
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		if err := s.downloadChecks(ctx); err != nil {
			s.l.Errorf("Failed to download checks: %v", err)
		}

		select {
		case <-ticker.C:
			// continue with next loop iteration
		case <-ctx.Done():
			return
		}
	}
}

// Checks returns available checks.
func (s *Service) Checks() []check.Check {
	s.m.RLock()
	defer s.m.RUnlock()

	r := make([]check.Check, 0, len(s.checks))
	return append(r, s.checks...)
}

func (s *Service) downloadChecks(ctx context.Context) error {
	s.l.Infof("Downloading checks from %s ...", s.host)

	host, _, err := net.SplitHostPort(s.host)
	if err != nil {
		return errors.Wrap(err, "failed to set checks host")
	}
	tlsConfig := tlsconfig.Get()
	tlsConfig.ServerName = host

	opts := []grpc.DialOption{
		// replacement is marked as experimental
		grpc.WithBackoffMaxDelay(timeout), //nolint:staticcheck
		grpc.WithBlock(),
		grpc.WithUserAgent("pmm-managed/" + s.pmmVersion),
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cc, err := grpc.DialContext(ctx, s.host, opts...)
	if err != nil {
		return errors.Wrap(err, "failed to dial")
	}
	defer cc.Close() //nolint:errcheck

	resp, err := api.NewCheckedAPIClient(cc).GetAllChecks(ctx, &api.GetAllChecksRequest{})
	if err != nil {
		return errors.Wrap(err, "failed to request checks service")
	}

	if err = s.verifySignatures(resp); err != nil {
		return err
	}

	checks, err := check.Parse(strings.NewReader(resp.File))
	if err != nil {
		return err
	}

	s.updateChecks(checks)
	return nil
}

func (s *Service) verifySignatures(resp *api.GetAllChecksResponse) error {
	if len(resp.Signatures) == 0 {
		return errors.New("zero signatures received")
	}

	var err error
	for _, sign := range resp.Signatures {
		for _, key := range s.publicKeys {
			if err = check.Verify([]byte(resp.File), key, sign); err == nil {
				return nil
			}
			s.l.Debugf("Key %q doesn't match signature %q: %s.", key, sign, err)
		}
	}

	return errors.New("no verified signatures")
}

func (s *Service) updateChecks(checks []check.Check) {
	s.m.Lock()
	defer s.m.Unlock()

	s.checks = checks
}
