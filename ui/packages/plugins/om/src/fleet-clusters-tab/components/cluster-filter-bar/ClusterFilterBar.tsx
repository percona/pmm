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
import Typography from '@mui/material/Typography';
import type { MRT_TableInstance } from 'material-react-table';
import { MultiSelectFilter } from '../../../components/MultiSelectFilter';
import type { OmClusterRow } from '../../../types';
import {
  HEALTH_OPTIONS,
  TYPE_OPTIONS,
  UNNAMED_ENVIRONMENT_VALUE,
} from '../../FleetClustersTab.constants';
import { Messages } from '../../FleetClustersTab.messages';
import type { FilterColumn } from '../../FleetClustersTab.types';
import { pickedValues, withPicked } from '../../FleetClustersTab.utils';

/**
 * The filters as toolbar selects rather than header inputs, since Type is hidden and
 * Environment's column leaves the table while grouped by it. Driven by the table's state.
 */
export const ClusterFilterBar = ({
  table,
  rows,
}: {
  table: MRT_TableInstance<OmClusterRow>;
  rows: OmClusterRow[];
}) => {
  const { columnFilters, globalFilter } = table.getState();
  const picked = (id: FilterColumn) =>
    pickedValues(columnFilters.find((filter) => filter.id === id)?.value);
  const pick = (id: FilterColumn) => (values: string[]) =>
    table.setColumnFilters((filters) => [
      ...filters.filter((filter) => filter.id !== id),
      ...(values.length ? [{ id, value: values }] : []),
    ]);
  const environments = useMemo(
    () =>
      [...new Set(rows.map((row) => row.env_name || UNNAMED_ENVIRONMENT_VALUE))]
        .sort((a, b) => a.localeCompare(b))
        .map((value) => ({
          value,
          label:
            value === UNNAMED_ENVIRONMENT_VALUE
              ? Messages.unnamedEnvironment
              : value,
        })),
    [rows]
  );
  const filtered = columnFilters.length > 0 || !!globalFilter;
  return (
    <>
      <MultiSelectFilter
        label={Messages.filters.health}
        options={withPicked(HEALTH_OPTIONS, picked('health'))}
        value={picked('health')}
        onChange={pick('health')}
      />
      <MultiSelectFilter
        label={Messages.filters.type}
        options={withPicked(TYPE_OPTIONS, picked('type'))}
        value={picked('type')}
        onChange={pick('type')}
      />
      <MultiSelectFilter
        label={Messages.filters.environment}
        options={withPicked(environments, picked('env_name'))}
        value={picked('env_name')}
        onChange={pick('env_name')}
      />
      {filtered && (
        <>
          <Typography
            variant="body2"
            color="text.secondary"
            data-testid="om-cluster-match-count"
          >
            {Messages.filters.matchCount(
              table.getFilteredRowModel().rows.length,
              rows.length
            )}
          </Typography>
          <Button
            size="small"
            onClick={() => {
              table.resetColumnFilters(true);
              table.setGlobalFilter('');
            }}
          >
            {Messages.filters.clear}
          </Button>
        </>
      )}
    </>
  );
};
