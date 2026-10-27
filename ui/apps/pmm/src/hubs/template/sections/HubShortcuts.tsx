import { ShortcutTile, ShortcutTiles } from '@percona/peak-ui';
import type { FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { Messages } from '../hub.messages';
import type { HubDestination, HubDestinationLook } from '../hub.types';
import { useDestinationHref } from './HubListRow';

const HubShortcut: FC<{
  destination: HubDestination;
  look: HubDestinationLook;
}> = ({ destination, look }) => {
  const href = useDestinationHref(destination);
  const Icon = look.icon;

  return (
    <ShortcutTile
      component={RouterLink}
      to={href}
      icon={Icon && <Icon />}
      label={destination.label}
      tooltip={look.path.join(' › ')}
    />
  );
};

export interface HubShortcutsProps {
  shortcuts: HubDestination[];
  describeDestination: (destination: HubDestination) => HubDestinationLook;
}

/** The places the hub leads to most: a grid of tiles under the header's divider; navigation only, the same in every state. */
export const HubShortcuts: FC<HubShortcutsProps> = ({
  shortcuts,
  describeDestination,
}) => (
  <ShortcutTiles component="nav" aria-label={Messages.shortcuts}>
    {shortcuts.map((destination) => (
      <HubShortcut
        key={destination.route}
        destination={destination}
        look={describeDestination(destination)}
      />
    ))}
  </ShortcutTiles>
);
