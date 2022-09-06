package backup

import (
	"context"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/minio"
)

func TestListPITRTimelines(t *testing.T) {
	ctx := context.Background()

	mockedStorage := &mockStoragePath{}
	listedFiles := []minio.FileInfo{
		{
			Name: "rs0/20220829/20220829115611-1.20220829120544-10.oplog.s2",
			Size: 1024,
		},
	}

	location := models.BackupLocation{
		Name:        gofakeit.Name(),
		Description: "",
	}
	location.S3Config = &models.S3LocationConfig{
		Endpoint:     "https://awsS3.us-west-2.amazonaws.com/",
		AccessKey:    "access_key",
		SecretKey:    "secret_key",
		BucketName:   "example_bucket",
		BucketRegion: "us-east-1",
	}

	statFile := minio.FileInfo{
		Name: PITRfsPrefix + "rs0/20220829/20220829115611-1.20220829120544-10.oplog.s2",
		Size: 1024,
	}
	mockedStorage.On("List", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(listedFiles, nil)
	mockedStorage.On("FileStat", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(statFile, nil)

	ss := NewStorageService(mockedStorage)
	timelines, err := ss.ListPITRTimelines(ctx, location)
	assert.NoError(t, err)
	assert.Len(t, timelines, 1)
}
