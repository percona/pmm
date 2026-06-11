import { Theme } from '@mui/material';

export const getStyles = (theme: Theme) => ({
  healthy: {
    color: theme.palette.text.primary,
    borderColor: theme.palette.text.primary,
  },
  degraded: {
    color: theme.palette.warning.main,
    borderColor: theme.palette.warning.main,
  },
  critical: {
    color: theme.palette.error.contrastText,
    borderColor: theme.palette.error.surface,
    backgroundColor: theme.palette.error.surface,
    transition: 'none',
  },
  down: {
    color: theme.palette.error.contrastText,
    borderColor: theme.palette.error.surface,
    backgroundColor: theme.palette.error.surface,
    transition: 'none',
  },
});
