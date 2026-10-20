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

package commands

import (
	"strings"

	"github.com/percona/pmm/utils/servererror"
)

// ServerErrorMessage renders a PMM Server error response for humans, adding a hint about the
// likely cause where one can be derived from the response.
func ServerErrorMessage(e Error) string {
	hint := servererror.AuthHint(e.Code, e.GRPCCode)
	if hint == "" {
		return e.Error
	}

	// The hint is a sentence of its own, so exactly one period has to separate it from the
	// message. PMM Server messages usually end with one already, and keeping it too would
	// produce "Internal server error.. Please check PMM Server logs.". Only the final
	// period goes, so a message which ends in an ellipsis keeps it.
	msg := strings.TrimRight(e.Error, " \t\r\n")
	msg = strings.TrimSuffix(msg, ".")

	// PMM Server sends no message at all on some paths, and a hint must not be introduced
	// by a period with nothing in front of it.
	if msg == "" {
		return hint + "."
	}

	return msg + ". " + hint + "."
}
