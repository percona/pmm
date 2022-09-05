package backup

import (
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"

	backupv1beta1 "github.com/percona/pmm/api/managementpb/backup"
)

const (
	// PITRfsPrefix is a prefix (folder) for PITR chunks on the storage
	PITRfsPrefix = "pbmPitr"
)

// StorageService helps perform file lookups in a backup storage location
type StorageService struct {
	l       *logrus.Entry
	storage storagePath
}

// OplogChunk is index metadata for the oplog chunks
type OplogChunk struct {
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
	default:
		return CompressionTypeNone
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
	}
}

// NewStorageService creates new backup storage service.
func NewStorageService(storage storagePath) *StorageService {
	return &StorageService{
		l:       logrus.WithField("component", "services/backup/storage"),
		storage: storage,
	}
}

func (ss *StorageService) ListPITRTimelines(ctx context.Context, locationID []string) ([]*backupv1beta1.PITRTimeline, error) {
	var timeranges []*backupv1beta1.PITRTimeline
	pitrf, err := ss.storage.List(ctx, PITRfsPrefix, "", "")
	if err != nil {
		return timeranges, errors.Wrap(err, "get list of pitr chunks")
	}
	if len(pitrf) == 0 {
		return timeranges, nil
	}

	var pitr []interface{}
	for _, f := range pitrf {
		_, err := ss.storage.FileStat(ctx, "test_bucket", PITRfsPrefix+"/"+f.Name)
		if err != nil {
			ss.l.Warningf("skip pitr chunk %s/%s because of %v", PITRfsPrefix, f.Name, err)
			continue
		}
		chnk := PITRMetaFromFName(f.Name)
		if chnk != nil {
			pitr = append(pitr, chnk)
		}
	}

	for _, tr := range pitr {
		switch tr.(type) {
		case *OplogChunk:
			start := time.Unix(int64(tr.(*OplogChunk).StartTS.T), 0)
			end := time.Unix(int64(tr.(*OplogChunk).EndTS.T), 0)
			timeranges = append(timeranges, &backupv1beta1.PITRTimeline{
				StartTimestamp: timestamppb.New(start),
				EndTimestamp:   timestamppb.New(end),
				Filename:       tr.(*OplogChunk).FName,
			})
		}
	}
	return timeranges, nil
}

// PITRMetaFromFName parses given file name and returns PITRChunk metadata
// it returns nil if file wasn't parse successfully (e.g. wrong format)
// current fromat is 20200715155939-0.20200715160029-1.oplog.snappy
// (https://github.com/percona/percona-backup-mongodb/wiki/PITR:-storage-layout)
//
// !!! should be agreed with pbm/pitr.chunkPath()
func PITRMetaFromFName(f string) *OplogChunk {
	ppath := strings.Split(f, "/")
	if len(ppath) < 2 {
		return nil
	}
	chnk := &OplogChunk{}
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
