import { useRef, type KeyboardEvent } from 'react';
import Box from '@mui/material/Box';
import { collapseClasses } from '@mui/material/Collapse';
import List from '@mui/material/List';
import { listItemButtonClasses } from '@mui/material/ListItemButton';
import { listItemIconClasses } from '@mui/material/ListItemIcon';
import Popover from '@mui/material/Popover';
import Typography from '@mui/material/Typography';
import useMediaQuery from '@mui/material/useMediaQuery';
import type { NavFlyoutProps } from './nav-flyout.types';

// The sticky heading's height; rows scrolled into view stay below it.
const HEADING_HEIGHT = 40;

// The entries themselves, without their actions or what a closed group holds.
const getEntries = (list: HTMLElement | null) =>
  Array.from(list?.querySelectorAll<HTMLElement>(`.${listItemButtonClasses.root}`) ?? []).filter(
    (entry) => !entry.closest(`.${collapseClasses.hidden}`)
  );

const NavFlyout = ({
  id,
  label,
  anchorEl,
  onClose,
  width = 250,
  autoFocus = true,
  children,
}: NavFlyoutProps) => {
  const reduceMotion = useMediaQuery('(prefers-reduced-motion: reduce)');
  const listRef = useRef<HTMLUListElement>(null);
  const labelId = `${id}-label`;

  const focusEntry = () => {
    const entries = getEntries(listRef.current);

    if (autoFocus) {
      (
        entries.find((entry) => entry.getAttribute('aria-current') === 'page') ?? entries[0]
      )?.focus();
    }
  };

  // Up and Down move a row at a time and wrap round, Home and End go to the first and last rows.
  // From an entry's action, they move on from its row.
  const handleKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    const entries = getEntries(listRef.current);
    const last = entries.length - 1;
    const target = event.target as Node;
    const row = entries.reduce(
      (found, entry, index) =>
        entry === target || entry.compareDocumentPosition(target) & Node.DOCUMENT_POSITION_FOLLOWING
          ? index
          : found,
      -1
    );
    const moves: Partial<Record<string, number>> = {
      ArrowDown: row < last ? row + 1 : 0,
      ArrowUp: row > 0 ? row - 1 : last,
      Home: 0,
      End: last,
    };
    const next = moves[event.key];

    if (next === undefined || last < 0) {
      return;
    }

    event.preventDefault();
    entries[next].focus();
  };

  return (
    <Popover
      id={id}
      open={!!anchorEl}
      anchorEl={anchorEl}
      onClose={onClose}
      anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
      transformOrigin={{ vertical: 'top', horizontal: 'left' }}
      transitionDuration={reduceMotion ? 0 : 'auto'}
      disableAutoFocus={!autoFocus}
      disableEnforceFocus={!autoFocus}
      slotProps={{
        paper: { sx: { ml: 1, width, scrollPaddingTop: HEADING_HEIGHT } },
        transition: { onEntered: focusEntry },
      }}
    >
      {/* The padding sits inside the scrolling paper, as in a menu, so the heading pins to its top edge. */}
      <Box component="nav" aria-labelledby={labelId} sx={{ py: 1 }}>
        <Typography
          id={labelId}
          variant="sectionHeading"
          component="div"
          sx={{
            position: 'sticky',
            top: 0,
            zIndex: 1,
            minHeight: HEADING_HEIGHT,
            px: 2,
            py: 1.25,
            bgcolor: 'background.paper',
          }}
        >
          {label}
        </Typography>
        <List
          ref={listRef}
          disablePadding
          onKeyDown={handleKeyDown}
          sx={(theme) => ({
            // Entries run edge to edge, like menu rows.
            [`& .${listItemButtonClasses.root}`]: {
              borderRadius: 0,
              marginInline: 0,
            },
            // An icon-less entry's label starts 16px in, under the heading: 22px padding, less
            // NavItem's 14px spacer, plus its 8px label padding.
            [`& .${listItemButtonClasses.root}:not(:has(.${listItemIconClasses.root}))`]: {
              paddingLeft: theme.spacing(2.75),
            },
          })}
        >
          {children}
        </List>
      </Box>
    </Popover>
  );
};

NavFlyout.displayName = 'NavFlyout';

export default NavFlyout;
