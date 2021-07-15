// pmm-managed
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

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/percona/pmm-managed/models"
)

// Service is wrapper around minio client.
type Service struct {
	l *logrus.Entry
}

// New creates new minio service.
func New() *Service {
	return &Service{
		l: logrus.WithField("component", "minio-client"),
	}
}

// BucketExists return true if bucket can be accessed with provided credentials and exists.
func (s *Service) BucketExists(ctx context.Context, endpoint, accessKey, secretKey, name string) (bool, error) {
	minioClient, err := newClient(endpoint, accessKey, secretKey)
	if err != nil {
		return false, err
	}
	return minioClient.BucketExists(ctx, name)
}

// GetBucketLocation retrieves bucket location by specified bucket name.
func (s *Service) GetBucketLocation(ctx context.Context, endpoint, accessKey, secretKey, name string) (string, error) {
	minioClient, err := newClient(endpoint, accessKey, secretKey)
	if err != nil {
		return "", err
	}
	return minioClient.GetBucketLocation(ctx, name)
}

// RemoveRecursive removes objects recursively from storage with given prefix.
func (s *Service) RemoveRecursive(ctx context.Context, endpoint, accessKey, secretKey, bucketName, prefix string) (rerr error) {
	minioClient, err := newClient(endpoint, accessKey, secretKey)
	if err != nil {
		return err
	}

	objectsCh := make(chan minio.ObjectInfo)
	var g errgroup.Group
	g.Go(func() error {
		defer close(objectsCh)

		options := minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: true,
		}
		for object := range minioClient.ListObjects(ctx, bucketName, options) {
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
	for rErr := range minioClient.RemoveObjects(ctx, bucketName, objectsCh, minio.RemoveObjectsOptions{}) {
		errorsEncountered = true
		s.l.WithError(rErr.Err).Debugf("failed to remove object %q", rErr.ObjectName)
	}

	if errorsEncountered {
		return errors.Errorf("errors encountered while removing objects from bucket %q", bucketName)
	}

	return nil
}

func newClient(endpoint, accessKey, secretKey string) (*minio.Client, error) {
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
