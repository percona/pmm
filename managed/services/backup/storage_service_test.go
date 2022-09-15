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
				FName:       "pbmPitr/rs0/20220829/20220829115611-1.20220829120544-10.oplog.s2",
				Compression: CompressionTypeS2,
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
				FName:       "pbmPitr/rs0/20220829/20220829115611-1.20220829120544-10.oplog",
				Compression: CompressionTypeNone,
				StartTS:     primitive.Timestamp{T: uint32(1661774171), I: 1},
				EndTS:       primitive.Timestamp{T: uint32(1661774744), I: 10},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := pitrMetaFromFileName(tt.filename)
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

	t.Run("fails for empty storage location", func(t *testing.T) {
	})

	t.Run("successful", func(t *testing.T) {
		mockedStorage := &mockStoragePath{}
		listedFiles := []minio.FileInfo{
			{
				Name: "rs0/20220829/20220829115611-1.20220829120544-10.oplog.s2",
				Size: 1024,
			},
		}

		statFile := minio.FileInfo{
			Name: PITRfsPrefix + "rs0/20220829/20220829115611-1.20220829120544-10.oplog.s2",
			Size: 1024,
		}
		mockedStorage.On("List", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(listedFiles, nil)
		mockedStorage.On("FileStat", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(statFile, nil)

		ss := NewStorageService()
		ss.storage = mockedStorage
		timelines, err := ss.getPITRTimeRanges(ctx)
		assert.NoError(t, err)
		assert.Len(t, timelines, 1)
	})

	t.Run("fails on file list error", func(t *testing.T) {
		mockedStorage := &mockStoragePath{}
		mockedStorage.On("List", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("listing object error"))

		ss := NewStorageService()
		ss.storage = mockedStorage
		timelines, err := ss.getPITRTimeRanges(ctx)
		assert.NoError(t, err)
		assert.Nil(t, timelines)
	})

	t.Run("skips artifacts with file stat errors", func(t *testing.T) {
		mockedStorage := &mockStoragePath{}
		listedFiles := []minio.FileInfo{
			{
				Name: "rs0/20220829/20220829115611-1.20220829120544-10.oplog.s2",
				Size: 1024,
			},
		}

		mockedStorage.On("List", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(listedFiles, nil)
		mockedStorage.On("FileStat", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(minio.FileInfo{}, errors.New("file stat error"))

		ss := NewStorageService()
		ss.storage = mockedStorage
		timelines, err := ss.getPITRTimeRanges(ctx)
		assert.NoError(t, err)
		assert.Len(t, timelines, 0)
	})
}
