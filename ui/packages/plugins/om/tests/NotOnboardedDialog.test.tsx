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

import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { NotOnboardedDialog } from '../src/components/NotOnboardedDialog';
import { NOMAD_DOC_URL } from '../src/constants';

const renderDialog = (
  props: Partial<React.ComponentProps<typeof NotOnboardedDialog>> = {}
) =>
  render(
    <NotOnboardedDialog
      open
      onClose={vi.fn()}
      nodeName="node00"
      pmmAgentConnected
      anyNodeOnboarded
      {...props}
    />
  );

describe('NotOnboardedDialog', () => {
  it('names the node it is about', () => {
    renderDialog();

    expect(
      screen.getByText(/Operations cannot run anything on node00/)
    ).toBeInTheDocument();
  });

  // The whole point of F17: the state used to be a dead end, so whichever branch
  // a reader lands in has to end somewhere they can go.
  it('always offers the documentation', () => {
    renderDialog();

    expect(screen.getByRole('link')).toHaveAttribute('href', NOMAD_DOC_URL);
  });

  // Nothing downstream can be true without the client, so Nomad is not worth
  // discussing yet.
  it('sends a disconnected node to PMM Client first', () => {
    renderDialog({ pmmAgentConnected: false, anyNodeOnboarded: true });

    expect(screen.getByText(/PMM Client is not connected/)).toBeInTheDocument();
    expect(screen.queryByText(/iproute/)).toBeNull();
  });

  // One node broken among working ones is a node problem, and the two documented
  // client requirements are what to check.
  it('points at the node when other nodes do have an agent', () => {
    renderDialog({ pmmAgentConnected: true, anyNodeOnboarded: true });

    expect(
      screen.getByText(/Its agent is failing to start/)
    ).toBeInTheDocument();
    expect(screen.getByText(/iproute/)).toBeInTheDocument();
    expect(screen.queryByText(/No node here has an agent/)).toBeNull();
  });

  // Every node broken is a server problem, and saying "check iproute on this
  // node" would send the reader to the wrong machine entirely.
  it('points at the server when no node has an agent', () => {
    renderDialog({ pmmAgentConnected: true, anyNodeOnboarded: false });

    expect(screen.getByText(/No node here has an agent/)).toBeInTheDocument();
    expect(screen.getByText(/public address/)).toBeInTheDocument();
    expect(screen.queryByText(/iproute/)).toBeNull();
  });

  // The review asked for "the exact pmm-admin command for that host". There is
  // no such command - pmm-admin can list automation agents but has no verb that
  // adds one - so this guards against a future edit helpfully inventing one.
  it('never offers a command that does not exist', () => {
    for (const props of [
      { pmmAgentConnected: false, anyNodeOnboarded: true },
      { pmmAgentConnected: true, anyNodeOnboarded: true },
      { pmmAgentConnected: true, anyNodeOnboarded: false },
    ]) {
      const { unmount } = renderDialog(props);
      expect(screen.queryByText(/pmm-admin/)).toBeNull();
      unmount();
    }
  });
});
