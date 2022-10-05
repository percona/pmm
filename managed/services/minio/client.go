// Copyright (C) 2017 Percona LLC
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

// Package minio provides implementation for Minio operations.
package minio

import (
	"context"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/percona/pmm/managed/models"
)

// Client is a wrapper around minio.Client with the bucket name included.
type Client struct {
	l *logrus.Entry
}

// New creates a client with empty credentials.
func New() *Client {
	return &Client{ //nolint:exhaustruct
		l: logrus.WithField("component", "minio-client"),
	}
}

// FileInfo contains information about a single file in the bucket.
type FileInfo struct {
	Name string // with path
	Size int64
}

// BucketExists return true if bucket can be accessed with provided credentials and exists.
func (c *Client) BucketExists(ctx context.Context, endpoint, accessKey, secretKey, bucketName string) (bool, error) {
	mc, err := createMinioClient(endpoint, accessKey, secretKey)
	if err != nil {
		return false, err
	}
	return mc.BucketExists(ctx, bucketName)
}

// GetBucketLocation retrieves bucket location by specified bucket name.
func (c *Client) GetBucketLocation(ctx context.Context, endpoint, accessKey, secretKey, bucketName string) (string, error) {
	mc, err := createMinioClient(endpoint, accessKey, secretKey)
	if err != nil {
		return "", err
	}
	return mc.GetBucketLocation(ctx, bucketName)
}

// RemoveRecursive removes objects recursively from storage with given prefix.
func (c *Client) RemoveRecursive(ctx context.Context, endpoint, accessKey, secretKey, bucketName, prefix string) (rerr error) {
	mc, err := createMinioClient(endpoint, accessKey, secretKey)
	if err != nil {
		return err
	}

	objectsCh := make(chan minio.ObjectInfo)
	var g errgroup.Group
	g.Go(func() error {
		defer close(objectsCh)

		options := minio.ListObjectsOptions{ //nolint:exhaustruct
			Prefix:    prefix,
			Recursive: true,
		}
		for object := range mc.ListObjects(ctx, bucketName, options) {
			if object.Err != nil {
				return errors.WithStack(object.Err)
			}

			objectsCh <- object
		}

		return nil
	})

	defer func() {
		err := g.Wait()
		if err == nil {
			return
		}

		if rerr != nil {
			rerr = errors.Wrapf(rerr, "listing objects error: %s", err.Error())
		} else {
			rerr = errors.WithStack(err)
		}
	}()

	var errorsEncountered bool
	for rErr := range mc.RemoveObjects(ctx, bucketName, objectsCh, minio.RemoveObjectsOptions{}) {
		errorsEncountered = true
		c.l.WithError(rErr.Err).Debugf("failed to remove object %q", rErr.ObjectName)
	}

	if errorsEncountered {
		return errors.Errorf("errors encountered while removing objects from bucket %q", bucketName)
	}

	return nil
}

// List provides an abstraction over the minio API to list all objects in the bucket
// It scans path with prefix and returns all files with given suffix.
// Both prefix and suffix can be omitted.
func (c *Client) List(ctx context.Context, endpoint, accessKey, secretKey, bucketName, prefix, suffix string) ([]FileInfo, error) {
	var files []FileInfo
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	mc, err := createMinioClient(endpoint, accessKey, secretKey)
	if err != nil {
		return nil, err
	}

	options := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	for object := range mc.ListObjects(ctx, bucketName, options) {
		if object.Err != nil {
			return nil, errors.WithStack(object.Err)
		}
		f := object.Key
		f = strings.TrimPrefix(f, options.Prefix)
		if len(f) == 0 {
			continue
		}
		if f[0] == '/' {
			f = f[1:]
		}

		c.l.Infof(f)

		if strings.HasSuffix(f, suffix) {
			files = append(files, FileInfo{
				Name: f,
				Size: object.Size,
			})
		}
	}

	return files, nil
}

// FileStat returns file info. It returns error if file is empty or not exists.
func (c *Client) FileStat(ctx context.Context, endpoint, accessKey, secretKey, bucketName, name string) (FileInfo, error) {
	var file FileInfo
	mc, err := createMinioClient(endpoint, accessKey, secretKey)
	if err != nil {
		return file, err
	}

	stat, err := mc.StatObject(ctx, bucketName, name, minio.StatObjectOptions{}) //nolint:exhaustruct
	if err != nil {
		return file, err
	}

	if stat.IsDeleteMarker {
		return file, errors.New("file has delete marker")
	}

	file.Name = name
	file.Size = stat.Size

	if file.Size == 0 {
		return file, errors.New("file is empty")
	}

	return file, nil
}

func createMinioClient(endpoint, accessKey, secretKey string) (*minio.Client, error) {
	url, err := models.ParseEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	secure := true
	if url.Scheme == "http" {
		secure = false
	}

	return minio.New(url.Host, &minio.Options{
		Secure: secure,
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
	})
}
