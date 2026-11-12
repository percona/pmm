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

import { useMemo } from 'react';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import InputAdornment from '@mui/material/InputAdornment';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import CloseIcon from '@mui/icons-material/Close';
import SearchIcon from '@mui/icons-material/Search';
import type { MRT_TableInstance } from 'material-react-table';
import {
  MultiSelectFilter,
  type FilterOption,
} from '../../../components/MultiSelectFilter';
import type { OmClusterRow } from '../../../types';
import {
  HEALTH_OPTIONS,
  TYPE_OPTIONS,
  UNNAMED_ENVIRONMENT_VALUE,
} from '../../FleetClustersTab.constants';
import { Messages } from '../../FleetClustersTab.messages';
import type { FilterColumn } from '../../FleetClustersTab.types';
import {
  countValues,
  pickedValues,
  versionOptions,
  withPicked,
} from '../../FleetClustersTab.utils';
import { getFilterValue, getEnvironments } from './ClusterFilterBar.utils';

/**
 * The name search and the filters, above the table. Driven by the table's state, so
 * they follow the URL; each option counts the clusters it would match under the other
 * filters and the search.
 */
export const ClusterFilterBar = ({
  table,
  rows,
}: {
  table: MRT_TableInstance<OmClusterRow>;
  rows: OmClusterRow[];
}) => {
  const { columnFilters, globalFilter } = table.getState();
  const search: string = globalFilter ?? '';
  const picked = (id: FilterColumn) =>
    pickedValues(columnFilters.find((filter) => filter.id === id)?.value);
  const pick = (id: FilterColumn) => (values: string[]) =>
    table.setColumnFilters((filters) => [
      ...filters.filter((filter) => filter.id !== id),
      ...(values.length ? [{ id, value: values }] : []),
    ]);
  // TanStack's faceted rows leave out the column's own filter, as a count must.
  const counts = (id: FilterColumn) =>
    countValues(
      table
        .getColumn(id)
        .getFacetedRowModel()
        .rows.map((row) => row.original),
      (row) => getFilterValue(id, row)
    );
  const environments = useMemo(
    () =>
      getEnvironments(rows).map((env) => ({
        value: env,
        label:
          env === UNNAMED_ENVIRONMENT_VALUE ? Messages.unnamedEnvironment : env,
      })),
    [rows]
  );
  const versions = useMemo(() => versionOptions(rows), [rows]);
  const filter = (id: FilterColumn, label: string, options: FilterOption[]) => (
    <MultiSelectFilter
      label={label}
      options={withPicked(options, picked(id))}
      value={picked(id)}
      onChange={pick(id)}
      counts={counts(id)}
    />
  );

  return (
    <Paper variant="outlined" sx={{ p: 1.5, width: '100%' }}>
      <Stack direction="row" alignItems="center" gap={1.5} flexWrap="wrap">
        <TextField
          size="small"
          placeholder={`${Messages.search}…`}
          value={search}
          onChange={(event) => table.setGlobalFilter(event.target.value)}
          sx={{ minWidth: 240 }}
          slotProps={{
            htmlInput: { 'aria-label': Messages.search },
            input: {
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon fontSize="small" />
                </InputAdornment>
              ),
              endAdornment: search ? (
                <InputAdornment position="end">
                  <IconButton
                    size="small"
                    edge="end"
                    aria-label={Messages.clearSearch}
                    onClick={() => table.setGlobalFilter('')}
                  >
                    <CloseIcon fontSize="small" />
                  </IconButton>
                </InputAdornment>
              ) : undefined,
            },
          }}
        />
        {filter('health', Messages.filters.health, HEALTH_OPTIONS)}
        {filter('type', Messages.filters.type, TYPE_OPTIONS)}
        {filter('env_name', Messages.filters.environment, environments)}
        {filter('versions', Messages.filters.version, versions)}
        {(columnFilters.length > 0 || !!search) && (
          <Button
            size="small"
            onClick={() => {
              table.resetColumnFilters(true);
              table.setGlobalFilter('');
            }}
          >
            {Messages.filters.clear}
          </Button>
        )}
      </Stack>
    </Paper>
  );
};
