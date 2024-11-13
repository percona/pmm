// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pgstatstatements

//go:generate ../../../../bin/reform

// pgStatDatabase represents a row in pg_stat_database view.
//
//reform:pg_catalog.pg_stat_database
type pgStatDatabase struct {
	DatID   int64   `reform:"datid"`
	DatName *string `reform:"datname"`
}

// pgUser represents a row in pg_user view.
//
//reform:pg_catalog.pg_user
type pgUser struct {
	UserID   int64   `reform:"usesysid"`
	UserName *string `reform:"usename"`
}
