import AddOutlined from '@mui/icons-material/AddOutlined';
import MoreVertOutlined from '@mui/icons-material/MoreVertOutlined';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import { Tooltip } from '@percona/peak-ui';
import {
  type FC,
  type MouseEvent,
  type ReactNode,
  useId,
  useState,
} from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { ThemeScope } from 'components/theme-scope';
import { Messages } from '../hub.messages';
import type { HubMenuAction, HubViewModel } from '../hub.types';
import { DoorButton } from './DoorLink';
import { visuallyHidden } from './hub.sx';

// Each menu ends under the right edge of the control that opens it.
const BELOW_RIGHT = {
  anchorOrigin: { vertical: 'bottom', horizontal: 'right' },
  transformOrigin: { vertical: 'top', horizontal: 'right' },
} as const;

const useMenu = () => {
  const buttonId = useId();
  const menuId = useId();
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);

  return {
    anchor,
    close: () => setAnchor(null),
    trigger: {
      id: buttonId,
      'aria-haspopup': 'menu' as const,
      'aria-controls': anchor ? menuId : undefined,
      'aria-expanded': anchor ? true : undefined,
      onClick: (event: MouseEvent<HTMLElement>) =>
        setAnchor(event.currentTarget),
    },
    menu: {
      id: menuId,
      anchorEl: anchor,
      open: !!anchor,
      onClose: () => setAnchor(null),
      slotProps: { list: { 'aria-labelledby': buttonId } },
      ...BELOW_RIGHT,
    },
  };
};

const ActionItem: FC<{ action: HubMenuAction; onClose: () => void }> = ({
  action: { label, description, to, icon: Icon, external },
  onClose,
}) => {
  const content: ReactNode = (
    <>
      <ListItemIcon>
        <Icon />
      </ListItemIcon>
      <ListItemText
        primary={
          <>
            {label}
            {external && (
              <Box component="span" sx={visuallyHidden}>
                {` ${Messages.newTab}`}
              </Box>
            )}
          </>
        }
        secondary={description}
        slotProps={{ secondary: { variant: 'helperText' } }}
      />
    </>
  );
  // A description wraps under its label, with the icon on the label's line.
  const sx = description
    ? { whiteSpace: 'normal', alignItems: 'flex-start' }
    : undefined;

  return external ? (
    <MenuItem
      component="a"
      href={to}
      target="_blank"
      rel="noopener noreferrer"
      onClick={onClose}
      sx={sx}
    >
      {content}
    </MenuItem>
  ) : (
    <MenuItem component={RouterLink} to={to} onClick={onClose} sx={sx}>
      {content}
    </MenuItem>
  );
};

export interface HubHeaderActionsProps {
  /** Technology name, for "Add MongoDB". */
  name: string;
  actions: HubViewModel['actions'];
}

/** Add MongoDB, a menu of the ways in when there are two, then the More actions menu. */
export const HubHeaderActions: FC<HubHeaderActionsProps> = ({
  name,
  actions: { add, more },
}) => {
  const addMenu = useMenu();
  const moreMenu = useMenu();
  // "Add" on screen, as the page already says MongoDB; "Add MongoDB" for assistive tech.
  const addContext = Messages.addContext(name);

  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
      {/* Only Add is neutral, Peak's Base theme; its menu and More stay in the PMM theme. */}
      {add.length === 1 && (
        <ThemeScope theme="base">
          <DoorButton
            door={{ label: Messages.add, to: add[0].to, context: addContext }}
            size="medium"
            startIcon={<AddOutlined />}
          />
        </ThemeScope>
      )}
      {add.length > 1 && (
        <>
          <ThemeScope theme="base">
            <Button
              variant="text"
              startIcon={<AddOutlined />}
              {...addMenu.trigger}
            >
              {Messages.add}
              <Box component="span" sx={visuallyHidden}>
                {` ${addContext}`}
              </Box>
            </Button>
          </ThemeScope>
          <Menu
            {...addMenu.menu}
            slotProps={{
              ...addMenu.menu.slotProps,
              paper: { sx: { minWidth: 240 } },
            }}
          >
            {add.map((action) => (
              <ActionItem
                key={action.to}
                action={action}
                onClose={addMenu.close}
              />
            ))}
          </Menu>
        </>
      )}
      {more.length > 0 && (
        <>
          <Tooltip title={Messages.moreActions}>
            <IconButton aria-label={Messages.moreActions} {...moreMenu.trigger}>
              <MoreVertOutlined />
            </IconButton>
          </Tooltip>
          <Menu
            {...moreMenu.menu}
            slotProps={{
              ...moreMenu.menu.slotProps,
              paper: { sx: { width: 240 } },
            }}
          >
            {more.map((action) => (
              <ActionItem
                key={action.to}
                action={action}
                onClose={moreMenu.close}
              />
            ))}
          </Menu>
        </>
      )}
    </Box>
  );
};
