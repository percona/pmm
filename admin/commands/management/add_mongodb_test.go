// Copyright (C) 2024 Percona LLC
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

package management

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	mongodb "github.com/percona/pmm/api/managementpb/json/client/mongo_db"
)

func TestAddMongoDB(t *testing.T) {
	t.Run("TablestatEnabled", func(t *testing.T) {
		res := &addMongoDBResult{
			Service: &mongodb.AddMongoDBOKBodyService{
				ServiceID:   "/service_id/1",
				ServiceName: "mysql-1",
			},
		}
		expected := strings.TrimSpace(`
MongoDB Service added.
Service ID  : /service_id/1
Service name: mysql-1
		`)
		assert.Equal(t, expected, strings.TrimSpace(res.String()))
	})
}
