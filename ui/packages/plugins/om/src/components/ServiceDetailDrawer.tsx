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

import type { ReactNode } from 'react';
import { Box, Drawer, IconButton, Stack, Typography } from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { StatusBadge } from './HealthBadge';
import { MemberState } from './MemberState';
import { Duration, Percent } from './Metric';
import { ProbeValue } from './ProbeValue';
import { ServiceLink } from './ServiceLink';
import { Unavailable } from './Unavailable';
import { PROCESS_ROLE_LABEL } from '../constants';
import type { OmEstateStatus } from '../inventory';
import type { OmServiceInventoryRow } from '../types';

/**
 * Everything about one service, without leaving the table.
 *
 * The services table opens with eight of its twenty-four columns. The rest were
 * reachable only through the column chooser, which turns a column on for every
 * row - so the one question the short table cannot answer is "tell me
 * everything about *this* row", and answering it meant re-widening the table the
 * column budget had just narrowed.
 *
 * A drawer rather than an expanding row, for two reasons. It is one of the two
 * things the design review asked for ("a detail drawer or page"), and an
 * expanding row lives inside the table's own horizontal scroll - so a wide
 * panel would inherit the sideways scrolling that the same finding exists to
 * remove. PMM Extensions' `TaskRunDetailDrawer` already does this for task
 * runs; this follows its shape.
 */

const DRAWER_WIDTH = { xs: '100%', sm: 520, md: 640 } as const;

/** Links the drawer's heading to its dialog role. Only one is open at a time. */
const HEADING_ID = 'om-service-detail-heading';

/** One label and its value, on a row that wraps rather than clipping. */
const Field = ({ label, children }: { label: string; children: ReactNode }) => (
  <Stack direction="row" gap={2} sx={{ py: 0.75 }}>
    <Typography
      variant="body2"
      color="text.secondary"
      sx={{ minWidth: 150, flexShrink: 0 }}
    >
      {label}
    </Typography>
    <Box sx={{ minWidth: 0, overflowWrap: 'anywhere' }}>{children}</Box>
  </Stack>
);

const Group = ({ title, children }: { title: string; children: ReactNode }) => (
  <Box>
    <Typography variant="subtitle2" gutterBottom>
      {title}
    </Typography>
    {children}
  </Box>
);

/** A plain string field, or the reason it is absent. */
const Text = ({ value }: { value: string | null | undefined }) =>
  value ? (
    <Typography variant="body2">{value}</Typography>
  ) : (
    <Unavailable reason="service_not_observed" />
  );

export const ServiceDetailDrawer = ({
  row,
  estate,
  onClose,
}: {
  /** The service to describe, or null when nothing is selected. */
  row: OmServiceInventoryRow | null;
  /** What the estate query is doing, so a missing scan row reads correctly. */
  estate: OmEstateStatus;
  onClose: () => void;
}) => (
  <Drawer
    anchor="right"
    open={row !== null}
    onClose={onClose}
    slotProps={{
      paper: {
        sx: { width: DRAWER_WIDTH },
        role: 'dialog',
        'aria-modal': true,
        'aria-labelledby': HEADING_ID,
      },
    }}
    data-testid="om-service-detail-drawer"
  >
    {row && (
      <Stack sx={{ height: '100%' }}>
        <Stack
          direction="row"
          alignItems="center"
          gap={1}
          sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}
        >
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Typography
              variant="h6"
              noWrap
              title={row.service_name}
              id={HEADING_ID}
            >
              {row.service_name}
            </Typography>
          </Box>
          <StatusBadge status={row.status} />
          <IconButton
            size="small"
            onClick={onClose}
            aria-label="Close service details"
          >
            <CloseIcon />
          </IconButton>
        </Stack>

        <Stack sx={{ p: 2, flex: 1, overflowY: 'auto' }} gap={2.5}>
          <Group title="Where it runs">
            <Field label="Node">
              <Text value={row.host} />
            </Field>
            <Field label="Environment">
              <Text value={row.env_name} />
            </Field>
            <Field label="Cluster">
              <Text value={row.cluster_name} />
            </Field>
            <Field label="Replication set">
              <Text value={row.replication_set} />
            </Field>
            <Field label="Endpoint">
              <Text value={row.endpoint} />
            </Field>
          </Group>

          <Group title="What it is">
            <Field label="Member state">
              <MemberState service={row} />
            </Field>
            <Field label="Process">
              <Text
                value={PROCESS_ROLE_LABEL[row.process_role] ?? row.process_role}
              />
            </Field>
            <Field label="Service type">
              <Text value={row.service_type} />
            </Field>
            <Field label="Vendor">
              <Text value={row.vendor} />
            </Field>
            <Field label="Edition">
              <Text value={row.edition} />
            </Field>
            <Field label="Version">
              <Text value={row.version} />
            </Field>
            <Field label="Installed version">
              {/* From the scan rather than the fleet document, so an absent value
                  means something different and `ProbeValue` says which. */}
              <ProbeValue
                inventory={row.inventory}
                value={row.inventory?.installed_version}
                estate={estate}
              />
            </Field>
          </Group>

          <Group title="Load">
            <Field label="CPU">
              <Percent value={row.cpu_usage_percent} />
            </Field>
            <Field label="Connections free">
              <Percent value={row.connections_free_percent} />
            </Field>
            <Field label="Replication lag">
              <Duration value={row.replication_lag_seconds} />
            </Field>
            <Field label="Oplog window">
              <Duration value={row.oplog_window_seconds} />
            </Field>
          </Group>

          <Group title="What a scan found">
            <Field label="Config path">
              <ProbeValue
                inventory={row.inventory}
                value={row.inventory?.config_path}
                estate={estate}
              />
            </Field>
            <Field label="Command line">
              {/* The one field that is a paragraph rather than a value, and the
                  reason the table could never show it: in a cell it is either
                  truncated to uselessness or it sets the table's width. */}
              <ProbeValue
                inventory={row.inventory}
                value={row.inventory?.argv}
                estate={estate}
              />
            </Field>
          </Group>

          <Group title="Identifiers">
            <Field label="Service ID">
              <Text value={row.service_id} />
            </Field>
            <Field label="Node ID">
              <ProbeValue
                inventory={row.inventory}
                value={row.inventory?.node_id}
                estate={estate}
              />
            </Field>
          </Group>

          <Box>
            {/* The design review's last P17 bullet: a detail view links out to
                where the rest of PMM describes the same service. */}
            <ServiceLink serviceName={row.service_name} />
          </Box>
        </Stack>
      </Stack>
    )}
  </Drawer>
);
