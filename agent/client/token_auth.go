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

package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc/credentials"
)

type tokenAuth struct {
	token string
}

// GetRequestMetadata implements credentials.PerRPCCredentials interface.
func (t *tokenAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) { //nolint:revive
	return map[string]string{
		"Authorization": "Bearer " + t.token,
	}, nil
}

// RequireTransportSecurity implements credentials.PerRPCCredentials interface.
func (*tokenAuth) RequireTransportSecurity() bool {
	return false
}

// check interfaces.
var (
	_ credentials.PerRPCCredentials = (*tokenAuth)(nil)
)
