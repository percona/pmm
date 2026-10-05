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
import { NOMAD_DOC_URL } from '../constants';

/**
 * What to do about a node Operations cannot run anything on.
 *
 * The design review's F17: "Not onboarded" was a chip with a tooltip naming the
 * state and no way out, so a reader who hit it left the app to work out what to
 * do - exactly when they were stuck.
 *
 * It suggested a dialog carrying the node's `pmm-admin` command. There isn't
 * one. The automation agent is a Nomad client that pmm-agent starts by itself
 * when the server has the feature on; `pmm-admin` can list Nomad agents but has
 * no verb that adds or registers one, and the inventory API's only knob is
 * enabling an agent that already exists. Printing an invented command would be
 * worse than the tooltip it replaced, so this explains the state instead and
 * sends the reader to the right layer.
 *
 * Which layer is the useful part, and it is answerable from data the page
 * already holds:
 *
 * - **PMM Client is not connected.** Nothing downstream can be true, so there is
 *   no point discussing Nomad.
 * - **No node in the fleet is onboarded.** Then it is not this node. Nomad is off
 *   by default server-side, and enabling it needs a public address as well as
 *   the flag - PMM skips starting Nomad silently when the address is missing,
 *   which is a failure shaped exactly like this one.
 * - **This node alone.** Its agent cannot start: the documented client
 *   requirements, or a pmm-agent too old to have one.
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
                pmm-agent is recent enough to carry an automation agent at all.
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
