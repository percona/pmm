import Link from '@mui/material/Link';
import { Chip } from '@percona/peak-ui';
import type { FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import type { ManagedAgent } from 'types/agent.types';
import { Messages } from '../../Inventory.messages';
import type { InventoryTab } from '../../Inventory.types';
import { getAgentsUrl, isAgentRunning } from '../../Inventory.utils';

interface Props {
  agents: ManagedAgent[];
  tab: InventoryTab;
  id: string;
}

export const AgentsStatusChip: FC<Props> = ({ agents, tab, id }) => {
  const running = agents.filter(isAgentRunning).length;
  const color =
    running === agents.length ? 'success' : running ? 'warning' : 'error';

  return (
    <Chip
      size="small"
      color={color}
      data-testid="agents-status-chip"
      label={
        <Link component={RouterLink} to={getAgentsUrl(tab, id)} color="inherit">
          {Messages.agentsSummary(running, agents.length)}
        </Link>
      }
    />
  );
};
