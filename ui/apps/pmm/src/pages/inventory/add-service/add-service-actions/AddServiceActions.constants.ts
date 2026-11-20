import type { SxProps, Theme } from '@mui/material/styles';

// Same bar as the Settings page's submit button.
export const STICKY_ACTIONS_SX: SxProps<Theme> = {
  position: 'sticky',
  bottom: 0,
  py: 2,
  bgcolor: 'background.paper',
  borderTop: 1,
  borderColor: 'divider',
  zIndex: 1,
  boxShadow: (theme) =>
    `-8px 0 0 0 ${theme.palette.background.paper}, 30px 0 0 0 ${theme.palette.background.paper}`,
};
