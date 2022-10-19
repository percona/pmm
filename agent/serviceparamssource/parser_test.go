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

package serviceparamssource

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
)

func TestDefaultsFileParser(t *testing.T) {
	t.Parallel()
	cnfFilePath, err := filepath.Abs("../utils/tests/testdata/serviceparamssource/.my.cnf")
	assert.NoError(t, err)
	jsonFilePath, err := filepath.Abs("../utils/tests/testdata/serviceparamssource/parameters.json")
	assert.NoError(t, err)

	testCases := []struct {
		name        string
		req         *agentpb.ParseServiceParamsSourceRequest
		expectedErr string
	}{
		{
			name: "Test json parser",
			req: &agentpb.ParseServiceParamsSourceRequest{
				ServiceType: inventorypb.ServiceType_HAPROXY_SERVICE,
				FilePath:    jsonFilePath,
			},
		},
		{
			name: "Valid MySQL file",
			req: &agentpb.ParseServiceParamsSourceRequest{
				ServiceType: inventorypb.ServiceType_MYSQL_SERVICE,
				FilePath:    cnfFilePath,
			},
		},
		{
			name: "File not found",
			req: &agentpb.ParseServiceParamsSourceRequest{
				ServiceType: inventorypb.ServiceType_MYSQL_SERVICE,
				FilePath:    "path/to/invalid/file.cnf",
			},
			expectedErr: `file doesn't exist`,
		},
		{
			name: "Unrecognized file type (haproxy not implemented yet)",
			req: &agentpb.ParseServiceParamsSourceRequest{
				ServiceType: inventorypb.ServiceType_HAPROXY_SERVICE,
				FilePath:    cnfFilePath,
			},
			expectedErr: `unrecognized file type`,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			c := New()
			resp := c.ParseServiceParamsSource(testCase.req)
			require.NotNil(t, resp)
			if testCase.expectedErr == "" {
				assert.Empty(t, resp.Error)
			} else {
				require.NotEmpty(t, resp.Error)
				assert.Regexp(t, `.*`+testCase.expectedErr+`.*`, resp.Error)
			}
		})
	}
}

func TestValidateResults(t *testing.T) {
	t.Parallel()
	t.Run("validation error", func(t *testing.T) {
		t.Parallel()
		err := validateResults(&serviceParamsSource{
			"",
			"",
			"",
			"",
			0,
			"",
		})

		require.Error(t, err)
	})

	t.Run("validation ok - user and password", func(t *testing.T) {
		t.Parallel()
		err := validateResults(&serviceParamsSource{
			"root",
			"root123",
			"",
			"",
			0,
			"",
		})

		require.NoError(t, err)
	})

	t.Run("validation ok - only port", func(t *testing.T) {
		t.Parallel()
		err := validateResults(&serviceParamsSource{
			"",
			"",
			"",
			"",
			3133,
			"",
		})

		require.NoError(t, err)
	})
}
