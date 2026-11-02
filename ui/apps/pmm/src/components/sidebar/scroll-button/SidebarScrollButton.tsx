import NorthIcon from '@mui/icons-material/North';
import SouthIcon from '@mui/icons-material/South';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import { Tooltip } from '@percona/peak-ui';
import { FC } from 'react';

interface Props {
  direction: 'up' | 'down';
  visible: boolean;
  onClick: () => void;
}

// A pointer alternative to the hidden scrollbar, for people who can click but
// not scroll. Keyboard users reach every item by tabbing, so it stays out of
// the tab order; its label keeps it available to voice control.
const SidebarScrollButton: FC<Props> = ({ direction, visible, onClick }) => {
  const label = `Scroll ${direction}`;

  return (
    <Box
      sx={(theme) => ({
        position: 'absolute',
        [direction === 'up' ? 'top' : 'bottom']: theme.spacing(0.5),
        left: '50%',
        transform: 'translateX(-50%)',
        zIndex: 1,
        borderRadius: '50%',
        backgroundColor: theme.palette.background.paper,
        opacity: visible ? 1 : 0,
        visibility: visible ? 'visible' : 'hidden',
        transition: theme.transitions.create(['opacity', 'visibility'], {
          duration: theme.transitions.duration.shorter,
        }),
      })}
    >
      {/* An empty title closes the tooltip when the button fades out under the
        pointer at the end of the list. */}
      <Tooltip
        title={visible ? label : ''}
        placement={direction === 'up' ? 'bottom' : 'top'}
      >
        <IconButton
          size="small"
          tabIndex={-1}
          aria-label={label}
          data-testid={`sidebar-scroll-${direction}`}
          onClick={onClick}
        >
          {direction === 'up' ? <NorthIcon /> : <SouthIcon />}
        </IconButton>
      </Tooltip>
    </Box>
  );
};

export default SidebarScrollButton;
