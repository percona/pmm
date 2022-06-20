// pmm-agent
// Copyright 2019 Percona LLC
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

package mongo_fix

import (
	"net/url"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ClientOptionsForDSN applies URI to Client.
func ClientOptionsForDSN(dsn string) (*options.ClientOptions, error) {
	clientOptions := options.Client().ApplyURI(dsn)

	// Workaround for PMM-9320
	// if username or password is set, need to replace it with correctly parsed credentials.
	parsedDsn, err := url.Parse(dsn)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse DSN")
	}
	username := parsedDsn.User.Username()
	password, _ := parsedDsn.User.Password()
	if username != "" || password != "" {
		clientOptions = clientOptions.SetAuth(options.Credential{Username: username, Password: password})
	}

	return clientOptions, nil
}
