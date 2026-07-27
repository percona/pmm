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

import type { SectionField, TaskHistoryStatus } from '@sep/api';

// ── Category browser ─────────────────────────────────────────────────────

export interface AtwSnippetSummary {
  /** Snippet filename; use with snippet plugin API path helpers. */
  name: string;
  title: string;
  description: string;
}

export interface AtwCategoryListing {
  category_root: string;
  parent_category: string;
  parent_category_label: string;
  category: string;
  category_label: string;
  snippet_count: number;
  snippets: AtwSnippetSummary[];
}

// ── Incidents ────────────────────────────────────────────────────────────
//
// STUB TYPES (backend-assumed): the incident/batch-execution contracts below
// are served by the ATW backend (SEP-1591 / SEP-1592) but are not yet in PMM's
// committed OpenAPI spec, so they are authored here to mirror the backend
// ``atw__*`` schemas. Replace with generated ``SepComponents['schemas'][...]``
// references once ``specs/sep.json`` is regenerated.

export interface AtwIncident {
  case_ref: string | null;
  /** Format: date-time */
  created_at: string;
  created_by: string;
  /** Format: uuid4 */
  id: string;
  name: string;
  updated_at: string | null;
}

export interface AtwIncidentWrite {
  case_ref?: string | null;
  name?: string;
}

export interface AtwIncidentUpdate {
  case_ref?: string | null;
  name?: string;
}

// ── Merged execution schema ──────────────────────────────────────────────

/**
 * The merged execution form for a batch selection. `shared` holds the
 * batch-level execution fields plus every parameter the selection declares
 * identically; `per_snippet` holds each snippet's remaining fields. The wire
 * field union matches the framework `SectionField` shape the SchemaFormRenderer
 * consumes, so both arrays are typed as `SectionField[]`.
 */
export interface AtwMergedSchema {
  shared: SectionField[];
  per_snippet: AtwSnippetSchema[];
}

export interface AtwSnippetSchema {
  snippet_filename: string;
  fields: SectionField[];
}

// ── Batch execution ──────────────────────────────────────────────────────

export interface AtwBatchExecuteItemWrite {
  args: Record<string, unknown>;
  snippet_filename: string;
}

export interface AtwBatchExecuteWrite {
  executor_host: string;
  items: AtwBatchExecuteItemWrite[];
  shared_args: Record<string, unknown>;
  sudo: boolean;
}

export interface AtwBatchExecuteItemResponse {
  error?: string | Record<string, unknown>[] | null;
  snippet_filename: string;
  task_history_id?: number | null;
  task_name?: string | null;
}

export interface AtwBatchExecuteResponse {
  items: AtwBatchExecuteItemResponse[];
}

// ── Incident execution history ───────────────────────────────────────────

export interface AtwIncidentExecution {
  /** Format: date-time */
  created_at: string;
  finished_at?: string | null;
  has_logs?: boolean | null;
  /** Format: uuid4 */
  id: string;
  snippet_filename: string;
  started_at?: string | null;
  task_history_id: number;
  task_status?: TaskHistoryStatus | null;
}
