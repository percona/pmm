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

import { Chip, Link, Stack, Tooltip, Typography } from '@mui/material';
import { Link as RouterLink } from 'react-router-dom';
import { OM_ROUTE_NODES } from '../constants';
import type { OmInventoryHost } from '../types';

/**
 * Whether one selected node can be installed onto, and why not.
 *
 * The whole point of P1: every one of these conditions used to be checked on the
 * wizard's final button, inside TriggerHostBootstrap, after the form was filled in.
 * They are the same `automation_blocked_reasons` the Nodes page shows, read here so
 * the two surfaces cannot disagree about why a node is refused.
 *
 * Blocked rows link to that node on the Nodes page, which is where its newest scan
 * is - the same destination NodeNamesLinked uses for the trigger's own refusal, so
 * following a reason lands in one place however the reader got to it.
 */
export const HostReadiness = ({
  host,
  omBase,
}: {
  host: OmInventoryHost;
  omBase: string;
}) => {
  if (host.automation_eligible) {
    return (
      <Chip size="small" color="success" variant="outlined" label="Ready" />
    );
  }
  return (
    <Stack spacing={0.5}>
      <Tooltip title={host.automation_blocked_reasons.join('; ')}>
        <Chip
          size="small"
          // Neutral when nothing is wrong with the node, amber when something is.
          // A healthy replica-set member painted as needing attention sends the
          // reader hunting for a fault on a machine that is working perfectly.
          color={host.automation_blocked_by_design ? 'default' : 'warning'}
          variant={host.automation_blocked_by_design ? 'outlined' : 'filled'}
          label={host.automation_blocked_by_design ? 'Not a target' : 'Blocked'}
          sx={{ alignSelf: 'flex-start', cursor: 'help' }}
        />
      </Tooltip>
      <Typography variant="caption" color="text.secondary">
        {host.automation_blocked_reasons.join('; ')}
      </Typography>
      {/* Only for a fault: there is no scan to go and read about a node that is
          working exactly as intended. */}
      {!host.automation_blocked_by_design && (
        <Link
          component={RouterLink}
          variant="caption"
          to={`${omBase}/${OM_ROUTE_NODES}?node=${encodeURIComponent(host.name)}`}
        >
          See this node and its latest scan
        </Link>
      )}
    </Stack>
  );
};
