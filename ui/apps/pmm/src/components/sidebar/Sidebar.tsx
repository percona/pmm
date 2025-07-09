import {
  Collapse,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  Stack,
} from '@mui/material';
import { FC, useEffect, useState } from 'react';
import { PmmRoundedIcon } from 'icons';
import { Link, useLocation } from 'react-router-dom';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import KeyboardArrowUpIcon from '@mui/icons-material/KeyboardArrowUp';
import { useLinkWithVariables } from 'hooks/utils/useLinkWithVariables';
import { useNavigation } from 'contexts/navigation';
import { isActive } from 'lib/utils/navigation.utils';
import { NavItem } from 'lib/types';

export const SidebarItem: FC<{ item: NavItem }> = ({ item }) => {
  const location = useLocation();
  const active = isActive(item, location.pathname);
  const [open, setIsOpen] = useState(active);
  const to = useLinkWithVariables(item.url);
  const linkProps = !item.target
    ? {
        LinkComponent: Link,
        to: to,
      }
    : {
        href: item.url,
        target: item.target,
      };

  useEffect(() => {
    if (active) {
      setIsOpen(true);
    }
  }, [active]);

  if (item?.children) {
    return (
      <>
        <ListItemButton
          disableGutters
          sx={{ pl: 4 }}
          onClick={() => setIsOpen(!open)}
        >
          <ListItemText
            primary={item.text}
            primaryTypographyProps={{
              fontWeight: active ? 'bold' : '',
            }}
          />
          <Stack pl={2} pr={2}>
            {open ? <KeyboardArrowUpIcon /> : <KeyboardArrowDownIcon />}
          </Stack>
        </ListItemButton>
        <Collapse in={open} timeout="auto">
          <List component="div" disablePadding sx={{ ml: 2 }}>
            {item.children.map((item) => (
              <SidebarItem key={item.id} item={item} />
            ))}
          </List>
        </Collapse>
      </>
    );
  }

  return (
    <ListItem key={item.url} disablePadding>
      {/* @ts-ignore */}
      <ListItemButton
        disableGutters
        sx={{ px: 4, fontWeight: active ? 'bold' : '' }}
        {...linkProps}
      >
        {item.text}
      </ListItemButton>
    </ListItem>
  );
};

/**
 * TODO currently just mostly a placeholder until the "real" menu is in place
 */
export const Sidebar: FC = () => {
  const { navTree } = useNavigation();

  return (
    <Stack
      direction="column"
      sx={(theme) => ({
        gap: 1,
        backgroundColor: theme.palette.common.white,
        alignItems: 'flex-start',
      })}
      data-testid="pmm-sidebar"
    >
      <Stack
        sx={{
          pt: 2,
          px: 4,
        }}
      >
        <PmmRoundedIcon sx={{ height: 30, width: 'auto' }} />
      </Stack>
      <List disablePadding sx={{ width: '100%' }}>
        {navTree.map((item) => (
          <SidebarItem key={item.url} item={item} />
        ))}
      </List>
    </Stack>
  );
};
