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

import { useSearchParams } from 'react-router-dom';
import { Box, Tab, Tabs, Typography } from '@mui/material';
import { OmHeader } from './components/OmHeader';
import { SyncButton } from './components/SyncButton';
import { tabPanelProps, tabProps } from './tabA11y';
import { FleetClustersTab } from './FleetClustersTab';
import { FleetServicesTab } from './FleetServicesTab';

/** The tabs, and the query-parameter values that address them. */
const TABS = ['clusters', 'services'] as const;

type TabId = (typeof TABS)[number];

/**
 * What a reader is looking at, per tab. Two readings of one snapshot, so the
 * difference is worth saying out loud rather than leaving them to infer it from
 * the columns.
 */
const SUBTITLE: Record<TabId, string> = {
  clusters:
    'Every monitored MongoDB cluster and the state of its members. Unfold one to see them.',
  services:
    'The same fleet as one row per MongoDB service, for sorting and filtering across it.',
};

/**
 * The fleet, at the two altitudes a reader needs it.
 *
 * Clusters and services were two nav entries over the *same snapshot*: one keyed by
 * cluster, one flat. Two entries made that look like two sources, and the design
 * review found readers checking both to see whether they agreed. They are tabs now,
 * under one header that owns the things common to both -- what the snapshot is, how
 * old it is, and the action that rebuilds it.
 *
 * Nodes stayed a separate entry rather than becoming a third tab. It is a different
 * population, not a different altitude: it includes machines with no database on them,
 * which is the whole reason it exists, and it is the only page carrying actions that
 * change a machine.
 *
 * The tab is in the query string, not component state, so a link to the service
 * reading survives being pasted into a ticket -- the pattern the settings tabs already
 * use.
 */
export const FleetPage = () => {
  const [params, setParams] = useSearchParams();
  const requested = params.get('tab');
  const tab: TabId = TABS.includes(requested as TabId)
    ? (requested as TabId)
    : 'clusters';

  return (
    <Box>
      <OmHeader
        title="Fleet"
        subtitle={
          <Typography variant="body2" color="text.secondary">
            {SUBTITLE[tab]}
          </Typography>
        }
        actions={<SyncButton />}
      />
      <Tabs
        value={tab}
        onChange={(_event, next: TabId) => {
          // `replace`, so flipping between two readings of the same page does not
          // make Back walk the reader through every tab they looked at.
          const nextParams = new URLSearchParams(params);
          nextParams.set('tab', next);
          setParams(nextParams, { replace: true });
        }}
        sx={{ mb: 2 }}
      >
        <Tab
          value="clusters"
          label="Clusters"
          {...tabProps('fleet', 'clusters')}
        />
        <Tab
          value="services"
          label="Services"
          {...tabProps('fleet', 'services')}
        />
      </Tabs>
      <Box {...tabPanelProps('fleet', tab)}>
        {tab === 'clusters' ? <FleetClustersTab /> : <FleetServicesTab />}
      </Box>
    </Box>
  );
};
