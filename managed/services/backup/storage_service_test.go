package backup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/percona/pmm/managed/services/minio"
)

func TestListPITRTimelines(t *testing.T) {
	ctx := context.Background()
	locationIDs := []string{"1111"}

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
	mockedStorage.On("List", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(listedFiles, nil)
	mockedStorage.On("FileStat", mock.Anything, mock.Anything, mock.Anything).Return(statFile, nil)

	ss := NewStorageService(mockedStorage)
	timelines, err := ss.ListPITRTimelines(ctx, locationIDs)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(timelines))
}
