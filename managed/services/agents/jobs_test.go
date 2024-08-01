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

package agents

import (
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"

	backuppb "github.com/percona/pmm/api/backup/v1"
	"github.com/percona/pmm/managed/models"
)

func TestArtifactMetadataFromProto(t *testing.T) {
	t.Run("all fields are filled", func(t *testing.T) {
		protoMetadata := backuppb.Metadata{
			FileList:           []*backuppb.File{{Name: "dir1", IsDirectory: true}, {Name: "file1"}, {Name: "file2"}},
			RestoreTo:          &timestamppb.Timestamp{Seconds: 123, Nanos: 456},
			BackupToolMetadata: &backuppb.Metadata_PbmMetadata{PbmMetadata: &backuppb.PbmMetadata{Name: "some name"}},
		}

		expected := &models.Metadata{
			FileList:       []models.File{{Name: "dir1", IsDirectory: true}, {Name: "file1"}, {Name: "file2"}},
			RestoreTo:      pointer.ToTime(time.Unix(123, 456).UTC()),
			BackupToolData: &models.BackupToolData{PbmMetadata: &models.PbmMetadata{Name: "some name"}},
		}

		actual := artifactMetadataFromProto(&protoMetadata)
		assert.Equal(t, expected, actual)
	})

	t.Run("some fields are empty", func(t *testing.T) {
		protoMetadata := backuppb.Metadata{
			FileList: []*backuppb.File{{Name: "dir1", IsDirectory: true}, {Name: "file1"}, {Name: "file2"}},
		}

		expected := &models.Metadata{
			FileList: []models.File{{Name: "dir1", IsDirectory: true}, {Name: "file1"}, {Name: "file2"}},
		}

		actual := artifactMetadataFromProto(&protoMetadata)
		assert.Equal(t, expected, actual)
	})

	t.Run("argument is nil", func(t *testing.T) {
		var protoMetadata *backuppb.Metadata
		var expected *models.Metadata

		actual := artifactMetadataFromProto(protoMetadata)
		assert.Equal(t, expected, actual)
	})
}
