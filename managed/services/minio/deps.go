package minio

import (
	"context"

	"github.com/minio/minio-go/v7"
)

//go:generate ../../../bin/mockery -name=minioClient -case=snake -inpkg -testonly
type minioClient interface {
	ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
	StatObject(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
}
