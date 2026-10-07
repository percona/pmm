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
import { describe, expect, it } from 'vitest';
import { ClusterHealthBadge, StatusBadge } from '../src/components/HealthBadge';
import type { OmClusterHealth, OmServiceStatus } from '../src/types';

describe('StatusBadge', () => {
  const hoursAgo = (hours: number) =>
    new Date(Date.now() - hours * 3600 * 1000).toISOString();

  it('says how long a down service has been down, from when it was last up', () => {
    render(<StatusBadge status="SERVICE_STATUS_DOWN" lastUpAt={hoursAgo(3)} />);

    expect(screen.getByTestId('om-down-for')).toHaveTextContent(
      /^for 3h( \d+s)?$/
    );
  });

  it('says a down service was not up at all in the window it can see', () => {
    render(<StatusBadge status="SERVICE_STATUS_DOWN" lastUpAt={null} />);

    expect(screen.getByTestId('om-down-for')).toHaveTextContent(
      'not up in 24 hours'
    );
  });

  it('says nothing about time for a service that is up', () => {
    render(<StatusBadge status="SERVICE_STATUS_UP" lastUpAt={hoursAgo(3)} />);

    expect(screen.queryByTestId('om-down-for')).toBeNull();
  });

  it.each([
    ['SERVICE_STATUS_UP', 'Up', 'CheckCircleOutlineIcon'],
    ['SERVICE_STATUS_DOWN', 'Down', 'ErrorOutlineIcon'],
    ['SERVICE_STATUS_UNSPECIFIED', 'Unknown', 'HelpOutlineIcon'],
  ] as const)(
    'reads %s by word and icon, not colour alone',
    (status, label, icon) => {
      render(<StatusBadge status={status as OmServiceStatus} />);

      const chip = screen.getByTestId('om-service-status');
      expect(chip).toHaveTextContent(label);
      expect(chip.querySelector(`[data-testid="${icon}"]`)).not.toBeNull();
    }
  );

  it('reads a status it does not know as unknown, icon and word alike', () => {
    render(<StatusBadge status={'SOMETHING_NEW' as OmServiceStatus} />);

    const chip = screen.getByTestId('om-service-status');
    expect(chip).toHaveTextContent('Unknown');
    expect(
      chip.querySelector('[data-testid="HelpOutlineIcon"]')
    ).not.toBeNull();
  });

  it('gives Down the filled chip and Up the outlined one', () => {
    render(
      <>
        <StatusBadge status="SERVICE_STATUS_DOWN" />
        <StatusBadge status="SERVICE_STATUS_UP" />
      </>
    );

    const [down, up] = screen.getAllByTestId('om-service-status');
    expect(down).toHaveClass('MuiChip-filled');
    expect(up).toHaveClass('MuiChip-outlined');
  });
});

describe('ClusterHealthBadge', () => {
  it.each([
    ['healthy', 'Healthy', 'CheckCircleOutlineIcon', 'MuiChip-outlined'],
    ['degraded', 'Degraded', 'WarningAmberIcon', 'MuiChip-filled'],
    ['down', 'Down', 'ErrorOutlineIcon', 'MuiChip-filled'],
    ['unknown', 'Unknown', 'HelpOutlineIcon', 'MuiChip-outlined'],
  ] as const)(
    'reads %s as "%s" with its own icon',
    (health, label, icon, variant) => {
      render(<ClusterHealthBadge health={health as OmClusterHealth} />);

      const chip = screen.getByTestId('om-cluster-health');
      expect(chip).toHaveTextContent(label);
      expect(chip).toHaveClass(variant);
      expect(chip.querySelector(`[data-testid="${icon}"]`)).not.toBeNull();
    }
  );
});
