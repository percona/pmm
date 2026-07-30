import { Theme } from '@mui/material';

export const getStyles = ({
  palette: { background, warning, error, info },
}: Theme) => ({
  icon: {
    width: 20,
    height: 20,

    '.background': {
      fill: background.paper,
    },
    '.status-at-risk': {
      fill: warning.main,
    },
    '.status-down': {
      fill: error.light,
    },
    '.status-updating': {
      fill: info.light,
    },
  },
});
