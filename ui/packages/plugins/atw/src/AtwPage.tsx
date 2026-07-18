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

import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Box,
  CircularProgress,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Typography,
} from '@mui/material';
import type { SelectChangeEvent } from '@mui/material/Select';
import { useSnackbar } from 'notistack';
import {
  SchemaFormRenderer,
  buildSnippetExecutionFormPayload,
  snippetPluginExecutePath,
  snippetPluginSchemaPath,
  useSnippetPluginExecution,
  useSnippetPluginSchema,
} from '@sep/framework';
import { useAtwCategories } from './hooks';
import type { AtwCategoryListing, AtwSnippetSummary } from './types';

export function AtwPage() {
  const { enqueueSnackbar } = useSnackbar();
  const [selectedRoot, setSelectedRoot] = useState<string>('');
  const [selectedParent, setSelectedParent] = useState<string>('');
  const [selectedCategory, setSelectedCategory] = useState<string>('');
  const [selectedSnippet, setSelectedSnippet] = useState<string>('');

  const categoriesQuery = useAtwCategories();

  const rootOptions = useMemo(() => {
    const listing = categoriesQuery.data ?? [];
    return Array.from(new Set(listing.map((item: AtwCategoryListing) => item.category_root)));
  }, [categoriesQuery.data]);

  useEffect(() => {
    if (rootOptions.length !== 1) {
      return;
    }
    const only = rootOptions[0];
    if (selectedRoot === only) {
      return;
    }
    setSelectedRoot(only);
    setSelectedParent('');
    setSelectedCategory('');
    setSelectedSnippet('');
  }, [rootOptions, selectedRoot]);

  const parentOptions = useMemo((): Array<{ value: string; label: string }> => {
    const listing = categoriesQuery.data ?? [];
    const seen = new Set<string>();
    return listing
      .filter((item: AtwCategoryListing) => item.category_root === selectedRoot)
      .filter((item: AtwCategoryListing) => {
        if (seen.has(item.parent_category)) {
          return false;
        }
        seen.add(item.parent_category);
        return true;
      })
      .map((item: AtwCategoryListing) => ({
        value: item.parent_category,
        label: item.parent_category_label,
      }));
  }, [categoriesQuery.data, selectedRoot]);

  const leafListingOptions = useMemo(() => {
    const listing = categoriesQuery.data ?? [];
    return listing.filter(
      (item: AtwCategoryListing) =>
        item.category_root === selectedRoot && item.parent_category === selectedParent,
    );
  }, [categoriesQuery.data, selectedRoot, selectedParent]);

  const selectedCategoryRow = useMemo<AtwCategoryListing | null>(
    () =>
      leafListingOptions.find((item: AtwCategoryListing) => item.category === selectedCategory) ??
      null,
    [leafListingOptions, selectedCategory],
  );

  const selectedSnippetRow = useMemo<AtwSnippetSummary | null>(
    () =>
      selectedCategoryRow?.snippets.find(
        (snippet: AtwSnippetSummary) => snippet.name === selectedSnippet,
      ) ?? null,
    [selectedCategoryRow, selectedSnippet],
  );

  const selectedSnippetFilename = selectedSnippetRow?.name ?? null;
  const schemaQuery = useSnippetPluginSchema(
    selectedSnippetFilename ? snippetPluginSchemaPath(selectedSnippetFilename) : undefined,
  );
  const executionMutation = useSnippetPluginExecution(
    selectedSnippetFilename ? snippetPluginExecutePath(selectedSnippetFilename) : undefined,
  );

  const submitError = executionMutation.isError
    ? (executionMutation.error?.message ?? 'Execution failed')
    : null;

  const handleSubmit = (values: Record<string, unknown>) => {
    if (!selectedSnippetRow) {
      return;
    }

    executionMutation.mutate(buildSnippetExecutionFormPayload(values), {
      onSuccess: () => {
        enqueueSnackbar('Data collection started', { variant: 'success' });
      },
    });
  };

  if (categoriesQuery.isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (categoriesQuery.error) {
    return (
      <Alert severity="error">Failed to load ATW categories: {categoriesQuery.error.message}</Alert>
    );
  }

  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 1 }}>
        Collect Diagnostic Data
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        Select an issue category, then run the matching troubleshooting snippet.
      </Typography>

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} sx={{ mb: 3 }}>
        {rootOptions.length > 1 ? (
          <FormControl fullWidth>
            <InputLabel id="atw-root-label">Category</InputLabel>
            <Select
              labelId="atw-root-label"
              value={selectedRoot}
              label="Category"
              onChange={(event: SelectChangeEvent) => {
                setSelectedRoot(event.target.value);
                setSelectedParent('');
                setSelectedCategory('');
                setSelectedSnippet('');
              }}
            >
              {rootOptions.map((option) => (
                <MenuItem key={option} value={option}>
                  {option}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        ) : null}

        <FormControl fullWidth>
          <InputLabel id="atw-parent-label">Subcategory 1</InputLabel>
          <Select
            labelId="atw-parent-label"
            value={selectedParent}
            label="Subcategory 1"
            disabled={!selectedRoot}
            onChange={(event: SelectChangeEvent) => {
              setSelectedParent(event.target.value);
              setSelectedCategory('');
              setSelectedSnippet('');
            }}
          >
            {parentOptions.map((option) => (
              <MenuItem key={option.value} value={option.value}>
                {option.label}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        <FormControl fullWidth disabled={!selectedRoot || !selectedParent}>
          <InputLabel id="atw-category-label">Subcategory 2</InputLabel>
          <Select
            labelId="atw-category-label"
            value={selectedCategory}
            label="Subcategory 2"
            onChange={(event: SelectChangeEvent) => {
              setSelectedCategory(event.target.value);
              setSelectedSnippet('');
            }}
          >
            {leafListingOptions.map((row: AtwCategoryListing) => (
              <MenuItem key={row.category} value={row.category}>
                {row.category_label}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        <FormControl fullWidth disabled={!selectedRoot || !selectedCategoryRow}>
          <InputLabel id="atw-snippet-label">Snippet</InputLabel>
          <Select
            labelId="atw-snippet-label"
            value={selectedSnippet}
            label="Snippet"
            onChange={(event: SelectChangeEvent) => {
              setSelectedSnippet(event.target.value);
            }}
          >
            {(selectedCategoryRow?.snippets ?? []).map((snippet) => (
              <MenuItem key={snippet.name} value={snippet.name}>
                {snippet.title}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </Stack>

      {selectedSnippetRow && (
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {selectedSnippetRow.description}
        </Typography>
      )}

      {selectedSnippetRow && schemaQuery.isLoading && (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
          <CircularProgress />
        </Box>
      )}

      {selectedSnippetRow && schemaQuery.error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          Failed to load snippet form: {schemaQuery.error.message}
        </Alert>
      )}

      {selectedSnippetRow && schemaQuery.data && (
        <SchemaFormRenderer
          sections={schemaQuery.data.forms ?? []}
          onSubmit={handleSubmit}
          submitLabel="Execute"
          loading={executionMutation.isPending}
          submitError={submitError}
        />
      )}
    </Box>
  );
}
