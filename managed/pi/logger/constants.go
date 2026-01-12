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

package logger

const (
	// RequestIDAttr attribute name for request ID field in log record.
	RequestIDAttr = "requestID"
	// UserIDAttr attribute name for user ID field in log record.
	UserIDAttr = "userID"
	// AppIDAttr attribute name for application ID field in log record.
	AppIDAttr = "appID"
	// HookAttr attribute name hook field in log record.
	HookAttr = "hook"
)
