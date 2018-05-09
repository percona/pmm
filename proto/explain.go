/*
   Copyright (c) 2016, Percona LLC and/or its affiliates. All rights reserved.

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package proto

type ExplainQuery struct {
	UUID    string
	Db      string
	Query   string
	Convert bool // convert if not SELECT and MySQL <= 5.5 or >= 5.6 but no privs
}

type ExplainResult struct {
	Classic []*ExplainRow
	JSON    string // since MySQL 5.6.5
}

type ExplainRow struct {
	Id           NullInt64
	SelectType   NullString
	Table        NullString
	Partitions   NullString // split by comma; since MySQL 5.1
	CreateTable  NullString // @todo
	Type         NullString
	PossibleKeys NullString // split by comma
	Key          NullString
	KeyLen       NullString // https://jira.percona.com/browse/PCT-863
	Ref          NullString
	Rows         NullInt64
	Filtered     NullFloat64 // as of 5.7.3
	Extra        NullString  // split by semicolon
}

type Table struct {
	Db    string
	Table string
}

type TableInfoQuery struct {
	UUID   string
	Create []Table // SHOW CREATE TABLE Db.Table
	Index  []Table // SHOW INDEXES FROM Db.Table
	Status []Table // SHOW TABLE STATUS FROM Db LIKE 'Table'
}

// ShowIndexRow describes one row from `SHOW INDEX FROM %s` query.
//# mysql -e 'SHOW INDEX FROM mysql.user\G'
//*************************** 1. row ***************************
//        Table: user
//   Non_unique: 0
//     Key_name: PRIMARY
// Seq_in_index: 1
//  Column_name: Host
//    Collation: A
//  Cardinality: 2
//     Sub_part: NULL
//       Packed: NULL
//         Null:
//   Index_type: BTREE
//      Comment:
//Index_comment:
//      Visible: YES
//*************************** 2. row ***************************
//        Table: user
//   Non_unique: 0
//     Key_name: PRIMARY
// Seq_in_index: 2
//  Column_name: User
//    Collation: A
//  Cardinality: 5
//     Sub_part: NULL
//       Packed: NULL
//         Null:
//   Index_type: BTREE
//      Comment:
//Index_comment:
//      Visible: YES
//
//# mysql -e 'DESCRIBE INFORMATION_SCHEMA.STATISTICS'
//+---------------+------------------+------+-----+---------+-------+
//| Field         | Type             | Null | Key | Default | Extra |
//+---------------+------------------+------+-----+---------+-------+
//| TABLE_CATALOG | varchar(64)      | NO   |     | NULL    |       |
//| TABLE_SCHEMA  | varchar(64)      | NO   |     | NULL    |       |
//| TABLE_NAME    | varchar(64)      | NO   |     | NULL    |       |
//| NON_UNIQUE    | int(1)           | NO   |     | 0       |       |
//| INDEX_SCHEMA  | varchar(64)      | NO   |     | NULL    |       |
//| INDEX_NAME    | varchar(64)      | YES  |     | NULL    |       |
//| SEQ_IN_INDEX  | int(10) unsigned | NO   |     | NULL    |       |
//| COLUMN_NAME   | varchar(64)      | YES  |     | NULL    |       |
//| COLLATION     | varchar(1)       | YES  |     | NULL    |       |
//| CARDINALITY   | bigint(21)       | YES  |     | NULL    |       |
//| SUB_PART      | bigint(21)       | YES  |     | NULL    |       |
//| PACKED        | binary(0)        | YES  |     | NULL    |       |
//| NULLABLE      | varchar(3)       | NO   |     |         |       |
//| INDEX_TYPE    | varchar(11)      | NO   |     |         |       |
//| COMMENT       | varchar(8)       | NO   |     |         |       |
//| INDEX_COMMENT | varchar(2048)    | NO   |     | NULL    |       |
//| IS_VISIBLE    | varchar(3)       | NO   |     |         |       |
//+---------------+------------------+------+-----+---------+-------+
type ShowIndexRow struct {
	Table        string
	NonUnique    bool
	KeyName      string
	SeqInIndex   int
	ColumnName   string
	Collation    NullString
	Cardinality  NullInt64
	SubPart      NullInt64
	Packed       NullString
	Null         NullString
	IndexType    string
	Comment      NullString
	IndexComment NullString
	Visible      NullString
}

type ShowTableStatus struct {
	Name          string
	Engine        string
	Version       string
	RowFormat     string
	Rows          NullInt64
	AvgRowLength  NullInt64
	DataLength    NullInt64
	MaxDataLength NullInt64
	IndexLength   NullInt64
	DataFree      NullInt64
	AutoIncrement NullInt64
	CreateTime    NullTime
	UpdateTime    NullTime
	CheckTime     NullTime
	Collation     NullString
	Checksum      NullString
	CreateOptions NullString
	Comment       NullString
}

type TableInfo struct {
	Create string                    `json:",omitempty"`
	Index  map[string][]ShowIndexRow `json:",omitempty"`
	Status *ShowTableStatus          `json:",omitempty"`
	Errors []string                  `json:",omitempty"`
}

type TableInfoResult map[string]*TableInfo
