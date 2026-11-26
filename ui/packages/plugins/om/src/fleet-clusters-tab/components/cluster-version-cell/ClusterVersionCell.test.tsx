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
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import type { OmClusterRow } from '../../../types';
import { ClusterVersionCell } from './ClusterVersionCell';

const renderCell = (cluster: Partial<OmClusterRow>) =>
  render(
    <ClusterVersionCell
      cluster={
        { version_drift: false, version_sets: [], ...cluster } as OmClusterRow
      }
    />
  );

/** jsdom has no :focus-visible, so MUI cannot open a tooltip on focus here: check the reach. */
const tabTo = async (text: string) => {
  await userEvent.tab();
  const trigger = screen.getByText(text);
  expect(trigger).toHaveFocus();
  return trigger;
};

describe('ClusterVersionCell', () => {
  it('lets the keyboard reach the drift details', async () => {
    renderCell({
      versions: ['7.0.9-1', '7.0.24-13'],
      version_drift: true,
      version_sets: [
        { replication_set: 'rs0', versions: ['7.0.9-1', '7.0.24-13'] },
      ] as OmClusterRow['version_sets'],
    });

    await tabTo('Mixed');
  });

  it('lets the keyboard reach the list of versions', async () => {
    renderCell({ versions: ['7.0.24-13', '8.0.4-1'] });

    expect(await tabTo('2 versions')).toHaveAccessibleName(
      '7.0.24-13, 8.0.4-1'
    );
  });
});
