/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

/**
 * Shared types for the snippets plugin frontend package.
 *
 * Mirrors the Pydantic API models defined in
 * `app/sep/plugins/snippets/models.py`.
 */

export type {
  SnippetExecutionRequest,
  SnippetExecutionResponse,
} from '@sep/framework';

export interface SnippetResponse {
  filename: string;
  title: string;
  description: string;
  size: number;
  md5_digest: string;
  is_approved: boolean;
  approved_at: string | null;
  updated_by: string | null;
  reason: string;
  requires_sudo: boolean;
  sudo_optional: boolean;
  sudo_default: boolean;
  interpreter: string | null;
  created_at: string;
  updated_at: string | null;
}

export interface ScriptPreviewResponse {
  content: string;
  language: string;
  is_truncated: boolean;
}

export interface SnippetBatchApproveRequest {
  filenames: string[];
}

export interface BatchApprovalResponse {
  approved: string[];
  skipped_already_approved: string[];
  count: number;
}

export interface BatchApprovalErrorResponse {
  missing_in_db: string[];
  missing_on_disk: string[];
}

export interface SnippetsCapabilitiesResponse {
  manual_sync_enabled: boolean;
}

export interface RefreshResponse {
  refreshed_at: string;
}
