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

import { describe, expect, it } from 'vitest';
import { tabPanelProps, tabProps } from '../src/tabA11y';

describe('tab / tabpanel pairing', () => {
  it('points each end at the other', () => {
    const tab = tabProps('settings', 'advanced');
    const panel = tabPanelProps('settings', 'advanced');

    expect(panel.role).toBe('tabpanel');
    expect(tab['aria-controls']).toBe(panel.id);
    expect(panel['aria-labelledby']).toBe(tab.id);
  });

  // Two pages each have a tab of the same name in reach of one another (the shell keeps
  // nothing mounted across routes today, but an id is a document-wide claim).
  it('scopes ids by page, so equal tab names do not collide', () => {
    expect(tabProps('fleet', 'services').id).not.toBe(
      tabProps('automations', 'services').id
    );
  });
});
