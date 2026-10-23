import type { FC } from 'react';
import type { HubViewModel } from '../hub.types';
import { DoorButton } from './DoorLink';

export interface HubActionsProps {
  actions: HubViewModel['actions'];
}

/** An empty estate's ways in, as buttons: the first one leads. */
export const HubActions: FC<HubActionsProps> = ({ actions: { add } }) => (
  <>
    {add.map(({ icon: Icon, label, to }, index) => (
      <DoorButton
        key={to}
        door={{ label, to }}
        variant={index === 0 ? 'contained' : 'outlined'}
        size="medium"
        startIcon={<Icon />}
      />
    ))}
  </>
);
