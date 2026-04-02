import { Theme } from '@mui/material';

export const getStyles = ({
  palette: { background, warning, error, info },
}: Theme) => ({
  icon: {
    width: 20,
    height: 20,

    '.background': {
      fill: background.default,
    },
    '.status-at-risk': {
      fill: warning.light,
    },
    '.status-down': {
      fill: error.light,
    },
    '.status-updating': {
      fill: info.light,
    },
  },
});
