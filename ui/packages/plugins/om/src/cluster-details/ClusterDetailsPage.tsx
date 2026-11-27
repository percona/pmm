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
import { useParams } from 'react-router-dom';
import LinearProgress from '@mui/material/LinearProgress';
import Stack from '@mui/material/Stack';
import { EmptyState } from '../components/EmptyState';
import { OmError } from '../components/OmError';
import { SnapshotBar } from '../components/SnapshotBar';
import { toClusterRows } from '../topology';
import { useOmTopology } from '../topologyHooks';
import { useOmBase } from '../useOmBase';
import { Messages } from './ClusterDetails.messages';
import { ClusterHeader } from './components/cluster-header';
import { MembersTable } from './components/members-table';

/** One cluster, read out of the same snapshot the Fleet list shows. */
export const ClusterDetailsPage = () => {
  const { clusterId } = useParams();
  const base = useOmBase();
  const { data, isPending, isError, error } = useOmTopology();
  const clusters = useMemo(() => toClusterRows(data), [data]);
  const cluster = clusters.find((row) => row.id === clusterId);

  if (isPending) {
    return <LinearProgress />;
  }

  if (isError) {
    return (
      <OmError
        placement="load"
        title={Messages.load.error}
        messages={(error as Error)?.message}
      />
    );
  }

  if (!cluster) {
    return (
      <EmptyState
        title={Messages.notFound.title}
        action={{ label: Messages.notFound.back, to: base }}
      >
        {Messages.notFound.body}
      </EmptyState>
    );
  }

  return (
    <Stack gap={3} data-testid="om-cluster-details">
      <ClusterHeader cluster={cluster} clusters={clusters} />
      <SnapshotBar envelope={data.snapshot} />
      <MembersTable cluster={cluster} />
    </Stack>
  );
};
