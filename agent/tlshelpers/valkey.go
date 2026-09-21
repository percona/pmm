// Copyright (C) 2023 Percona LLC
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

package tlshelpers

import (
	"crypto/tls"
	"crypto/x509"
	"errors"

	"github.com/gomodule/redigo/redis"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

// GetValkeyTLSConfig returns TLS config for Valkey connections.
func GetValkeyTLSConfig(files *agentv1.TextFiles, useTLS, tlsSkipVerify bool) ([]redis.DialOption, error) {
	if isEmptyTLSFiles(files) {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: tlsSkipVerify, //nolint:gosec
	}

	// Server-auth-only and pinned-certificate setups are both valid, so each piece of the
	// material is applied only when it is actually present rather than assumed complete.
	if certPEM, keyPEM := files.Files["tlsCert"], files.Files["tlsKey"]; certPEM != "" && keyPEM != "" {
		cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
		if err != nil {
			return nil, err
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if caPEM := files.Files["tlsCa"]; caPEM != "" {
		ca := x509.NewCertPool()
		if !ca.AppendCertsFromPEM([]byte(caPEM)) {
			return nil, errors.New("failed to append certs from PEM")
		}

		tlsConfig.RootCAs = ca
	}

	opts := []redis.DialOption{
		redis.DialUseTLS(useTLS),
		redis.DialTLSSkipVerify(tlsSkipVerify),
		redis.DialTLSConfig(tlsConfig),
	}

	return opts, nil
}

// isEmptyTLSFiles checks if the TLS files are empty.
func isEmptyTLSFiles(files *agentv1.TextFiles) bool {
	if files == nil || len(files.Files) == 0 {
		return true
	}
	if files.Files["tlsCert"] == "" && files.Files["tlsKey"] == "" && files.Files["tlsCa"] == "" {
		return true
	}
	return false
}
