package minio

/*import (
	"context"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestList(t *testing.T) {
	ctx := context.Background()
	s := New()

	t.Run("with empty prefix", func(t *testing.T) {
	})

	t.Run("with set prefix", func(t *testing.T) {
		mockedMinio := &mockMinioClient{}
		mockedMinio.On("ListObjects", ctx, mock.Anything, mock.Anything).Return(func() <-chan minio.ObjectInfo {
			objectsCh := make(chan minio.ObjectInfo, 1)
			defer close(objectsCh)
			objectsCh <- minio.ObjectInfo{
				ETag:        "89e12a84d2a063350c45516259fceba3",
				Key:         "pbmPitr/rs0/20220829/20220829115611-1.20220829120544-10.oplog.s2",
				Size:        17149,
				ContentType: "",
			}
			return objectsCh
		}())

		objects, err := s.List(ctx, mockedMinio, "michael.test.backup101", "pbmPitr", "")

		assert.NoError(t, err)
		assert.Len(t, objects, 1)
	})

	t.Run("with empty suffix", func(t *testing.T) {
	})

	t.Run("with set suffix", func(t *testing.T) {
	})
}

func TestFileStat(t *testing.T) {
	t.Run("error on delete marker", func(t *testing.T) {
	})

	t.Run("successfully stats file", func(t *testing.T) {
		ctx := context.Background()
		mockedMinio := &mockMinioClient{}
		object := minio.ObjectInfo{
			ETag:        "89e12a84d2a063350c45516259fceba3",
			Key:         "pbmPitr/rs0/20220829/20220829115611-1.20220829120544-10.oplog.s2",
			Size:        17149,
			ContentType: "",
		}
		mockedMinio.On("StatObject", ctx, mock.Anything, mock.Anything, mock.Anything).Return(object, nil)

		s := New()

		expectedFile := FileInfo{
			Name: "test_object.tar.gz",
			Size: 17149,
		}
		file, err := s.FileStat(ctx, mockedMinio, "test_bucket", "test_object.tar.gz")
		assert.NoError(t, err)
		assert.Equal(t, expectedFile, file)
	})
}
*/
