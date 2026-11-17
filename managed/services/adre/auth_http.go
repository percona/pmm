// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package adre

import "net/http"

func grafanaAuthHeadersFromRequest(r *http.Request) http.Header {
	h := make(http.Header)
	if v := r.Header.Get("Authorization"); v != "" {
		h.Set("Authorization", v)
	}
	if v := r.Header.Get("Cookie"); v != "" {
		h.Set("Cookie", v)
	}
	return h
}
