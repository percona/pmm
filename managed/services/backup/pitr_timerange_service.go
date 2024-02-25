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
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/percona/pmm/managed/models"
)

const (
	// - pitrFSPrefix is the prefix (folder) for all PITR artifacts in the backup location.
	pitrFSPrefix = "pbmPitr"
)

var errUnsupportedLocation = errors.New("unsupported location config")

// PBMPITRService helps to perform PITR chunks lookup in a backup storage.
type PBMPITRService struct {
	l *logrus.Entry
}

// NewPBMPITRService creates new backup PBMPITRService service.
func NewPBMPITRService() *PBMPITRService {
	return &PBMPITRService{
		l: logrus.WithField("component", "services/backup/pitr_storage"),
	}
}

// oplogChunk is index metadata for the oplog chunks.
type oplogChunk struct {
	RS          string              `bson:"rs"`
	FName       string              `bson:"fname"`
	Compression compressionType     `bson:"compression"`
	StartTS     primitive.Timestamp `bson:"start_ts"`
	EndTS       primitive.Timestamp `bson:"end_ts"`
	size        int64               `bson:"-"`
}

// Timeline is an internal representation of a PITR Timeline.
type Timeline struct {
	ReplicaSet string `json:"replica_set"`
	Start      uint32 `json:"start"`
	End        uint32 `json:"end"`
	Size       int64  `json:"-"`
}

type gap struct {
	s, e uint32
}

type gaps []gap

func (x gaps) Len() int { return len(x) }
func (x gaps) Less(i, j int) bool {
	return x[i].s < x[j].s || (x[i].s == x[j].s && x[i].e < x[j].e)
}
func (x gaps) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

// compressionType is the type of compression used for PITR oplog.
type compressionType string

const (
	compressionTypeNone      compressionType = "none"
	compressionTypePGZIP     compressionType = "pgzip"
	compressionTypeSNAPPY    compressionType = "snappy"
	compressionTypeLZ4       compressionType = "lz4"
	compressionTypeS2        compressionType = "s2"
	compressionTypeZstandard compressionType = "zstd"
)

// file return compression alg based on given file extension.
func file(ext string) compressionType {
	switch ext {
	case "gz":
		return compressionTypePGZIP
	case "lz4":
		return compressionTypeLZ4
	case "snappy":
		return compressionTypeSNAPPY
	case "s2":
		return compressionTypeS2
	case "zst":
		return compressionTypeZstandard
	default:
		return compressionTypeNone
	}
}

func (s *PBMPITRService) getPITROplogs(ctx context.Context, storage Storage, location *models.BackupLocation, artifact *models.Artifact) ([]*oplogChunk, error) {
	var oplogChunks []*oplogChunk

	if storage == nil {
		return oplogChunks, nil
	}
	if location.S3Config == nil {
		return nil, errUnsupportedLocation
	}

	var prefix string

	// Only artifacts taken with new agents can be restored from artifact folder.
	if len(artifact.MetadataList) == 0 {
		prefix = path.Join(artifact.Name, pitrFSPrefix)
	} else {
		prefix = path.Join(artifact.Folder, pitrFSPrefix)
	}

	s3Config := location.S3Config
	pitrFiles, err := storage.List(ctx, s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.BucketName, prefix, "")
	if err != nil {
		return nil, errors.Wrap(err, "get list of pitr chunks")
	}
	if len(pitrFiles) == 0 {
		return nil, nil
	}

	for _, f := range pitrFiles {
		if f.IsDeleteMarker {
			s.l.Debugf("skip pitr chunk %s/%s because of file has delete marker", prefix, f.Name)
			continue
		}
		chunk := pitrMetaFromFileName(prefix, f.Name)
		if chunk != nil {
			chunk.size = f.Size
			oplogChunks = append(oplogChunks, chunk)
		}
	}

	return oplogChunks, nil
}

// ListPITRTimeranges returns the available PITR timeranges for the given artifact in the provided location.
func (s *PBMPITRService) ListPITRTimeranges(ctx context.Context, storage Storage, location *models.BackupLocation, artifact *models.Artifact) ([]Timeline, error) {
	var timelines [][]Timeline

	oplogs, err := s.getPITROplogs(ctx, storage, location, artifact)
	if err != nil {
		return nil, errors.Wrap(err, "get slice")
	}
	if len(oplogs) == 0 {
		return nil, nil
	}

	t, err := getTimelines(oplogs), nil
	if err != nil {
		return nil, errors.Wrapf(err, "get PITR timeranges for backup '%s'", artifact.Name)
	}
	if len(t) != 0 {
		timelines = append(timelines, t)
	}

	mergedTimelines := mergeTimelines(timelines...)
	trimTimelines(mergedTimelines)

	return mergedTimelines, nil
}

// trimTimelines adds one second to the Start value of every timeline record. Required to fit PBM values.
func trimTimelines(timelines []Timeline) {
	for i := range timelines {
		timelines[i].Start += 1 //nolint:revive
	}
}

// pitrMetaFromFileName parses given file name and returns PITRChunk metadata
// it returns nil if the file wasn't parse successfully (e.g. wrong format)
// current format is 20200715155939-0.20200715160029-1.oplog.snappy
// (https://github.com/percona/percona-backup-mongodb/wiki/PITR:-storage-layout)
//
// !!! Should be agreed with pbm/pitr.chunkPath().
func pitrMetaFromFileName(prefix, f string) *oplogChunk {
	ppath := strings.Split(f, "/")
	if len(ppath) < 2 {
		return nil
	}
	chnk := &oplogChunk{}
	chnk.RS = ppath[0]
	chnk.FName = path.Join(prefix, f)

	fname := ppath[len(ppath)-1]
	fparts := strings.Split(fname, ".")
	if len(fparts) < 3 || fparts[2] != "oplog" {
		return nil
	}
	if len(fparts) == 4 {
		chnk.Compression = file(fparts[3])
	} else {
		chnk.Compression = compressionTypeNone
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

func getTimelines(slices []*oplogChunk) []Timeline {
	var tl Timeline
	var timelines []Timeline
	var prevEnd primitive.Timestamp
	for _, s := range slices {
		if prevEnd.T != 0 && prevEnd.Compare(s.StartTS) == -1 {
			timelines = append(timelines, tl)
			tl = Timeline{}
		}
		if tl.Start == 0 {
			tl.Start = s.StartTS.T
		}
		prevEnd = s.EndTS
		tl.End = s.EndTS.T
		tl.Size += s.size
		tl.ReplicaSet = s.RS
	}

	timelines = append(timelines, tl)
	return timelines
}

// mergeTimelines merges overlapping sets on timelines
// it presumes timelines are sorted and don't start from 0.
func mergeTimelines(timelines ...[]Timeline) []Timeline {
	// fast paths
	if len(timelines) == 0 {
		return nil
	}
	if len(timelines) == 1 {
		return timelines[0]
	}

	// First, we define the available range. It equals to the beginning of the latest start of the first
	// Timeline of any set and to the earliest end of the last Timeline of any set. Then define timelines' gaps
	// merge overlapping and apply resulted gap on the available range.
	//
	// given timelines:
	// 1 2 3 4     7 8 10 11          16 17 18 19 20
	//     3 4 5 6 7 8 10 11 12    15 16 17
	// 1 2 3 4 5 6 7 8 10 11 12       16 17 18
	//
	// available range:
	//     3 4 5 6 7 8 10 11 12 13 15 16 17
	// merged gaps:
	//         5 6           12 13 15       18 19 20
	// result:
	//     3 4     7 8 10 11          16 17
	//

	// limits of the available range
	// `start` is the latest start the timelines range
	// `end` - is the earliest end
	var start, end uint32

	// iterating through the timelines  1) define `start` and `end`,
	// 2) define gaps and add them into slice.
	var g gaps
	for _, tln := range timelines {
		if len(tln) == 0 {
			continue
		}

		if tln[0].Start > start {
			start = tln[0].Start
		}

		if end == 0 || tln[len(tln)-1].End < end {
			end = tln[len(tln)-1].End
		}

		if len(tln) == 1 {
			continue
		}
		var ls uint32
		for i, t := range tln {
			if i == 0 {
				ls = t.End
				continue
			}
			g = append(g, gap{ls, t.Start})
			ls = t.End
		}
	}
	sort.Sort(g)

	// if no gaps, just return available range
	if len(g) == 0 {
		return []Timeline{{Start: start, End: end}}
	}

	// merge overlapping gaps
	var g2 gaps
	var cend uint32
	for _, gp := range g {
		if gp.e <= start {
			continue
		}
		if gp.s >= end {
			break
		}

		if len(g2) != 0 {
			cend = g2[len(g2)-1].e
		}

		if gp.s > cend {
			g2 = append(g2, gp)
			continue
		}
		if gp.e > cend {
			g2[len(g2)-1].e = gp.e
		}
	}

	// split available Timeline with gaps
	var ret []Timeline
	for _, g := range g2 {
		if start < g.s {
			ret = append(ret, Timeline{Start: start, End: g.s})
		}
		start = g.e
	}
	if start < end {
		ret = append(ret, Timeline{Start: start, End: end})
	}

	return ret
}

// GetPITRFiles returns list of PITR chunks. If 'until' specified, returns only chunks created before that date, otherwise returns all artifact chunks.
func (s *PBMPITRService) GetPITRFiles(
	ctx context.Context,
	storage Storage,
	location *models.BackupLocation,
	artifact *models.Artifact,
	until *time.Time,
) ([]*oplogChunk, error) {
	opLogs, err := s.getPITROplogs(ctx, storage, location, artifact)
	if err != nil {
		return nil, err
	}

	if until != nil {
		var res []*oplogChunk
		for _, chunk := range opLogs {
			chunkStartTime := time.Unix(int64(chunk.StartTS.T), 0)
			// We're checking only start time because when pbm takes snapshot, chunk is being finalizing automatically.
			if chunkStartTime.Before(*until) {
				res = append(res, chunk)
			}
		}
		return res, nil
	}

	return opLogs, nil
}
