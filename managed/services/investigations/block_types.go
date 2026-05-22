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

// Package investigations implements the HTTP/JSON API and helpers for
// the Investigations feature (incident reports, blocks, timeline, comments).
package investigations

// Block type constants for investigation_blocks.type.
// Panel blocks use config_json with: dashboard_uid, panel_id, time_from, time_to (optional).
const (
	BlockTypeSummary           = "summary"
	BlockTypeMarkdown          = "markdown"
	BlockTypeTimeline          = "timeline"
	BlockTypeSinglePanel       = "single_panel"
	BlockTypePanelGroup        = "panel_group"
	BlockTypeLogsView          = "logs_view"
	BlockTypeQueryResult       = "query_result"
	BlockTypeFinding           = "finding"
	BlockTypeSlowQueryAnalysis = "slow_query_analysis"
	BlockTypeTopQueries        = "top_queries"
	BlockTypeSchemaView        = "schema_view"
	BlockTypeRemediationSteps  = "remediation_steps"
	BlockTypeImage             = "image"
	BlockTypeCommentThread     = "comment_thread"
	BlockTypeChatThread        = "chat_thread"
	BlockTypeAttachments       = "attachments"
)
