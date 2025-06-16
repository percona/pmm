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

package backup

import (
	"context"
	"fmt"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/percona/pmm/managed/models"
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

func TestGetPITROplogs(t *testing.T) {
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

	mockedStorage := &MockStorage{}

	t.Run("successful", func(t *testing.T) {
		listedFiles := []minio.FileInfo{
			{
				Name: "rs0/20220829/20220829115611-1.20220829120544-10.oplog.s2",
				Size: 1024,
			},
		}

		// statFile := minio.FileInfo{
		//	 Name: pitrFSPrefix + "rs0/20220829/20220829115611-1.20220829120544-10.oplog.s2",
		//	 Size: 1024,
		// }
		mockedStorage.On("List", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(listedFiles, nil).Once()

		service := NewPBMPITRService()
		timelines, err := service.getPITROplogs(ctx, mockedStorage, location, &models.Artifact{})
		assert.NoError(t, err)
		assert.Len(t, timelines, 1)
	})

	t.Run("fails on file list error", func(t *testing.T) {
		mockedStorage.On("List", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("listing object error")).Once()

		service := NewPBMPITRService()
		timelines, err := service.getPITROplogs(ctx, mockedStorage, location, &models.Artifact{})
		assert.Error(t, err)
		assert.Nil(t, timelines)
	})

	t.Run("skips artifacts with deletion markers", func(t *testing.T) {
		listedFiles := []minio.FileInfo{
			{
				Name:           "rs0/20220829/20220829115611-1.20220829120544-10.oplog.s2",
				Size:           1024,
				IsDeleteMarker: true,
			},
		}

		mockedStorage.On("List", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(listedFiles, nil).Once()

		service := NewPBMPITRService()
		timelines, err := service.getPITROplogs(ctx, mockedStorage, location, &models.Artifact{})
		assert.NoError(t, err)
		assert.Empty(t, timelines)
	})

	mockedStorage.AssertExpectations(t)
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
				t.Fatalf("wrong timelines, exepct <%d> %v, got <%d> %v", len(test.expect), printTTL(test.expect...), len(got), printTTL(got...))
			}
			for i, gl := range got {
				if test.expect[i] != gl {
					t.Errorf("wrong timeline %d, exepct %v, got %v", i, printTTL(test.expect[i]), printTTL(gl))
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

func printTTL(tlns ...Timeline) string {
	ret := make([]string, 0, len(tlns))
	for _, t := range tlns {
		ret = append(ret, fmt.Sprintf("[%v - %v]", t.Start, t.End))
	}

	return strings.Join(ret, ", ")
}

func TestGetPITRFiles(t *testing.T) {
	ctx := context.Background()
	S3Config := models.S3LocationConfig{
		Endpoint:     "https://s3.us-west-2.amazonaws.com",
		AccessKey:    "access_key",
		SecretKey:    "secret_key",
		BucketName:   "example_bucket",
		BucketRegion: "us-east-1",
	}
	location := &models.BackupLocation{
		S3Config: &S3Config,
	}

	mockedStorage := &MockStorage{}
	service := NewPBMPITRService()

	listedFiles := []minio.FileInfo{
		{Name: "rs0/20230411/20230411112014-2.20230411112507-12.oplog.s2"},
		{Name: "rs0/20230411/20230411112507-12.20230411112514-3.oplog.s2"},
		{Name: "rs0/20230411/20230411112514-3.20230411113007-8.oplog.s2"},
		{Name: "rs0/20230411/20230411113007-8.20230411113014-2.oplog.s2"},
		{Name: "rs0/20230411/20230411113014-2.20230411113507-8.oplog.s2"},
	}

	t.Run("'until' not specified", func(t *testing.T) {
		mockedStorage.On("List", ctx, S3Config.Endpoint, S3Config.AccessKey, S3Config.SecretKey, S3Config.BucketName, pitrFSPrefix, "").Return(listedFiles, nil).Twice()

		PITRChunks, err := service.GetPITRFiles(ctx, mockedStorage, location, &models.Artifact{}, nil)
		require.NoError(t, err)

		expectedRes, err := service.getPITROplogs(ctx, mockedStorage, location, &models.Artifact{})
		require.NoError(t, err)

		assert.Equal(t, expectedRes, PITRChunks)
	})

	t.Run("'until' specified", func(t *testing.T) {
		mockedStorage.On("List", ctx, S3Config.Endpoint, S3Config.AccessKey, S3Config.SecretKey, S3Config.BucketName, pitrFSPrefix, "").Return(listedFiles, nil).Twice()

		until, err := time.Parse("2006-01-02T15:04:05", "2023-04-11T11:25:14")
		require.NoError(t, err)

		PITRChunks, err := service.GetPITRFiles(ctx, mockedStorage, location, &models.Artifact{}, &until)
		require.NoError(t, err)

		expectedRes, err := service.getPITROplogs(ctx, mockedStorage, location, &models.Artifact{})
		require.NoError(t, err)

		assert.Equal(t, expectedRes[:2], PITRChunks)
	})

	mockedStorage.AssertExpectations(t)
}
