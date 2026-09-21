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

import type { SectionField, SepComponents } from '@sep/api';

type Schemas = SepComponents['schemas'];

// ── Category browser ─────────────────────────────────────────────────────

export interface AtwSnippetSummary {
  /** Snippet filename; the identity the batch-execute payload sends. */
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

export type AtwIncident = Schemas['atw__AtwIncidentResponse'];
export type AtwIncidentWrite = Schemas['atw__AtwIncidentWrite'];
export type AtwIncidentUpdate = Schemas['atw__AtwIncidentUpdate'];

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

export type AtwBatchExecuteWrite = Schemas['atw__ATWBatchExecuteWrite'];
export type AtwBatchExecuteItemWrite = Schemas['atw__ATWBatchExecuteItemWrite'];
export type AtwBatchExecuteResponse = Schemas['atw__ATWBatchExecuteResponse'];
export type AtwBatchExecuteItemResponse =
  Schemas['atw__ATWBatchExecuteItemResponse'];

/**
 * A batch dispatch remembered for this browser tab, so "Run again" and "Edit
 * parameters and run again" have something to act on.
 *
 * Nothing re-runnable rides on the wire today: an execution's `masked_args` is
 * a display string with credential values masked out (or withheld entirely),
 * and carries no host, sudo choice, or structured args. So this is
 * session-local only — a page reload, or an execution this tab never
 * dispatched, has nothing remembered for it, and the caller falls back to
 * reselecting the snippet with no parameter values.
 */
export interface AtwRememberedDispatch {
  /** The batch's snippets, in the order the form rendered them. */
  snippets: AtwSnippetSummary[];
  /** The submitted form values, keyed exactly as `SchemaFormRenderer` registers them. */
  values: Record<string, unknown>;
}

/** A request to reopen the Collect form for one past execution. */
export interface AtwRerunRequest {
  /**
   * Bumped on every request, so the form remounts (and re-seeds its defaults)
   * even when the snippet selection this request asks for is unchanged from
   * what is already selected.
   */
  nonce: number;
  /** The snippet to reselect when nothing was remembered for this execution. */
  snippetFilename: string;
  /** The exact batch this execution belonged to, when this tab dispatched it. */
  remembered?: AtwRememberedDispatch;
}

// ── Incident execution history ───────────────────────────────────────────

/**
 * One recorded execution, as the Results pane renders it.
 *
 * `snippet_title` and `executor_host` are widened onto the generated schema
 * rather than waiting for it: the side-car adds both under PMM-15520, and
 * neither can be reached from this pane any other way. The title lives on a
 * different resource, and no endpoint that serves titles yields a complete
 * filename-to-title map — the category listing is filtered by a presentation
 * tag and the search endpoint is paginated, so an execution can name a snippet
 * neither returns. The host is recorded on the batch, not the execution, and
 * `/api/sep/task-history/` filters only by task name and status. So both are
 * optional, and every reader falls back: the title to the filename, the host to
 * showing nothing. Drop these two once the regenerated spec carries them.
 */
export type AtwIncidentExecution =
  Schemas['atw__ATWIncidentExecutionResponse'] & {
    snippet_title?: string | null;
    executor_host?: string | null;
  };

// ── Diagnostics send ─────────────────────────────────────────────────────

export type AtwSendJobWrite = Schemas['atw__AtwSendJobWrite'];
export type AtwSendLog = Schemas['atw__AtwSendLogResponse'];
export type AtwConfig = Schemas['atw__AtwConfigResponse'];
export type AtwCaseMatch = Schemas['atw__AtwCaseMatch'];
export type AtwCaseSearchResponse = Schemas['atw__AtwCaseSearchResponse'];

/** One page of a paginated list endpoint, as the API envelope carries it. */
export interface AtwPage<T> {
  items: T[];
  total: number;
  offset: number;
  limit: number;
}

/** The offset/limit window a paginated list is currently showing. */
export interface AtwPageParams {
  offset: number;
  limit: number;
}

/**
 * One execution snapshotted onto a send log's `detail`, so a failed attempt can
 * be re-sent with the same selection even after the incident has moved on.
 */
export interface AtwSendLogExecution {
  id: string;
  task_history_id: number;
  snippet_filename: string;
}

/**
 * One step the delivery plan reported while the send ran.
 *
 * `kind` is optional only for history: the backend writes it on every entry it
 * records now, but rows persisted before it existed carry entries without it.
 */
export interface AtwSendLogStep {
  name: string;
  kind?: 'resolution' | 'upload';
  status: 'running' | 'success' | 'failed';
  outputs: Record<string, string> | null;
}

/**
 * The evidence a send attempt records.
 *
 * The backend column is free-form JSON, so the generated client types it as an
 * opaque record; this is the shape the orchestrator actually writes. Every field
 * is optional because a row accumulates them as the attempt progresses — a
 * pending row carries only `executions`.
 */
export interface AtwSendLogDetail {
  executions?: AtwSendLogExecution[];
  steps?: AtwSendLogStep[];
  upload_response?: Record<string, unknown> | null;
  upload_reference?: string | null;
  bundle_size?: number;
  file_count?: number;
  error?: string;
}
