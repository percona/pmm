// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package pgstatstatements

//go:generate reform

// pgStatDatabase represents a row in pg_stat_database view.
//reform:pg_catalog.pg_stat_database
type pgStatDatabase struct {
	DatID   int64   `reform:"datid"`
	DatName *string `reform:"datname"`
}

// pgUser represents a row in pg_user view.
//reform:pg_catalog.pg_user
type pgUser struct {
	UserID   int64   `reform:"usesysid"`
	UserName *string `reform:"usename"`
}

// pgStatStatements represents a row in pg_stat_statements view.
//reform:pg_stat_statements
type pgStatStatements struct {
	UserID    int64   `reform:"userid"`
	DBID      int64   `reform:"dbid"`
	QueryID   *int64  `reform:"queryid"`
	Query     *string `reform:"query"`
	Calls     int64   `reform:"calls"`
	TotalTime float64 `reform:"total_time"`
	//MinTime           *float64 `reform:"min_time"`
	//MaxTime           *float64 `reform:"max_time"`
	//MeanTime          *float64 `reform:"mean_time"`
	//StddevTime        *float64 `reform:"stddev_time"`
	Rows              int64   `reform:"rows"`
	SharedBlksHit     int64   `reform:"shared_blks_hit"`
	SharedBlksRead    int64   `reform:"shared_blks_read"`
	SharedBlksDirtied int64   `reform:"shared_blks_dirtied"`
	SharedBlksWritten int64   `reform:"shared_blks_written"`
	LocalBlksHit      int64   `reform:"local_blks_hit"`
	LocalBlksRead     int64   `reform:"local_blks_read"`
	LocalBlksDirtied  int64   `reform:"local_blks_dirtied"`
	LocalBlksWritten  int64   `reform:"local_blks_written"`
	TempBlksRead      int64   `reform:"temp_blks_read"`
	TempBlksWritten   int64   `reform:"temp_blks_written"`
	BlkReadTime       float64 `reform:"blk_read_time"`
	BlkWriteTime      float64 `reform:"blk_write_time"`
}
