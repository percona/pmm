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
  Button,
  Chip,
  CircularProgress,
  Stack,
  Typography,
  type ChipOwnProps,
} from '@mui/material';
import { useAtwCategories } from './hooks';
import type { AtwCategoryListing, AtwSnippetSummary } from './types';

/**
 * Chip color for selected vs idle. Asserted: atw sees two MUI Chip color
 * unions, so a plain ternary fails tsc even though both literals are valid.
 */
function filterChipColor(selected: boolean): ChipOwnProps['color'] {
  return (selected ? 'primary' : 'default') as ChipOwnProps['color'];
}

export interface CategoryFiltersProps {
  /**
   * Called with the snippets matching the current optional filters whenever
   * the selection changes (empty while no filter is active). Memoize this
   * callback so it does not re-fire the reporting effect on every render.
   */
  onSnippetsChange: (snippets: AtwSnippetSummary[]) => void;
}

interface RootFilterOption {
  root: string;
  count: number;
}

interface ParentFilterOption {
  value: string;
  label: string;
  count: number;
}

/**
 * Unique snippet filenames per category root from the listing response.
 * Counts match what the Collect picker shows after leaf-union dedupe — the same
 * script can sit on several leaves under one root, so summing `snippet_count`
 * would overstate the chip. Roots absent from the response stay omitted.
 */
export function rootFilterOptions(
  listing: readonly AtwCategoryListing[]
): RootFilterOption[] {
  const namesByRoot = new Map<string, Set<string>>();
  for (const item of listing) {
    let names = namesByRoot.get(item.category_root);
    if (!names) {
      names = new Set();
      namesByRoot.set(item.category_root, names);
    }
    for (const snippet of item.snippets) {
      names.add(snippet.name);
    }
  }
  return [...namesByRoot.entries()].map(([root, names]) => ({
    root,
    count: names.size,
  }));
}

/**
 * Unique snippet filenames per problem area under the selected root.
 */
export function parentFilterOptions(
  listing: readonly AtwCategoryListing[],
  root: string
): ParentFilterOption[] {
  const byParent = new Map<string, { label: string; names: Set<string> }>();
  for (const item of listing) {
    if (item.category_root !== root) {
      continue;
    }
    let entry = byParent.get(item.parent_category);
    if (!entry) {
      entry = {
        label: item.parent_category_label,
        names: new Set(),
      };
      byParent.set(item.parent_category, entry);
    }
    for (const snippet of item.snippets) {
      entry.names.add(snippet.name);
    }
  }
  return [...byParent.entries()].map(([value, entry]) => ({
    value,
    label: entry.label,
    count: entry.names.size,
  }));
}

/**
 * Union leaf-category snippets matching the optional root / parent filters,
 * deduped by filename.
 */
export function snippetsForFilters(
  listing: readonly AtwCategoryListing[],
  root: string,
  parent: string
): AtwSnippetSummary[] {
  if (root === '') {
    return EMPTY_SNIPPETS;
  }
  const byName = new Map<string, AtwSnippetSummary>();
  for (const item of listing) {
    if (item.category_root !== root) {
      continue;
    }
    if (parent !== '' && item.parent_category !== parent) {
      continue;
    }
    for (const snippet of item.snippets) {
      byName.set(snippet.name, snippet);
    }
  }
  if (byName.size === 0) {
    return EMPTY_SNIPPETS;
  }
  return [...byName.values()];
}

/**
 * Optional taxonomy filters for the Collect pane: database and problem-area
 * chips with counts from the category listing. Filters are never required —
 * search alone reaches every approved script. The taxonomy is drawn only from
 * the API response so empty sidecar roots never appear as dead chips.
 */
export function CategoryFilters({ onSnippetsChange }: CategoryFiltersProps) {
  const [selectedRoot, setSelectedRoot] = useState('');
  const [selectedParent, setSelectedParent] = useState('');

  const categoriesQuery = useAtwCategories();
  const listing = categoriesQuery.data ?? EMPTY_LISTING;

  const roots = useMemo(() => rootFilterOptions(listing), [listing]);
  const parents = useMemo(
    () =>
      selectedRoot === ''
        ? EMPTY_PARENTS
        : parentFilterOptions(listing, selectedRoot),
    [listing, selectedRoot]
  );

  const availableSnippets = useMemo(
    () => snippetsForFilters(listing, selectedRoot, selectedParent),
    [listing, selectedRoot, selectedParent]
  );

  useEffect(() => {
    onSnippetsChange(availableSnippets);
  }, [availableSnippets, onSnippetsChange]);

  // A refetch can drop roots/parents that were selected. Clear them so we do not
  // keep an invisible filter with no chips and no Clear control.
  useEffect(() => {
    if (selectedRoot === '') {
      return;
    }
    if (!roots.some((option) => option.root === selectedRoot)) {
      setSelectedRoot('');
      setSelectedParent('');
      return;
    }
    if (
      selectedParent !== '' &&
      !parents.some((option) => option.value === selectedParent)
    ) {
      setSelectedParent('');
    }
  }, [roots, parents, selectedRoot, selectedParent]);

  if (categoriesQuery.isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
        <CircularProgress size={24} />
      </Box>
    );
  }

  if (categoriesQuery.error) {
    return (
      <Alert severity="error">
        Failed to load ATW categories: {categoriesQuery.error.message}
      </Alert>
    );
  }

  if (roots.length === 0) {
    return null;
  }

  const hasFilter = selectedRoot !== '';

  return (
    <Stack spacing={1.5}>
      <Stack
        direction="row"
        alignItems="center"
        justifyContent="space-between"
        spacing={1}
      >
        <Typography variant="subtitle2" color="text.secondary">
          Filter by category
        </Typography>
        {hasFilter ? (
          <Button
            size="small"
            onClick={() => {
              setSelectedRoot('');
              setSelectedParent('');
            }}
          >
            Clear filters
          </Button>
        ) : null}
      </Stack>

      <Box>
        <Typography variant="caption" color="text.secondary" display="block">
          Database
        </Typography>
        <Stack
          direction="row"
          flexWrap="wrap"
          useFlexGap
          spacing={1}
          sx={{ mt: 0.5 }}
        >
          {roots.map((option) => {
            const selected = selectedRoot === option.root;
            return (
              <Chip
                key={option.root}
                label={`${option.root} (${option.count})`}
                size="small"
                clickable
                color={filterChipColor(selected)}
                variant={selected ? 'filled' : 'outlined'}
                aria-pressed={selected}
                onClick={() => {
                  if (selected) {
                    setSelectedRoot('');
                    setSelectedParent('');
                    return;
                  }
                  setSelectedRoot(option.root);
                  setSelectedParent('');
                }}
              />
            );
          })}
        </Stack>
      </Box>

      {selectedRoot !== '' && parents.length > 0 ? (
        <Box>
          <Typography variant="caption" color="text.secondary" display="block">
            Problem area
          </Typography>
          <Stack
            direction="row"
            flexWrap="wrap"
            useFlexGap
            spacing={1}
            sx={{ mt: 0.5 }}
          >
            {parents.map((option) => {
              const selected = selectedParent === option.value;
              return (
                <Chip
                  key={option.value}
                  label={`${option.label} (${option.count})`}
                  size="small"
                  clickable
                  color={filterChipColor(selected)}
                  variant={selected ? 'filled' : 'outlined'}
                  aria-pressed={selected}
                  onClick={() => {
                    setSelectedParent(selected ? '' : option.value);
                  }}
                />
              );
            })}
          </Stack>
        </Box>
      ) : null}
    </Stack>
  );
}

/** Stable empty listing so memos stay referentially quiet while loading finishes. */
const EMPTY_LISTING: AtwCategoryListing[] = [];

/** Stable empty parent row while no root is selected. */
const EMPTY_PARENTS: ParentFilterOption[] = [];

/** Stable empty reference so the reporting effect does not fire on every render. */
const EMPTY_SNIPPETS: AtwSnippetSummary[] = [];
