// Copyright (C) 2023 Percona LLC
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

package alerting

import (
	"context"

	gapi "github.com/grafana/grafana-api-golang-client"

	"github.com/percona/pmm/managed/services"
)

type grafanaClient interface {
	CreateAlertRule(ctx context.Context, folderName, groupName string, rule *services.Rule) error
	GetDatasourceUIDByID(ctx context.Context, id int64) (string, error)
	GetFolderByUID(ctx context.Context, uid string) (*gapi.Folder, error)
}
