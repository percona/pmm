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

package logs

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	logsv1 "github.com/percona/pmm/api/logs/v1"
)

const defaultContextLines = 50

// Service implements LogsService over ClickHouse otel.logs.
type Service struct {
	logsv1.UnimplementedLogsServiceServer

	db *sql.DB
	l  *logrus.Entry
}

// New returns a logs service reading from the ClickHouse otel database.
func New(db *sql.DB) *Service {
	return &Service{db: db, l: logrus.WithField("component", "logs")}
}

// checkLBAC refuses users with label-based access filters: log lines carry no labels to filter on.
func checkLBAC(ctx context.Context) error {
	md, _ := metadata.FromIncomingContext(ctx)
	for _, v := range md.Get("x-proxy-filter") {
		if v != "" {
			return status.Error(codes.PermissionDenied, "Logs are not available to users with label-based access filters.")
		}
	}
	return nil
}

// start checks access and converts the request filter.
func start(ctx context.Context, f *logsv1.LogFilter) (Filter, error) {
	err := checkLBAC(ctx)
	if err != nil {
		return Filter{}, err
	}
	res := Filter{
		Start:       f.GetStart().AsTime(),
		End:         f.GetEnd().AsTime(),
		NodeNames:   f.GetNodeNames(),
		Sources:     f.GetSources(),
		Presets:     f.GetPresets(),
		MinSeverity: f.GetMinSeverity(),
		Text:        f.GetText(),
		Pattern:     f.GetPattern(),
	}
	if res.End.Before(res.Start) {
		return Filter{}, status.Error(codes.InvalidArgument, "The end of the time range is before its start.")
	}
	if res.End.Sub(res.Start) > maxRange {
		return Filter{}, status.Errorf(codes.InvalidArgument, "The time range can be at most %d days.", maxRangeDays)
	}
	return res, nil
}

// query runs q and calls scan for each row.
func (s *Service) query(ctx context.Context, q string, args []any, scan func(*sql.Rows) error) error {
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return s.queryError(err)
	}
	defer rows.Close() //nolint:errcheck
	for rows.Next() {
		err = scan(rows)
		if err != nil {
			return s.queryError(err)
		}
	}
	err = rows.Err()
	if err != nil {
		return s.queryError(err)
	}
	return nil
}

func (s *Service) queryError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "TIMEOUT_EXCEEDED"), strings.Contains(msg, "code: 159,"):
		return status.Error(codes.DeadlineExceeded, "The query took longer than 30 s; narrow the time range or add filters.")
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "Request canceled.")
	}
	s.l.WithError(err).Error("Logs query failed.")
	return status.Error(codes.Internal, "Failed to query logs.")
}

func scanLine(rows *sql.Rows, extra ...any) (*logsv1.LogLine, error) {
	var (
		l  logsv1.LogLine
		ts time.Time
	)
	dest := append([]any{
		&ts, &l.NodeName, &l.Source, &l.Preset, &l.FileName, &l.SeverityNumber, &l.SeverityText,
		&l.Body, &l.BodyTruncated, &l.Attributes,
	}, extra...)
	err := rows.Scan(dest...)
	if err != nil {
		return nil, err
	}
	l.Time = timestamppb.New(ts)
	return &l, nil
}

// SearchLogs returns log lines, newest first, one page at a time.
func (s *Service) SearchLogs(ctx context.Context, req *logsv1.SearchLogsRequest) (*logsv1.SearchLogsResponse, error) {
	f, err := start(ctx, req.Filter)
	if err != nil {
		return nil, err
	}
	var after *Cursor
	if req.Cursor != "" {
		c, err := DecodeCursor(req.Cursor)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "Invalid cursor.")
		}
		after = &c
	}
	q, args := searchQuery(f, after, int(req.Limit))
	res := &logsv1.SearchLogsResponse{}
	var last Cursor
	err = s.query(ctx, q, args, func(rows *sql.Rows) error {
		var h uint64
		l, err := scanLine(rows, &h)
		if err != nil {
			return err
		}
		ts := l.Time.AsTime()
		if ts.Equal(last.Time) && h == last.Hash {
			last.Skip++
		} else {
			last = Cursor{Time: ts, Hash: h, Skip: 1}
		}
		res.Lines = append(res.Lines, l)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(res.Lines) == pageSize(int(req.Limit)) {
		// A page of nothing but lines identical to the cursor's continues its count.
		if after != nil && last.Skip == len(res.Lines) && after.Time.Equal(last.Time) && after.Hash == last.Hash {
			last.Skip += after.Skip
		}
		res.NextCursor = last.Encode()
	}
	return res, nil
}

// GetLogVolume returns line counts per severity class over time.
func (s *Service) GetLogVolume(ctx context.Context, req *logsv1.GetLogVolumeRequest) (*logsv1.GetLogVolumeResponse, error) {
	f, err := start(ctx, req.Filter)
	if err != nil {
		return nil, err
	}
	q, args, step := volumeQuery(f)
	res := &logsv1.GetLogVolumeResponse{Step: durationpb.New(step)}
	err = s.query(ctx, q, args, func(rows *sql.Rows) error {
		var (
			ts                         time.Time
			errs, warns, infos, others uint64
		)
		err := rows.Scan(&ts, &errs, &warns, &infos, &others)
		if err != nil {
			return err
		}
		res.Buckets = append(res.Buckets, &logsv1.VolumeBucket{
			Start:   timestamppb.New(ts),
			Error:   int64(errs),
			Warning: int64(warns),
			Info:    int64(infos),
			Other:   int64(others),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// GetLogPatterns groups lines that differ only in values.
func (s *Service) GetLogPatterns(ctx context.Context, req *logsv1.GetLogPatternsRequest) (*logsv1.GetLogPatternsResponse, error) {
	f, err := start(ctx, req.Filter)
	if err != nil {
		return nil, err
	}
	q, args := patternsQuery(f)
	res := &logsv1.GetLogPatternsResponse{}
	err = s.query(ctx, q, args, func(rows *sql.Rows) error {
		var (
			p                 logsv1.LogPattern
			current, previous uint64
			first, last       time.Time
		)
		err := rows.Scan(&p.Pattern, &current, &previous, &p.IsNew, &p.MaxSeverityNumber, &first, &last, &p.Sample)
		if err != nil {
			return err
		}
		p.Count = int64(current)
		p.FirstSeen = timestamppb.New(first)
		p.LastSeen = timestamppb.New(last)
		res.Patterns = append(res.Patterns, &p)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// GetLogContext returns the lines around one line of a file.
func (s *Service) GetLogContext(ctx context.Context, req *logsv1.GetLogContextRequest) (*logsv1.GetLogContextResponse, error) {
	err := checkLBAC(ctx)
	if err != nil {
		return nil, err
	}
	n := int(req.Lines)
	if n == 0 {
		n = defaultContextLines
	}
	at := req.Time.AsTime()
	q, args := contextQuery(at, req.NodeName, req.FileName, req.Source, n)
	res := &logsv1.GetLogContextResponse{}
	anchored := false
	err = s.query(ctx, q, args, func(rows *sql.Rows) error {
		l, err := scanLine(rows)
		if err != nil {
			return err
		}
		// The first line at the requested time, else the last one before it.
		if ts := l.Time.AsTime(); !anchored && !ts.After(at) {
			res.AnchorIndex = int32(len(res.Lines)) //nolint:gosec
			anchored = ts.Equal(at)
		}
		res.Lines = append(res.Lines, l)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// GetLogFacets returns the nodes, sources and presets with lines in the time range.
func (s *Service) GetLogFacets(ctx context.Context, req *logsv1.GetLogFacetsRequest) (*logsv1.GetLogFacetsResponse, error) {
	f, err := start(ctx, req.Filter)
	if err != nil {
		return nil, err
	}
	nodes, sources, presets, args := facetsQueries(f)
	res := &logsv1.GetLogFacetsResponse{}
	for _, facet := range []struct {
		q   string
		out *[]*logsv1.Facet
	}{
		{nodes, &res.NodeNames},
		{sources, &res.Sources},
		{presets, &res.Presets},
	} {
		err = s.query(ctx, facet.q, args, func(rows *sql.Rows) error {
			var (
				v string
				c uint64
			)
			err := rows.Scan(&v, &c)
			if err != nil {
				return err
			}
			*facet.out = append(*facet.out, &logsv1.Facet{Value: v, Count: int64(c)})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// Check interfaces.
var _ logsv1.LogsServiceServer = (*Service)(nil)
