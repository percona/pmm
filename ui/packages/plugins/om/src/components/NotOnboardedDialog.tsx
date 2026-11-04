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

import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Link,
  Stack,
  Typography,
} from '@mui/material';
import { NOMAD_DOC_URL, PMM_AGENT_AUTOMATION_MIN_VERSION } from '../constants';

/**
 * What to do about a node Operations cannot run anything on.
 *
 * There is no command to hand the user: the automation agent is created by the server
 * when pmm-agent registers, and neither `pmm-admin` nor the inventory API can add one.
 * So this diagnoses which layer is at fault instead, which is what a reader cannot work
 * out alone:
 *
 * - PMM Client not connected - nothing downstream can be true.
 * - No node has an agent - the server's feature is off, or its public address is unset.
 * - This node alone - its agent cannot start; version first, then the client requirements.
 */
export const NotOnboardedDialog = ({
  open,
  onClose,
  nodeName,
  pmmAgentConnected,
  anyNodeOnboarded,
}: {
  open: boolean;
  onClose: () => void;
  nodeName: string;
  pmmAgentConnected: boolean;
  anyNodeOnboarded: boolean;
}) => (
  <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
    <DialogTitle>Operations cannot run anything on {nodeName}</DialogTitle>
    <DialogContent>
      <Stack gap={2}>
        <Typography variant="body2">
          PMM knows about this node, but the automation agent that scans it and
          installs MongoDB on it is not registered. Scans and installs are
          unavailable until it is.
        </Typography>
        {!pmmAgentConnected ? (
          <>
            <Typography variant="body2">
              <strong>PMM Client is not connected to this node</strong>, so the
              automation agent cannot run either. Start there:
            </Typography>
            <Typography variant="body2" component="ul" sx={{ pl: 3, m: 0 }}>
              <li>
                Check that pmm-agent is running on the node, and that it can
                reach this server.
              </li>
              <li>
                The node stays listed here while it is disconnected, so this row
                is a node PMM has seen rather than one it is in touch with.
              </li>
            </Typography>
          </>
        ) : anyNodeOnboarded ? (
          <>
            <Typography variant="body2">
              PMM Client is connected and other nodes do have an agent, so this
              is about this node. Its agent is failing to start. What to check,
              in the order worth checking it:
            </Typography>
            <Typography variant="body2" component="ul" sx={{ pl: 3, m: 0 }}>
              <li>
                The <code>iproute</code> package is installed on the node.
              </li>
              <li>cgroup access is available to the agent.</li>
              <li>
                pmm-agent is {PMM_AGENT_AUTOMATION_MIN_VERSION} or newer. An
                older one is given no automation agent at all, so this is the
                check worth doing first on a node that has never had one.
              </li>
            </Typography>
          </>
        ) : (
          <>
            <Typography variant="body2">
              <strong>No node here has an agent</strong>, which points at the
              server rather than at this node. Automation is off by default and
              needs two things, not one:
            </Typography>
            <Typography variant="body2" component="ul" sx={{ pl: 3, m: 0 }}>
              <li>the feature enabled on PMM Server, and</li>
              <li>
                a public address set for it. With the flag on and no address,
                PMM skips starting the automation server without reporting an
                error - which looks precisely like this.
              </li>
            </Typography>
            <Typography variant="body2">
              Both are server settings and take a restart, so this is not
              something to fix per node.
            </Typography>
          </>
        )}
        <Typography variant="body2">
          <Link href={NOMAD_DOC_URL} target="_blank" rel="noopener">
            Requirements and setup in the documentation
          </Link>
        </Typography>
      </Stack>
    </DialogContent>
    <DialogActions>
      <Button onClick={onClose}>Close</Button>
    </DialogActions>
  </Dialog>
);
