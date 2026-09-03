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

import { useCallback, useMemo, useState } from 'react';
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
import type { FormSection, SectionField } from '@sep/api';
import { CategoryBrowser } from './CategoryBrowser';
import { useAtwBatchExecute, useAtwMergedSchema } from './hooks';
import type {
  AtwBatchExecuteResponse,
  AtwBatchExecuteWrite,
  AtwSnippetSummary,
} from './types';

export interface CollectPaneProps {
  incidentId: string;
}

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
 * The Collect pane: browse categories to feed a snippet multi-select, render the
 * merged execution form (a shared-parameter section plus one override card per
 * selected snippet), and batch-execute every selection against the incident.
 * Per-task status is polled by the Results pane's execution list.
 */
export function CollectPane({ incidentId }: CollectPaneProps) {
  const [available, setAvailable] = useState<AtwSnippetSummary[]>([]);
  const [selected, setSelected] = useState<AtwSnippetSummary[]>([]);
  const [itemErrors, setItemErrors] = useState<string[]>([]);

  const handleSnippetsChange = useCallback((snippets: AtwSnippetSummary[]) => {
    setAvailable(snippets);
  }, []);

  const selectedNames = useMemo(
    () => selected.map((snippet) => snippet.name),
    [selected]
  );
  const schemaQuery = useAtwMergedSchema(selectedNames);
  const batchMutation = useAtwBatchExecute(incidentId);

  // Options merge the current category's snippets with the current selection so
  // already-picked snippets from other categories still render as removable chips.
  const options = useMemo(() => {
    const byName = new Map<string, AtwSnippetSummary>();
    for (const snippet of [...selected, ...available]) {
      byName.set(snippet.name, snippet);
    }
    return [...byName.values()];
  }, [available, selected]);

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

      <CategoryBrowser onSnippetsChange={handleSnippetsChange} />

      <Autocomplete
        multiple
        sx={{ mt: 3 }}
        options={options}
        value={selected}
        onChange={(_event, value) => {
          setSelected(value);
          // The stale batch-result banner belongs to the previous selection.
          setItemErrors([]);
        }}
        getOptionLabel={(option) => option.title}
        isOptionEqualToValue={(option, value) => option.name === value.name}
        renderInput={(params) => (
          <TextField
            {...params}
            label="Snippets"
            placeholder={
              selected.length === 0 ? 'Select snippets to run' : undefined
            }
          />
        )}
      />

      {selected.length === 0 && (
        <Alert severity="info" sx={{ mt: 3 }}>
          Browse a category and select one or more snippets to build the
          execution form.
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

      {selected.length > 0 && schemaQuery.data && (
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
