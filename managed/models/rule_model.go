// Copyright (C) 2017 Percona LLC
// //
// // This program is free software: you can redistribute it and/or modify
// // it under the terms of the GNU Affero General Public License as published by
// // the Free Software Foundation, either version 3 of the License, or
// // (at your option) any later version.
// //
// // This program is distributed in the hope that it will be useful,
// // but WITHOUT ANY WARRANTY; without even the implied warranty of
// // MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// // GNU Affero General Public License for more details.
// //
// // You should have received a copy of the GNU Affero General Public License
// // along with this program. If not, see <https://www.gnu.org/licenses/>.
package models

//
// import (
// 	"database/sql/driver"
//
// 	"gopkg.in/reform.v1"
// )
//
// // FilterType represents rule filter type.
// type FilterType string
//
// // Available filter types.
// const (
// 	Equal = FilterType("=")
// 	Regex = FilterType("=~")
// )
//
// // Filters represents filters slice.
// type Filters []Filter
//
// // Value implements database/sql/driver Valuer interface.
// func (t Filters) Value() (driver.Value, error) { return jsonValue(t) }
//
// // Scan implements database/sql Scanner interface.
// func (t *Filters) Scan(src interface{}) error { return jsonScan(t, src) }
//
// // Filter represents rule filter.
// type Filter struct {
// 	Type FilterType `json:"type"`
// 	Key  string     `json:"key"`
// 	Val  string     `json:"value"`
// }
//
// // Value implements database/sql/driver.Valuer interface. Should be defined on the value.
// func (f Filter) Value() (driver.Value, error) { return jsonValue(f) }
//
// // Scan implements database/sql.Scanner interface. Should be defined on the pointer.
// func (f *Filter) Scan(src interface{}) error { return jsonScan(f, src) }
//
// // ChannelIDs is a slice of notification channel ids.
// type ChannelIDs []string
//
// // Value implements database/sql/driver Valuer interface.
// func (t ChannelIDs) Value() (driver.Value, error) { return jsonValue(t) }
//
// // Scan implements database/sql Scanner interface.
// func (t *ChannelIDs) Scan(src interface{}) error { return jsonScan(t, src) }
//
// // check interfaces.
// var (
// 	_ reform.BeforeInserter = (*Rule)(nil)
// 	_ reform.BeforeUpdater  = (*Rule)(nil)
// 	_ reform.AfterFinder    = (*Rule)(nil)
// )
