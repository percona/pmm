// Copyright (C) 2022 Percona LLC
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

package backup

import (
	"context"
	"fmt"
	"github.com/percona/pmm/managed/models"
	"path"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/percona/pmm/managed/services/minio"
)

func TestPitrMetaFromFileName(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected *oplogChunk
	}{
		{
			name:     "correctly formatted file name",
			filename: "rs0/20220829/20220829115611-1.20220829120544-10.oplog.s2",
			expected: &oplogChunk{
				RS:          "rs0",
				FName:       "test_artifact_name/pbmPitr/rs0/20220829/20220829115611-1.20220829120544-10.oplog.s2",
				Compression: compressionTypeS2,
				StartTS:     primitive.Timestamp{T: uint32(1661774171), I: 1},
				EndTS:       primitive.Timestamp{T: uint32(1661774744), I: 10},
			},
		},
		{
			name:     "incomplete file name",
			filename: "20220829115611-1.20220829120544-10.oplog.s2",
			expected: nil,
		},
		{
			name:     "without end timestamp",
			filename: "rs0/20220829/20220829115611-1.oplog.s2",
			expected: nil,
		},
		{
			name:     "without specified compression",
			filename: "rs0/20220829/20220829115611-1.20220829120544-10.oplog",
			expected: &oplogChunk{
				RS:          "rs0",
				FName:       "test_artifact_name/pbmPitr/rs0/20220829/20220829115611-1.20220829120544-10.oplog",
				Compression: compressionTypeNone,
				StartTS:     primitive.Timestamp{T: uint32(1661774171), I: 1},
				EndTS:       primitive.Timestamp{T: uint32(1661774744), I: 10},
			},
		},
	}

	prefix := path.Join("test_artifact_name", pitrFSPrefix)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := pitrMetaFromFileName(prefix, tt.filename)
			assert.Equal(t, tt.expected, chunk)
		})
	}
}

func TestPitrParseTs(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected *primitive.Timestamp
	}{
		{
			name:     "with time and index",
			filename: "20220829115611-10",
			expected: &primitive.Timestamp{T: uint32(1661774171), I: 10},
		},
		{
			name:     "time without index",
			filename: "20220829120544",
			expected: &primitive.Timestamp{T: uint32(1661774744), I: 0},
		},
		{
			name:     "with invalid timestamp",
			filename: "2022",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := pitrParseTS(tt.filename)
			assert.Equal(t, tt.expected, ts)
		})
	}
}

func TestListPITRTimelines(t *testing.T) {
	ctx := context.Background()
	location := &models.BackupLocation{
		S3Config: &models.S3LocationConfig{
			Endpoint:     "https://s3.us-west-2.amazonaws.com",
			AccessKey:    "access_key",
			SecretKey:    "secret_key",
			BucketName:   "example_bucket",
			BucketRegion: "us-east-1",
		},
	}

	t.Run("successful", func(t *testing.T) {
		mockedStorage := &mockBackupStorage{}
		listedFiles := []minio.FileInfo{
			{
				Name: "rs0/20220829/20220829115611-1.20220829120544-10.oplog.s2",
				Size: 1024,
			},
		}

		statFile := minio.FileInfo{
			Name: pitrFSPrefix + "rs0/20220829/20220829115611-1.20220829120544-10.oplog.s2",
			Size: 1024,
		}
		mockedStorage.On("List", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(listedFiles, nil)
		mockedStorage.On("FileStat", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(statFile, nil)

		ss := NewPITRStorageService(mockedStorage)
		timelines, err := ss.getPITROplogs(ctx, location, "")
		assert.NoError(t, err)
		assert.Len(t, timelines, 1)
	})

	t.Run("fails on file list error", func(t *testing.T) {
		mockedStorage := &mockBackupStorage{}
		mockedStorage.On("List", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("listing object error"))

		ss := NewPITRStorageService(mockedStorage)
		timelines, err := ss.getPITROplogs(ctx, location, "")
		assert.Error(t, err)
		assert.Nil(t, timelines)
	})

	t.Run("skips artifacts with file stat errors", func(t *testing.T) {
		mockedStorage := &mockBackupStorage{}
		listedFiles := []minio.FileInfo{
			{
				Name: "rs0/20220829/20220829115611-1.20220829120544-10.oplog.s2",
				Size: 1024,
			},
		}

		mockedStorage.On("List", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(listedFiles, nil)
		mockedStorage.On("FileStat", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(minio.FileInfo{}, errors.New("file stat error"))

		ss := NewPITRStorageService(mockedStorage)
		timelines, err := ss.getPITROplogs(ctx, location, "")
		assert.NoError(t, err)
		assert.Len(t, timelines, 0)
	})
}

func TestPITRMergeTimelines(t *testing.T) {
	tests := []struct {
		name   string
		tl     [][]Timeline
		expect []Timeline
	}{
		{
			name:   "nothing",
			tl:     [][]Timeline{},
			expect: []Timeline{},
		},
		{
			name: "empy set",
			tl: [][]Timeline{
				{},
			},
			expect: []Timeline{},
		},
		{
			name: "no match",
			tl: [][]Timeline{
				{
					{Start: 3, End: 6},
					{Start: 14, End: 19},
					{Start: 20, End: 42},
				},
				{
					{Start: 1, End: 3},
					{Start: 6, End: 14},
					{Start: 50, End: 55},
				},
				{
					{Start: 7, End: 10},
					{Start: 12, End: 19},
					{Start: 20, End: 26},
					{Start: 27, End: 60},
				},
			},
			expect: []Timeline{},
		},
		{
			name: "no match2",
			tl: [][]Timeline{
				{
					{Start: 1, End: 5},
					{Start: 8, End: 13},
				},
				{
					{Start: 6, End: 7},
				},
			},
			expect: []Timeline{},
		},
		{
			name: "no match3",
			tl: [][]Timeline{
				{
					{Start: 1, End: 5},
					{Start: 8, End: 13},
				},
				{
					{Start: 5, End: 8},
				},
			},
			expect: []Timeline{},
		},
		{
			name: "some empty",
			tl: [][]Timeline{
				{},
				{
					{Start: 4, End: 8},
				},
			},
			expect: []Timeline{{Start: 4, End: 8}},
		},
		{
			name: "no gaps",
			tl: [][]Timeline{
				{
					{Start: 1, End: 5},
				},
				{
					{Start: 4, End: 8},
				},
			},
			expect: []Timeline{{Start: 4, End: 5}},
		},
		{
			name: "no gaps2",
			tl: [][]Timeline{
				{
					{Start: 4, End: 8},
				},
				{
					{Start: 1, End: 5},
				},
			},
			expect: []Timeline{{Start: 4, End: 5}},
		},
		{
			name: "no gaps3",
			tl: [][]Timeline{
				{
					{Start: 1, End: 8},
				},
				{
					{Start: 1, End: 5},
				},
			},
			expect: []Timeline{{Start: 1, End: 5}},
		},
		{
			name: "overlaps",
			tl: [][]Timeline{
				{
					{Start: 2, End: 6},
					{Start: 8, End: 12},
					{Start: 13, End: 15},
				},
				{
					{Start: 1, End: 4},
					{Start: 9, End: 14},
				},
				{
					{Start: 3, End: 7},
					{Start: 8, End: 11},
					{Start: 12, End: 14},
				},
				{
					{Start: 2, End: 9},
					{Start: 10, End: 17},
				},
				{
					{Start: 1, End: 5},
					{Start: 6, End: 14},
					{Start: 15, End: 19},
				},
			},
			expect: []Timeline{
				{Start: 3, End: 4},
				{Start: 10, End: 11},
				{Start: 13, End: 14},
			},
		},
		{
			name: "all match",
			tl: [][]Timeline{
				{
					{Start: 3, End: 6},
					{Start: 14, End: 19},
					{Start: 19, End: 42},
				},
				{
					{Start: 3, End: 6},
					{Start: 14, End: 19},
					{Start: 19, End: 42},
				},
				{
					{Start: 3, End: 6},
					{Start: 14, End: 19},
					{Start: 19, End: 42},
				},
				{
					{Start: 3, End: 6},
					{Start: 14, End: 19},
					{Start: 19, End: 42},
				},
			},
			expect: []Timeline{
				{Start: 3, End: 6},
				{Start: 14, End: 19},
				{Start: 19, End: 42},
			},
		},
		{
			name: "partly overlap",
			tl: [][]Timeline{
				{
					{Start: 3, End: 8},
					{Start: 14, End: 19},
					{Start: 21, End: 42},
				},
				{
					{Start: 1, End: 3},
					{Start: 4, End: 7},
					{Start: 19, End: 36},
				},
				{
					{Start: 5, End: 8},
					{Start: 14, End: 19},
					{Start: 20, End: 42},
				},
			},
			expect: []Timeline{
				{Start: 5, End: 7},
				{Start: 21, End: 36},
			},
		},
		{
			name: "partly overlap2",
			tl: [][]Timeline{
				{
					{Start: 1, End: 4},
					{Start: 7, End: 11},
					{Start: 16, End: 20},
				},
				{
					{Start: 3, End: 12},
					{Start: 15, End: 17},
				},
				{
					{Start: 1, End: 12},
					{Start: 16, End: 18},
				},
			},
			expect: []Timeline{
				{Start: 3, End: 4},
				{Start: 7, End: 11},
				{Start: 16, End: 17},
			},
		},
		{
			name: "redundant chunks",
			tl: [][]Timeline{
				{
					{Start: 3, End: 6},
					{Start: 14, End: 19},
					{Start: 19, End: 40},
					{Start: 42, End: 100500},
				},
				{
					{Start: 2, End: 7},
					{Start: 7, End: 8},
					{Start: 8, End: 10},
					{Start: 14, End: 20},
					{Start: 20, End: 30},
				},
				{
					{Start: 1, End: 5},
					{Start: 13, End: 19},
					{Start: 20, End: 30},
				},
			},
			expect: []Timeline{
				{Start: 3, End: 5},
				{Start: 14, End: 19},
				{Start: 20, End: 30},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := mergeTimelines(test.tl...)
			if len(test.expect) != len(got) {
				t.Fatalf("wrong timelines, exepct <%d> %v, got <%d> %v", len(test.expect), printttl(test.expect...), len(got), printttl(got...))
			}
			for i, gl := range got {
				if test.expect[i] != gl {
					t.Errorf("wrong timeline %d, exepct %v, got %v", i, printttl(test.expect[i]), printttl(gl))
				}
			}
		})
	}
}

func BenchmarkMergeTimelines(b *testing.B) {
	tl := [][]Timeline{
		{
			{Start: 3, End: 8},
			{Start: 14, End: 19},
			{Start: 21, End: 42},
		},
		{
			{Start: 1, End: 3},
			{Start: 4, End: 7},
			{Start: 19, End: 36},
		},
		{
			{Start: 5, End: 8},
			{Start: 14, End: 19},
			{Start: 20, End: 42},
		},
		{
			{Start: 3, End: 6},
			{Start: 14, End: 19},
			{Start: 19, End: 40},
			{Start: 42, End: 100500},
		},
		{
			{Start: 2, End: 7},
			{Start: 7, End: 8},
			{Start: 8, End: 10},
			{Start: 14, End: 20},
			{Start: 20, End: 30},
			{Start: 31, End: 40},
			{Start: 41, End: 50},
			{Start: 51, End: 60},
		},
		{
			{Start: 1, End: 5},
			{Start: 13, End: 19},
			{Start: 20, End: 30},
		},
	}
	for i := 0; i < b.N; i++ {
		mergeTimelines(tl...)
	}
}

func printttl(tlns ...Timeline) string {
	var ret []string
	for _, t := range tlns {
		ret = append(ret, fmt.Sprintf("[%v - %v]", t.Start, t.End))
	}

	return strings.Join(ret, ", ")
}
