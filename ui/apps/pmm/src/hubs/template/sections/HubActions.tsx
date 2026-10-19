import AddOutlined from '@mui/icons-material/AddOutlined';
import type { FC } from 'react';
import type { HubViewModel } from '../hub.types';
import { DoorButton } from './DoorLink';

export interface HubActionsProps {
  actions: HubViewModel['actions'];
}

/** The page's ways to grow the estate: add a service, and install one with Operations. */
export const HubActions: FC<HubActionsProps> = ({ actions }) => (
  <>
    {actions.addService && (
      <DoorButton
        door={actions.addService}
        variant="contained"
        size="medium"
        startIcon={<AddOutlined />}
      />
    )}
    {actions.install && (
      <DoorButton door={actions.install} variant="outlined" size="medium" />
    )}
  </>
);
