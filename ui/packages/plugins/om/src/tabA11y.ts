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

/**
 * The ARIA pairing between a tab and the panel it shows. MUI's `Tab` renders
 * `role="tab"` and `aria-selected`, but neither end knows the other's id, so a screen
 * reader cannot get from the tab to its content. Every page here renders one panel
 * for the selected tab, so the panel takes the selected tab's ids.
 */
export const tabProps = (page: string, tab: string) => ({
  id: `om-${page}-tab-${tab}`,
  'aria-controls': `om-${page}-panel-${tab}`,
});

export const tabPanelProps = (page: string, tab: string) => ({
  role: 'tabpanel',
  id: `om-${page}-panel-${tab}`,
  'aria-labelledby': `om-${page}-tab-${tab}`,
});
