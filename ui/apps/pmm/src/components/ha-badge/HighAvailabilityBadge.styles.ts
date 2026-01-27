import { Theme } from '@mui/material/styles';

export const getStyles = (theme: Theme) => ({
  healthy: {
    color: theme.palette.text.primary,
    borderColor: theme.palette.text.primary,
  },
  degraded: {
    color: theme.palette.warning.light,
    borderColor: theme.palette.warning.light,
  },
  critical: {
    color:
      theme.palette.mode === 'light'
        ? theme.palette.error.dark
        : theme.palette.error.contrastText,
    borderColor:
      theme.palette.mode === 'light'
        ? theme.palette.error.surface
        : theme.palette.error.dark,
    backgroundColor:
      theme.palette.mode === 'light'
      ? theme.palette.error.surface
      : theme.palette.error.dark,
    transition: 'none',
  },
  down: {
    color:
      theme.palette.mode === 'light'
        ? theme.palette.error.dark
        : theme.palette.error.contrastText,
    borderColor:
      theme.palette.mode === 'light'
        ? theme.palette.error.surface
        : theme.palette.error.dark,
    backgroundColor:
      theme.palette.mode === 'light'
        ? theme.palette.error.surface
        : theme.palette.error.dark,
    transition: 'none',
  },
});
