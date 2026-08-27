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

import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Autocomplete,
  Box,
  CircularProgress,
  TextField,
  Typography,
} from '@mui/material';
import {
  SchemaFormRenderer,
  SNIPPET_FORM_RESERVED_FIELD_NAMES,
} from '@sep/framework';
import { useAuth, type FormSection, type SectionField } from '@sep/api';
import { CategoryBrowser } from './CategoryBrowser';
import {
  useAtwBatchExecute,
  useAtwMergedSchema,
  useAtwSnippetSearch,
} from './hooks';
import type {
  AtwBatchExecuteResponse,
  AtwBatchExecuteWrite,
  AtwSnippetSummary,
} from './types';

export interface CollectPaneProps {
  incidentId: string;
  isClosed?: boolean;
}

/** Pause after the last keystroke before the snippet search fires (ms). */
const SNIPPET_SEARCH_DEBOUNCE_MS = 300;

/** Stable empty list so an idle search does not churn the options memo. */
const NO_SNIPPETS: AtwSnippetSummary[] = [];

/** Stable empty provenance set for when no search result may be trusted. */
const NO_NAMES: ReadonlySet<string> = new Set<string>();

/** Namespace prefix for a selected snippet's override fields, keyed by position. */
function snippetPrefix(index: number): string {
  return `overrides.snip${index}.`;
}

/**
 * Prefix a field's name (and, for a one-of group, its discriminator and branch
 * leaf names) so each snippet's override fields live in their own react-hook-form
 * namespace. Two snippets can declare the same parameter with diverging
 * definitions — both stay per-snippet — so bare names would collide in a single
 * form.
 *
 * Field-gate predicates (`requires`/`forbidden`) reference sibling field names
 * verbatim and are NOT rewritten here, so a gated field would silently stop
 * gating once namespaced. ATW snippet parameters do not declare gates today;
 * {@link fieldDeclaresGate} detects any that appear so the pane can warn loudly
 * instead of rendering a quietly-broken form.
 */
export function namespaceField(
  field: SectionField,
  prefix: string
): SectionField {
  if (field.type === 'one_of') {
    return {
      ...field,
      name: `${prefix}${field.name}`,
      discriminator: `${prefix}${field.discriminator}`,
      branches: field.branches.map((branch) => ({
        ...branch,
        fields: branch.fields.map((leaf) => ({
          ...leaf,
          name: `${prefix}${leaf.name}`,
        })),
      })),
    };
  }
  return { ...field, name: `${prefix}${field.name}` };
}

/** Whether a field (or a one-of group's branch leaf) declares a conditional gate. */
export function fieldDeclaresGate(field: SectionField): boolean {
  if (field.type === 'one_of') {
    return field.branches.some((branch) =>
      branch.fields.some(fieldDeclaresGate)
    );
  }
  return Boolean(field.requires?.length) || Boolean(field.forbidden?.length);
}

/** Drop reserved UI fields and empty optional values from an args bag. */
function toArgs(values: Record<string, unknown>): Record<string, unknown> {
  const args: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(values)) {
    if (SNIPPET_FORM_RESERVED_FIELD_NAMES.has(key)) {
      continue;
    }
    if (value === '' || value === undefined) {
      continue;
    }
    args[key] = value;
  }
  return args;
}

/** Assemble the batch-execute request from the merged form's coerced values. */
export function buildBatchPayload(
  values: Record<string, unknown>,
  snippets: AtwSnippetSummary[]
): AtwBatchExecuteWrite {
  const overrides = (values.overrides ?? {}) as Record<
    string,
    Record<string, unknown>
  >;
  const sharedValues: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(values)) {
    if (key === 'overrides') {
      continue;
    }
    sharedValues[key] = value;
  }
  return {
    executor_host: String(values.executor_host ?? ''),
    sudo: Boolean(values.sudo ?? false),
    shared_args: toArgs(sharedValues),
    items: snippets.map((snippet, index) => ({
      snippet_filename: snippet.name,
      args: toArgs(overrides[`snip${index}`] ?? {}),
    })),
  };
}

/** Summarise the batch response's per-item errors for a banner. */
function batchItemErrors(response: AtwBatchExecuteResponse): string[] {
  return response.items
    .filter((item) => item.error !== null && item.error !== undefined)
    .map((item) => {
      const detail =
        typeof item.error === 'string' ? item.error : 'validation failed';
      return `${item.snippet_filename}: ${detail}`;
    });
}

/**
 * Merge the picker's option sources, deduping on filename.
 *
 * Three independent sources feed one Autocomplete: the current selection (so a
 * snippet picked under another category or search term stays a removable chip),
 * the selected leaf category's snippets, and the server-side search results. A
 * snippet reachable through several of them must appear once, and identity is
 * the filename — titles are not unique by contract.
 */
export function mergeSnippetOptions(
  ...sources: readonly AtwSnippetSummary[][]
): AtwSnippetSummary[] {
  const byName = new Map<string, AtwSnippetSummary>();
  for (const source of sources) {
    for (const snippet of source) {
      byName.set(snippet.name, snippet);
    }
  }
  return [...byName.values()];
}

/** Whether a snippet matches a typed term the way the server's search does. */
export function snippetMatchesTerm(
  snippet: AtwSnippetSummary,
  term: string
): boolean {
  const needle = term.trim().toLowerCase();
  if (needle === '') {
    return true;
  }
  return (
    snippet.title.toLowerCase().includes(needle) ||
    snippet.name.toLowerCase().includes(needle) ||
    snippet.description.toLowerCase().includes(needle)
  );
}

/**
 * Filter the merged options against the typed term.
 *
 * Replaces the Autocomplete's default filter, which matches the option label
 * only: the server matches filename and description too, so a search hit whose
 * title does not contain the term would otherwise be fetched and then hidden.
 * Any option the server returned (`serverMatched`) is kept as-is; the
 * category-derived rest is matched locally over the same three fields.
 */
export function filterSnippetOptions(
  options: readonly AtwSnippetSummary[],
  term: string,
  serverMatched: ReadonlySet<string>
): AtwSnippetSummary[] {
  if (term.trim() === '') {
    return [...options];
  }
  return options.filter(
    (option) =>
      serverMatched.has(option.name) || snippetMatchesTerm(option, term)
  );
}

/**
 * The Collect pane: browse categories to feed a snippet multi-select, render the
 * merged execution form (a shared-parameter section plus one override card per
 * selected snippet), and batch-execute every selection against the incident.
 * Per-task status is polled by the Results pane's execution list.
 */
export function CollectPane({
  incidentId,
  isClosed = false,
}: CollectPaneProps) {
  const { canMutate } = useAuth();
  const [available, setAvailable] = useState<AtwSnippetSummary[]>([]);
  const [selected, setSelected] = useState<AtwSnippetSummary[]>([]);
  const [itemErrors, setItemErrors] = useState<string[]>([]);
  const [searchInput, setSearchInput] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');

  const handleSnippetsChange = useCallback((snippets: AtwSnippetSummary[]) => {
    setAvailable(snippets);
  }, []);

  // Matches the Snippet Manager list's 300ms window.
  useEffect(() => {
    const handle = setTimeout(
      () => setDebouncedSearch(searchInput.trim()),
      SNIPPET_SEARCH_DEBOUNCE_MS
    );
    return () => clearTimeout(handle);
  }, [searchInput]);

  useEffect(() => {
    if (!isClosed) {
      return;
    }
    setSelected([]);
    setAvailable([]);
    setItemErrors([]);
  }, [isClosed]);

  const selectedNames = useMemo(
    () => selected.map((snippet) => snippet.name),
    [selected]
  );
  // A read-only session never renders the execute form, so it never needs the
  // merged schema either. Selecting snippets still works — only the fetch and
  // the form are withheld.
  const schemaQuery = useAtwMergedSchema(
    isClosed || !canMutate ? [] : selectedNames
  );
  const batchMutation = useAtwBatchExecute(incidentId);
  const searchQuery = useAtwSnippetSearch(debouncedSearch);

  // A disabled query keeps its previous data, so an emptied box must not leave
  // the last term's hits in the list: read results only while a term is active.
  const searchResults =
    debouncedSearch === ''
      ? NO_SNIPPETS
      : (searchQuery.data?.items ?? NO_SNIPPETS);

  const searchMatchedNames = useMemo(
    () => new Set(searchResults.map((snippet) => snippet.name)),
    [searchResults]
  );

  const options = useMemo(
    () => mergeSnippetOptions(selected, available, searchResults),
    [available, selected, searchResults]
  );

  // Whether the page currently in hand was fetched for the term now in the box.
  // `isPlaceholderData` is the precise signal: under `keepPreviousData` it is set
  // exactly while the previous term's page stands in for an unresolved one. Plain
  // `isFetching` would also fire on a background refetch of the *same* term, and
  // no result is stale then.
  const searchPageIsCurrent = !searchQuery.isPlaceholderData;

  // Server provenance only counts while that page belongs to the typed text —
  // otherwise the stand-in page would smuggle the old term's hits past the local
  // filter, which keeps anything the server matched unconditionally.
  const filterOptions = useCallback(
    (candidates: AtwSnippetSummary[], state: { inputValue: string }) => {
      const fetchedTermIsCurrent =
        state.inputValue.trim() === debouncedSearch && searchPageIsCurrent;
      return filterSnippetOptions(
        candidates,
        state.inputValue,
        fetchedTermIsCurrent ? searchMatchedNames : NO_NAMES
      );
    },
    [debouncedSearch, searchMatchedNames, searchPageIsCurrent]
  );

  // The endpoint pages, so a broad term can match more than one page holds.
  // Report the overflow instead of silently showing the first page. Suppressed
  // while a stand-in page is showing: its total belongs to the previous term,
  // and the notice names the term it counts.
  const searchPagination = searchQuery.data?.pagination ?? null;
  const hiddenMatchCount =
    debouncedSearch !== '' && searchPagination && searchPageIsCurrent
      ? Math.max(searchPagination.total - searchResults.length, 0)
      : 0;

  const sections = useMemo<FormSection[]>(() => {
    const merged = schemaQuery.data;
    if (!merged) {
      return [];
    }
    const result: FormSection[] = [];
    if (merged.shared.length > 0) {
      result.push({ title: 'Shared parameters', fields: merged.shared });
    }
    selected.forEach((snippet, index) => {
      const entry = merged.per_snippet.find(
        (item) => item.snippet_filename === snippet.name
      );
      const fields = entry?.fields ?? [];
      if (fields.length === 0) {
        return;
      }
      result.push({
        title: snippet.title,
        description: snippet.description || undefined,
        collapsible: true,
        fields: fields.map((field) =>
          namespaceField(field, snippetPrefix(index))
        ),
      });
    });
    return result;
  }, [schemaQuery.data, selected]);

  // Namespacing per-snippet fields does not rewrite their gate predicates, so a
  // gated field would stop gating once namespaced. Detect any such snippet and
  // warn loudly rather than render a quietly-broken form. ATW snippets declare
  // no gates today, so this normally stays empty.
  const gatedSnippetTitles = useMemo(() => {
    const merged = schemaQuery.data;
    if (!merged) {
      return [];
    }
    return selected
      .filter((snippet) => {
        const entry = merged.per_snippet.find(
          (item) => item.snippet_filename === snippet.name
        );
        return (entry?.fields ?? []).some(fieldDeclaresGate);
      })
      .map((snippet) => snippet.title);
  }, [schemaQuery.data, selected]);

  const handleSubmit = (values: Record<string, unknown>) => {
    setItemErrors([]);
    batchMutation.mutate(buildBatchPayload(values, selected), {
      onSuccess: (response) => {
        setItemErrors(batchItemErrors(response));
      },
    });
  };

  const submitError = batchMutation.isError
    ? (batchMutation.error?.message ?? 'Batch execution failed')
    : null;

  // Remount the form when the selection changes so react-hook-form rebuilds its
  // registered fields and defaults for the new merged schema.
  const formKey = selectedNames.join('|');

  return (
    <Box>
      <Typography variant="h6" sx={{ mb: 2 }}>
        Collect
      </Typography>

      {isClosed && (
        <Alert severity="info" sx={{ mb: 2 }}>
          This incident is closed. Reopen it to run more diagnostic snippets.
        </Alert>
      )}

      {!isClosed && <CategoryBrowser onSnippetsChange={handleSnippetsChange} />}

      {searchQuery.error && debouncedSearch !== '' && (
        <Alert severity="error" sx={{ mt: 3 }}>
          Snippet search failed: {searchQuery.error.message}
        </Alert>
      )}

      {/*
        Above the picker on purpose: the option popup opens downward and would
        cover a notice rendered under the input, which is exactly when the user
        is reading the results it describes.
      */}
      {hiddenMatchCount > 0 && (
        // Polite, not the Alert default's assertive: this mounts and unmounts as
        // the user keeps typing, and must not interrupt a screen reader mid-word.
        <Alert severity="info" role="status" sx={{ mt: 3 }}>
          Showing the first {searchResults.length} of {searchPagination?.total}{' '}
          snippets matching &ldquo;{debouncedSearch}&rdquo;. Type more of the
          name or description to narrow the results.
        </Alert>
      )}

      <Autocomplete
        multiple
        disabled={isClosed}
        sx={{ mt: 3 }}
        options={options}
        value={selected}
        onChange={(_event, value) => {
          setSelected(value);
          // The stale batch-result banner belongs to the previous selection.
          setItemErrors([]);
        }}
        inputValue={searchInput}
        onInputChange={(_event, value) => setSearchInput(value)}
        filterOptions={filterOptions}
        loading={debouncedSearch !== '' && searchQuery.isFetching}
        loadingText="Searching snippets…"
        noOptionsText={
          searchInput.trim() === ''
            ? 'Type to search every snippet, or pick a category above.'
            : 'No approved snippet matches this search.'
        }
        getOptionLabel={(option) => option.title}
        // Without this MUI keys each row on the label, and titles are not unique
        // by contract — two snippets sharing one would collide on re-render.
        getOptionKey={(option) => option.name}
        isOptionEqualToValue={(option, value) => option.name === value.name}
        renderOption={(props, option) => {
          const { key, ...liProps } = props as typeof props & { key: string };
          return (
            // The server matches filename and description too, so a hit can look
            // unrelated to the typed term.
            <Box
              component="li"
              key={key}
              {...liProps}
              sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}
            >
              <Typography variant="body2">{option.title}</Typography>
              <Typography variant="caption" color="text.secondary">
                {option.name}
              </Typography>
            </Box>
          );
        }}
        renderInput={(params) => (
          <TextField
            {...params}
            label="Snippets"
            placeholder={
              selected.length === 0
                ? 'Search or select snippets to run'
                : undefined
            }
          />
        )}
      />

      {selected.length === 0 && (
        <Alert severity="info" sx={{ mt: 3 }}>
          Search for a snippet by name or description, or browse a category,
          then select one or more snippets to build the execution form.
        </Alert>
      )}

      {selected.length > 0 && schemaQuery.isLoading && (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
          <CircularProgress />
        </Box>
      )}

      {selected.length > 0 && schemaQuery.error && (
        <Alert severity="error" sx={{ mt: 3 }}>
          Failed to load execution form: {schemaQuery.error.message}
        </Alert>
      )}

      {gatedSnippetTitles.length > 0 && (
        <Alert severity="warning" sx={{ mt: 3 }}>
          These snippets use conditional (required/forbidden) fields, which
          batch collection does not evaluate — review their inputs before
          executing: {gatedSnippetTitles.join(', ')}.
        </Alert>
      )}

      {itemErrors.length > 0 && (
        <Alert severity="warning" sx={{ mt: 3 }}>
          Some snippets did not dispatch:
          <Box component="ul" sx={{ m: 0, pl: 2 }}>
            {itemErrors.map((message) => (
              <li key={message}>{message}</li>
            ))}
          </Box>
        </Alert>
      )}

      {selected.length > 0 && schemaQuery.data && !isClosed && canMutate && (
        <Box sx={{ mt: 3 }}>
          <SchemaFormRenderer
            key={formKey}
            sections={sections}
            onSubmit={handleSubmit}
            submitLabel="Execute batch"
            loading={batchMutation.isPending}
            submitError={submitError}
          />
        </Box>
      )}
    </Box>
  );
}
