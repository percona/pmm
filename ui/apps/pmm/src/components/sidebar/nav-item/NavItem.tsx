import {
  ListItemButton,
  ListItemText,
  Stack,
  Collapse,
  List,
  ListItem,
  ListItemIcon,
  typographyClasses,
  Divider,
} from '@mui/material';
import { useLinkWithVariables } from 'hooks/utils/useLinkWithVariables';
import { isActive } from 'lib/utils/navigation.utils';
import { FC, useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { NavItemProps } from './NavItem.types';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import KeyboardArrowUpIcon from '@mui/icons-material/KeyboardArrowUp';
import { Icon } from 'components/icon';
import { getLinkProps } from './NavItem.utils';

const NavItem: FC<NavItemProps> = ({ item, drawerOpen }) => {
  const location = useLocation();
  const active = isActive(item, location.pathname);
  const [open, setIsOpen] = useState(active);
  const url = useLinkWithVariables(item.url);
  const linkProps = getLinkProps(item, url);
  const color = active ? 'primary.main' : 'primary.text.primary';
  const children = item.children?.filter((i) => !i.hidden);

  useEffect(() => {
    if (active) {
      setIsOpen(true);
    }
  }, [active]);

  if (children?.length && drawerOpen) {
    return (
      <>
        <ListItemButton
          disableGutters
          sx={{
            pl: 3.4,
            color,

            [`.${typographyClasses.root}`]: {
              fontWeight: 600,
            },
          }}
          onClick={() => setIsOpen(!open)}
        >
          {item.icon && (
            <ListItemIcon
              sx={{
                minWidth: 'auto',
                pr: drawerOpen ? 1 : 0,
              }}
            >
              <Icon name={item.icon} sx={{ color }} />
            </ListItemIcon>
          )}
          <ListItemText
            primary={item.text}
            primaryTypographyProps={{
              style: { whiteSpace: 'normal' },
            }}
          />
          <Stack pl={2} pr={2}>
            {open ? <KeyboardArrowUpIcon /> : <KeyboardArrowDownIcon />}
          </Stack>
        </ListItemButton>
        <Collapse in={open} timeout="auto">
          <List component="div" disablePadding sx={{ ml: 2 }}>
            {children.map((item) => (
              <NavItem key={item.id} item={item} drawerOpen={drawerOpen} />
            ))}
          </List>
        </Collapse>
      </>
    );
  }

  if (item.isDivider) {
    return (
      <ListItem
        sx={[
          !drawerOpen && {
            justifyContent: 'center',
          },
        ]}
      >
        <Divider
          sx={
            drawerOpen
              ? {
                  mr: 1,
                  ml: 2,
                  flex: 1,
                }
              : {
                  flex: 1,
                }
          }
        />
      </ListItem>
    );
  }

  return (
    <ListItem key={item.url} disablePadding>
      {/* @ts-ignore */}
      <ListItemButton
        disableGutters
        sx={[
          {
            px: 4,
            color,
            backgroundColor: active ? 'rgba(	220, 63, 0, 0.08)' : 'initial',

            [`.${typographyClasses.root}`]: {
              fontWeight: 600,
            },
          },
          !drawerOpen && {
            justifyContent: 'center',
          },
        ]}
        {...linkProps}
      >
        {item.icon && (
          <ListItemIcon
            sx={{
              minWidth: 'auto',
              pr: drawerOpen ? 1 : 0,
            }}
          >
            <Icon
              name={item.icon}
              sx={{
                ml: '-5px',
                color,
              }}
            />
          </ListItemIcon>
        )}
        {drawerOpen && (
          <ListItemText
            primary={item.text}
            primaryTypographyProps={{ style: { whiteSpace: 'normal' } }}
          />
        )}
      </ListItemButton>
    </ListItem>
  );
};

export default NavItem;
