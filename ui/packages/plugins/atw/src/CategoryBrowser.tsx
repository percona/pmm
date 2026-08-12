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
} from '@mui/material';
import type { SelectChangeEvent } from '@mui/material/Select';
import { useAtwCategories } from './hooks';
import type { AtwCategoryListing, AtwSnippetSummary } from './types';

export interface CategoryBrowserProps {
  /**
   * Called with the snippets of the currently-selected leaf category whenever
   * the selection changes (empty until a leaf category is picked). Memoize this
   * callback so it does not re-fire the reporting effect on every render.
   */
  onSnippetsChange: (snippets: AtwSnippetSummary[]) => void;
}

/**
 * Cascading category browser (root → subcategory 1 → subcategory 2) extracted
 * from the former single-page ATW page. The single-root case auto-selects and
 * hides the top-level Category control; the selected leaf category's snippets
 * are reported through `onSnippetsChange` for a caller-owned snippet picker.
 */
export function CategoryBrowser({ onSnippetsChange }: CategoryBrowserProps) {
  const [selectedRoot, setSelectedRoot] = useState<string>('');
  const [selectedParent, setSelectedParent] = useState<string>('');
  const [selectedCategory, setSelectedCategory] = useState<string>('');

  const categoriesQuery = useAtwCategories();

  const rootOptions = useMemo(() => {
    const listing = categoriesQuery.data ?? [];
    return Array.from(
      new Set(listing.map((item: AtwCategoryListing) => item.category_root))
    );
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
        item.category_root === selectedRoot &&
        item.parent_category === selectedParent
    );
  }, [categoriesQuery.data, selectedRoot, selectedParent]);

  const selectedCategoryRow = useMemo<AtwCategoryListing | null>(
    () =>
      leafListingOptions.find(
        (item: AtwCategoryListing) => item.category === selectedCategory
      ) ?? null,
    [leafListingOptions, selectedCategory]
  );

  const availableSnippets = selectedCategoryRow?.snippets ?? EMPTY_SNIPPETS;

  useEffect(() => {
    onSnippetsChange(availableSnippets);
  }, [availableSnippets, onSnippetsChange]);

  if (categoriesQuery.isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
        <CircularProgress />
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

  return (
    <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
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

      <FormControl fullWidth disabled={!selectedRoot}>
        <InputLabel id="atw-parent-label">Subcategory 1</InputLabel>
        <Select
          labelId="atw-parent-label"
          value={selectedParent}
          label="Subcategory 1"
          onChange={(event: SelectChangeEvent) => {
            setSelectedParent(event.target.value);
            setSelectedCategory('');
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
          }}
        >
          {leafListingOptions.map((row: AtwCategoryListing) => (
            <MenuItem key={row.category} value={row.category}>
              {row.category_label}
            </MenuItem>
          ))}
        </Select>
      </FormControl>
    </Stack>
  );
}

/** Stable empty reference so the reporting effect does not fire on every render. */
const EMPTY_SNIPPETS: AtwSnippetSummary[] = [];
