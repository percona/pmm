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
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/types/known/timestamppb"

	backupv1beta1 "github.com/percona/pmm/api/managementpb/backup"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/minio"
)

const (
	// PITRfsPrefix is a prefix (folder) for PITR chunks on the storage
	PITRfsPrefix = "pbmPitr"
)

var errUnsupportedLocation = errors.New("unsupported location config")

// StorageService helps perform file lookups in a backup storage location
type StorageService struct {
	l       *logrus.Entry
	storage backupStorage
}

// oplogChunk is index metadata for the oplog chunks
type oplogChunk struct {
	RS          string              `bson:"rs"`
	FName       string              `bson:"fname"`
	Compression CompressionType     `bson:"compression"`
	StartTS     primitive.Timestamp `bson:"start_ts"`
	EndTS       primitive.Timestamp `bson:"end_ts"`
	size        int64               `bson:"-"`
}

// CompressionType is the type of compression used for PITR oplog
type CompressionType string

const (
	CompressionTypeNone      CompressionType = "none"
	CompressionTypeGZIP      CompressionType = "gzip"
	CompressionTypePGZIP     CompressionType = "pgzip"
	CompressionTypeSNAPPY    CompressionType = "snappy"
	CompressionTypeLZ4       CompressionType = "lz4"
	CompressionTypeS2        CompressionType = "s2"
	CompressionTypeZstandard CompressionType = "zstd"
)

// FileCompression return compression alg based on given file extension
func FileCompression(ext string) CompressionType {
	switch ext {
	case "gz":
		return CompressionTypePGZIP
	case "lz4":
		return CompressionTypeLZ4
	case "snappy":
		return CompressionTypeSNAPPY
	case "s2":
		return CompressionTypeS2
	case "zst":
		return CompressionTypeZstandard
	default:
		return CompressionTypeNone
	}
}

// NewStorageService creates new backup storage service.
func NewStorageService() *StorageService {
	return &StorageService{
		l: logrus.WithField("component", "services/backup/storage"),
	}
}

func (ss *StorageService) getPITRTimeRanges(ctx context.Context) ([]*backupv1beta1.PitrTimeline, error) {
	var err error
	var timeranges []*backupv1beta1.PitrTimeline

	pitrf, err := ss.storage.List(ctx, PITRfsPrefix, "")
	if err != nil {
		return timeranges, errors.Wrap(err, "get list of pitr chunks")
	}
	if len(pitrf) == 0 {
		return timeranges, nil
	}

	var pitr []interface{}
	for _, f := range pitrf {
		_, err := ss.storage.FileStat(ctx, PITRfsPrefix+"/"+f.Name)
		if err != nil {
			ss.l.Warningf("skip pitr chunk %s/%s because of %v", PITRfsPrefix, f.Name, err)
			continue
		}
		chunk := pitrMetaFromFileName(f.Name)
		if chunk != nil {
			pitr = append(pitr, chunk)
		}
	}

	for _, tr := range pitr {
		switch tr.(type) {
		case *oplogChunk:
			start := time.Unix(int64(tr.(*oplogChunk).StartTS.T), 0)
			end := time.Unix(int64(tr.(*oplogChunk).EndTS.T), 0)
			timeranges = append(timeranges, &backupv1beta1.PitrTimeline{
				StartTimestamp: timestamppb.New(start),
				EndTimestamp:   timestamppb.New(end),
				Filename:       tr.(*oplogChunk).FName,
			})

		default:
			continue
		}
	}
	return timeranges, nil
}

func (ss *StorageService) ListPITRTimelines(ctx context.Context, location models.BackupLocation) ([]*backupv1beta1.PitrTimeline, error) {
	var err error
	switch {
	case location.S3Config != nil:
		ss.storage, err = minio.NewClient(location.S3Config.Endpoint, location.S3Config.AccessKey, location.S3Config.SecretKey, location.S3Config.BucketName)
		if err != nil {
			return nil, err
		}
	default:
		// todo(idoqo): add support for local storage after https://github.com/percona/pmm/pull/1158/
		return nil, errUnsupportedLocation
	}

	return ss.getPITRTimeRanges(ctx)
}

// pitrMetaFromFileName parses given file name and returns PITRChunk metadata
// it returns nil if the file wasn't parse successfully (e.g. wrong format)
// current fromat is 20200715155939-0.20200715160029-1.oplog.snappy
// (https://github.com/percona/percona-backup-mongodb/wiki/PITR:-storage-layout)
//
// !!! should be agreed with pbm/pitr.chunkPath()
func pitrMetaFromFileName(f string) *oplogChunk {
	ppath := strings.Split(f, "/")
	if len(ppath) < 2 {
		return nil
	}
	chnk := &oplogChunk{}
	chnk.RS = ppath[0]
	chnk.FName = path.Join(PITRfsPrefix, f)

	fname := ppath[len(ppath)-1]
	fparts := strings.Split(fname, ".")
	if len(fparts) < 3 || fparts[2] != "oplog" {
		return nil
	}
	if len(fparts) == 4 {
		chnk.Compression = FileCompression(fparts[3])
	} else {
		chnk.Compression = CompressionTypeNone
	}

	start := pitrParseTS(fparts[0])
	if start == nil {
		return nil
	}
	end := pitrParseTS(fparts[1])
	if end == nil {
		return nil
	}

	chnk.StartTS = *start
	chnk.EndTS = *end

	return chnk
}

func pitrParseTS(tstr string) *primitive.Timestamp {
	tparts := strings.Split(tstr, "-")
	t, err := time.Parse("20060102150405", tparts[0])
	if err != nil {
		// just skip this file
		return nil
	}
	ts := primitive.Timestamp{T: uint32(t.Unix())}
	if len(tparts) > 1 {
		ti, err := strconv.Atoi(tparts[1])
		if err != nil {
			// just skip this file
			return nil
		}
		ts.I = uint32(ti)
	}

	return &ts
}
