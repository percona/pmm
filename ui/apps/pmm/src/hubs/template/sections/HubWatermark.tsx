import '@fontsource/poppins/800.css';
import Box from '@mui/material/Box';
import { Messages } from '../hub.messages';

const ANGLE = -24;

/** One big, see-through "Work in progress" line across the hub, centred in the window as the page scrolls under it. */
export const HubWatermark = () => (
  <Box
    sx={{
      position: 'absolute',
      inset: 0,
      zIndex: 1,
      pointerEvents: 'none',
      userSelect: 'none',
      // Clip, not hidden: hidden would make this box the line's scroller and stop it sticking to the window.
      overflow: 'clip',
    }}
  >
    <Box
      sx={{
        position: 'sticky',
        top: 0,
        height: 'min(100%, 100dvh)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <Box
        component="span"
        sx={(theme) => ({
          display: 'inline-block',
          transform: `rotate(${ANGLE}deg)`,
          fontFamily: theme.typography.h1.fontFamily,
          fontWeight: 800,
          // Most of the hub's width, or less in a short window, so the slanted line stays in view.
          fontSize: 'min(10cqi, 20dvh)',
          lineHeight: 1,
          whiteSpace: 'nowrap',
          color: theme.palette.text.primary,
          opacity: 0.12,
        })}
      >
        {Messages.workInProgress}
      </Box>
    </Box>
  </Box>
);
