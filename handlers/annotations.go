// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/percona/pmm-managed/api"
)

type AnnotationsServer struct {
	Addr string
}

// Create creates a new annotation.
func (s *AnnotationsServer) Create(ctx context.Context, req *api.AnnotationsCreateRequest) (*api.AnnotationsCreateResponse, error) {
	// PMM-2347: We always add `pmm_annotation` tag.
	req.Tags = append(req.Tags, "pmm_annotation")

	// Encode json.
	b := &bytes.Buffer{}
	err := json.NewEncoder(b).Encode(req)
	if err != nil {
		return nil, err
	}

	// Call annotations API with json.
	r, err := http.Post(fmt.Sprintf("http://%s/api/annotations", s.Addr), "application/json", b)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	// Decode response json.
	resp := &api.AnnotationsCreateResponse{}
	err = json.NewDecoder(r.Body).Decode(resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// check interface
var _ api.AnnotationsServer = (*AnnotationsServer)(nil)
