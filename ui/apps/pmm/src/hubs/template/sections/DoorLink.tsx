import Box from '@mui/material/Box';
import Button, { type ButtonProps } from '@mui/material/Button';
import Link, { type LinkProps } from '@mui/material/Link';
import type { FC, ReactNode } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { PreviewChip } from 'components/preview-chip';
import type { HubDoor } from '../hub.types';
import { visuallyHidden } from './hub.sx';

interface DoorProps {
  door: HubDoor;
  /** Visible content; the door's label when absent. */
  children?: ReactNode;
}

export type DoorLinkProps = DoorProps &
  Omit<LinkProps<typeof RouterLink>, 'to' | 'children' | 'component'>;

export type DoorButtonProps = DoorProps &
  Omit<ButtonProps<typeof RouterLink>, 'to' | 'children' | 'component'>;

const DoorExtras: FC<{ door: HubDoor }> = ({ door }) => (
  <>
    {door.context && (
      <Box component="span" sx={visuallyHidden}>
        {` ${door.context}`}
      </Box>
    )}
    {door.preview && (
      // No height of its own, like Peak's button icon slots, so the chip never makes a button taller.
      <Box
        component="span"
        sx={{ display: 'inline-flex', alignItems: 'center', height: 0, ml: 1 }}
      >
        <PreviewChip level={door.preview} />
      </Box>
    )}
  </>
);

/** A router link to a hub door, with its hidden context and Preview chip. */
export const DoorLink: FC<DoorLinkProps> = ({
  door,
  children,
  underline = 'hover',
  ...rest
}) => (
  <Link component={RouterLink} to={door.to} underline={underline} {...rest}>
    {children ?? door.label}
    <DoorExtras door={door} />
  </Link>
);

/** A small text button that opens a hub door. */
export const DoorButton: FC<DoorButtonProps> = ({
  door,
  children,
  variant = 'text',
  size = 'small',
  ...rest
}) => (
  <Button
    component={RouterLink}
    to={door.to}
    variant={variant}
    size={size}
    {...rest}
  >
    {children ?? door.label}
    <DoorExtras door={door} />
  </Button>
);
