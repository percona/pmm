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

// Package tlshelpers contains helpers for databases tls connections.
package tlshelpers

import (
	"crypto/tls"
	"crypto/x509"
	"errors"

	"github.com/gomodule/redigo/redis"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

// GetValkeyTlsConfig returns TLS config for Valkey connections.
func GetValkeyTlsConfig(files *agentv1.TextFiles) ([]redis.DialOption, error) {
	var opts []redis.DialOption
	if files != nil {
		ca := x509.NewCertPool()
		cert, err := tls.X509KeyPair([]byte(files.Files["tlsCert"]), []byte(files.Files["tlsKey"]))
		if err != nil {
			return nil, err
		}
		ok := ca.AppendCertsFromPEM([]byte(files.Files["tlsCa"]))
		if !ok {
			return nil, errors.New("failed to append certs from PEM")
		}
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			Certificates:       []tls.Certificate{cert},
			RootCAs:            ca,
		}
		opts = append(opts, redis.DialUseTLS(true))
		opts = append(opts, redis.DialTLSSkipVerify(true))
		opts = append(opts, redis.DialTLSConfig(tlsConfig))
	}
	return opts, nil
}
