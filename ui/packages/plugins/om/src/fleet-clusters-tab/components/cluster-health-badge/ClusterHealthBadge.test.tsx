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
import { ThemeContextProvider, pmmThemeOptions } from '@percona/peak-ui';
import { describe, expect, it } from 'vitest';
import type { OmClusterHealth } from '../../../types';
import { ClusterHealthBadge } from './ClusterHealthBadge';

describe('ClusterHealthBadge', () => {
  it.each([
    ['healthy', 'Healthy'],
    ['degraded', 'Degraded'],
    ['down', 'Down'],
    ['unknown', 'Unknown'],
  ] as const)(
    'reads %s as "%s" with the status icon used across the tab',
    (health, label) => {
      render(
        <ThemeContextProvider themeOptions={pmmThemeOptions}>
          <ClusterHealthBadge health={health as OmClusterHealth} />
        </ThemeContextProvider>
      );

      const chip = screen.getByTestId('om-cluster-health');
      expect(chip).toHaveTextContent(label);
      expect(chip).toHaveClass('MuiChip-outlined');
      expect(
        chip.querySelector(`[data-testid="om-cluster-health-icon-${health}"]`)
      ).not.toBeNull();
    }
  );
});
