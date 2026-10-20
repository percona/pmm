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

import { useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Link as MuiLink,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import { Table } from '@percona/peak-ui';
import type { MRT_ColumnDef } from 'material-react-table';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '@pmm-extensions/api';
import { formatTimestamp } from '@pmm-extensions/framework';
import {
  ATW_INCIDENT_LIST_LIMIT,
  useAtwConfig,
  useAtwIncidentLifecycle,
  useAtwIncidents,
  useCreateAtwIncident,
} from './hooks';
import {
  DeleteIncidentDialog,
  IncidentActionsMenu,
  IncidentStatusChip,
  isIncidentClosed,
  RenameIncidentDialog,
} from './IncidentActions';
import { IncidentsEmptyState } from './IncidentsEmptyState';
import { SendUnavailableNotice } from './SendUnavailableNotice';
import type { AtwIncident } from './types';

/**
 * The "Not collected" column's heading and its explanation. The count is the
 * side-car's `failed_run_count`, whose set is failed, lost, stale and
 * unlaunchable — runs that produced nothing, not only runs whose script
 * errored — so the heading names the outcome rather than calling all of them
 * failures. Recorded on PMM-15514.
 */
export const NOT_COLLECTED_HEADER = 'Not collected';
export const NOT_COLLECTED_DESCRIPTION =
  'Runs that produced no data: failed, lost, stale or could not launch.';

/**
 * When the incident last changed. The side-car never serves it null for an
 * incident it computes it for, but the field is optional on the wire, so an
 * older side-car falls back to the incident's own timestamps.
 */
function lastActivity(incident: AtwIncident): string {
  return (
    incident.last_activity_at ?? incident.updated_at ?? incident.created_at
  );
}

function timeValue(value: string): number {
  const time = new Date(value).getTime();
  return Number.isNaN(time) ? 0 : time;
}

/**
 * Landing page rendered at ``/atw``: the incident list, as a sortable,
 * filterable table that answers which incident needs attention without opening
 * one — its state, how many runs it holds and how many collected nothing, and
 * when it last moved. Creating an incident is one click: the server names it
 * with a timestamp and the workspace lets the user rename it in place.
 */
export function IncidentListPage() {
  const { canMutate } = useAuth();
  const navigate = useNavigate();
  const { data, isLoading, error } = useAtwIncidents({
    offset: 0,
    limit: ATW_INCIDENT_LIST_LIMIT,
  });
  const { data: config } = useAtwConfig();
  const incidents = useMemo(() => data?.items ?? [], [data]);
  const isEmpty = data?.total === 0;
  const showIntro = isLoading || !isEmpty || Boolean(error);
  const sendDisabledReasons = config?.send_disabled_reasons ?? [];
  const createMutation = useCreateAtwIncident();
  const lifecycle = useAtwIncidentLifecycle();

  const [renameTarget, setRenameTarget] = useState<AtwIncident | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<AtwIncident | null>(null);

  const handleCreate = () => {
    createMutation.mutate(
      {},
      { onSuccess: (incident) => navigate(incident.id) }
    );
  };

  const columns = useMemo<MRT_ColumnDef<AtwIncident>[]>(
    () => [
      {
        id: 'name',
        header: 'Name',
        accessorKey: 'name',
        size: 220,
        Cell: ({ row }) => (
          <MuiLink
            component={Link}
            to={row.original.id}
            onClick={(event) => event.stopPropagation()}
          >
            {row.original.name}
          </MuiLink>
        ),
      },
      {
        id: 'case_ref',
        header: 'Case',
        accessorFn: (row) => row.case_ref ?? '',
        size: 120,
        Cell: ({ row }) => row.original.case_ref || '—',
      },
      {
        id: 'status',
        header: 'Status',
        accessorFn: (row) => (isIncidentClosed(row) ? 'Closed' : 'Open'),
        size: 110,
        filterVariant: 'select',
        filterSelectOptions: ['Open', 'Closed'],
        Cell: ({ row }) => <IncidentStatusChip incident={row.original} />,
      },
      {
        id: 'run_count',
        header: 'Runs',
        accessorKey: 'run_count',
        size: 90,
        enableColumnFilter: false,
      },
      {
        id: 'failed_run_count',
        header: NOT_COLLECTED_HEADER,
        accessorKey: 'failed_run_count',
        size: 140,
        enableColumnFilter: false,
        Header: () => (
          <Tooltip title={NOT_COLLECTED_DESCRIPTION}>
            <span>{NOT_COLLECTED_HEADER}</span>
          </Tooltip>
        ),
        Cell: ({ row }) => {
          const count = row.original.failed_run_count;
          return (
            <Typography
              variant="inherit"
              component="span"
              color={count > 0 ? 'error.main' : undefined}
              fontWeight={count > 0 ? 'medium' : undefined}
            >
              {count}
            </Typography>
          );
        },
      },
      {
        id: 'last_activity',
        header: 'Last activity',
        accessorFn: (row) => timeValue(lastActivity(row)),
        sortingFn: 'basic',
        size: 150,
        enableColumnFilter: false,
        enableGlobalFilter: false,
        Cell: ({ row }) => {
          const formatted = formatTimestamp(lastActivity(row.original));
          return (
            <Box component="span" title={formatted?.title}>
              {formatted?.display ?? '—'}
            </Box>
          );
        },
      },
      {
        id: 'created_by',
        header: 'Created by',
        accessorKey: 'created_by',
        size: 130,
      },
    ],
    []
  );

  return (
    <Box>
      <Stack
        direction="row"
        alignItems="center"
        justifyContent="space-between"
        sx={{ mb: 1 }}
      >
        <Typography variant="h4">Support diagnostics</Typography>
        {/*
          Withheld while the list is unavailable: a create would hit the same
          backend that just failed, so offering it only produces a second error
          on top of one the user cannot act on. Also withheld when empty — the
          primary CTA lives inside the empty state instead (PMM-15515).
        */}
        {!error && canMutate && (isLoading || !isEmpty) && (
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            disabled={isLoading}
            loading={createMutation.isPending}
            onClick={handleCreate}
          >
            New incident
          </Button>
        )}
      </Stack>
      {showIntro && (
        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
          Open an incident to run diagnostic snippets and review their results
          in one place.
        </Typography>
      )}

      {createMutation.isError && (
        <Alert
          severity="error"
          sx={{ mb: 2 }}
          onClose={() => createMutation.reset()}
        >
          {createMutation.error?.message ?? 'Failed to create incident'}
        </Alert>
      )}

      {lifecycle.error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={lifecycle.reset}>
          {lifecycle.error}
        </Alert>
      )}

      {isLoading && (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
          <CircularProgress />
        </Box>
      )}

      {error && (
        <Alert severity="error">
          Failed to load incidents: {error.message}
        </Alert>
      )}

      {!isLoading && !error && canMutate && sendDisabledReasons.length > 0 && (
        <SendUnavailableNotice />
      )}

      {!isLoading && !error && isEmpty && (
        <IncidentsEmptyState
          onCreate={canMutate ? handleCreate : undefined}
          creating={createMutation.isPending}
        />
      )}

      {data && data.total > incidents.length && (
        <Alert severity="info" sx={{ mb: 2 }} role="status">
          Showing the {incidents.length} most recently created of {data.total}{' '}
          incidents.
        </Alert>
      )}

      {incidents.length > 0 && (
        <Table
          tableName="atw-incidents"
          columns={columns}
          data={incidents}
          getRowId={(row) => row.id}
          initialState={{
            density: 'compact',
            showGlobalFilter: true,
            pagination: { pageIndex: 0, pageSize: 20 },
            sorting: [{ id: 'last_activity', desc: true }],
          }}
          enableHiding={false}
          enableColumnActions={false}
          enableRowActions={canMutate}
          renderRowActions={({ row }) => (
            <IncidentActionsMenu
              incident={row.original}
              lifecycle={lifecycle}
              size="small"
              onRename={setRenameTarget}
              onDelete={setDeleteTarget}
            />
          )}
          displayColumnDefOptions={{
            'mrt-row-actions': { header: '', size: 120 },
          }}
          emptyFilterResultsMessage="No incident matches these filters."
          // Peak's own row click: it owns the row's onClick and cursor.
          enableRowHoverAction
          rowHoverAction={(row) => navigate(row.original.id)}
        />
      )}

      <RenameIncidentDialog
        incident={renameTarget}
        onClose={() => setRenameTarget(null)}
      />
      <DeleteIncidentDialog
        incident={deleteTarget}
        onClose={() => setDeleteTarget(null)}
      />
    </Box>
  );
}
