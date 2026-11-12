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
import LinearProgress from '@mui/material/LinearProgress';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import { EmptyState } from '../components/EmptyState';
import { OmError } from '../components/OmError';
import { SnapshotBar } from '../components/SnapshotBar';
import { FLEET_NOT_COLLECTED, OPERATIONS_DOC_URL } from '../constants';
import { useOmInventoryServices } from '../inventoryHooks';
import { toClusterRows } from '../topology';
import { useOmTopology } from '../topologyHooks';
import { ClustersTable } from './components/clusters-table';
import { FleetCounts } from './components/fleet-counts';
import { Messages } from './FleetClustersTab.messages';

export const FleetClustersTab = () => {
  const { data, isPending, isError, error } = useOmTopology();
  const { isError: inventoryFailed, error: inventoryError } =
    useOmInventoryServices();
  const rows = useMemo(() => toClusterRows(data), [data]);

  if (isPending) {
    return <LinearProgress />;
  }

  if (isError) {
    // FleetPage renders the header and Sync, the way out of the first-run 503, above this.
    return (
      <OmError
        placement="load"
        title={Messages.load.error}
        messages={(error as Error)?.message}
      />
    );
  }

  return (
    <Stack gap={3}>
      <Stack gap={1}>
        <SnapshotBar envelope={data.snapshot} />
        {inventoryFailed && (
          <OmError
            placement="load"
            severity="warning"
            title={Messages.load.scanError}
            messages={[
              Messages.load.scanErrorDetail,
              inventoryError instanceof Error ? inventoryError.message : null,
            ]}
          />
        )}
        <FleetCounts summary={data.summary} />
      </Stack>

      {rows.length === 0 ? (
        <EmptyState title={Messages.empty.title}>
          {Messages.empty.intro}{' '}
          {data.snapshot.generated_at
            ? Messages.empty.noServices
            : FLEET_NOT_COLLECTED}{' '}
          <Link href={OPERATIONS_DOC_URL} target="_blank" rel="noopener">
            {Messages.empty.docLink}
          </Link>
        </EmptyState>
      ) : (
        <ClustersTable rows={rows} />
      )}
    </Stack>
  );
};
