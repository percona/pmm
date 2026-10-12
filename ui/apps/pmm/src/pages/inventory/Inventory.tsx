import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import { Page } from 'components/page';
import { PMM_INVENTORY_NODES_PATH, PMM_INVENTORY_PATH } from 'lib/constants';
import type { FC } from 'react';
import { Outlet, useMatch, useNavigate } from 'react-router-dom';
import { OrgRole } from 'types/user.types';
import { Messages } from './Inventory.messages';
import type { InventoryTab } from './Inventory.types';

export const Inventory: FC = () => {
  const navigate = useNavigate();
  const tab: InventoryTab = useMatch(PMM_INVENTORY_NODES_PATH)
    ? 'nodes'
    : 'services';

  return (
    <Page
      title={Messages.title}
      maxWidth="full"
      surface="paper"
      roles={[OrgRole.Admin]}
    >
      <Stack gap={3} sx={{ flex: 1 }}>
        <Tabs
          data-testid="inventory-tabs"
          value={tab}
          onChange={(_, value: InventoryTab) =>
            navigate(`${PMM_INVENTORY_PATH}/${value}`)
          }
          sx={{ borderBottom: 1, borderColor: 'divider' }}
        >
          <Tab
            data-testid="inventory-tab-services"
            value="services"
            label={Messages.tabs.services}
          />
          <Tab
            data-testid="inventory-tab-nodes"
            value="nodes"
            label={Messages.tabs.nodes}
          />
        </Tabs>
        <Outlet />
      </Stack>
    </Page>
  );
};
