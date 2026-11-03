import { css } from '@emotion/css';
import type { GrafanaTheme2 } from '@grafana/data';

export const getStyles = (theme: GrafanaTheme2) => ({
  link: css({
    color: theme.colors.text.link,
    cursor: 'pointer',
    fontWeight: theme.typography.fontWeightMedium,
    textDecoration: 'underline',
    textUnderlineOffset: '2px',
    '&:hover': {
      color: theme.colors.emphasize(theme.colors.text.link, 0.08),
    },
  }),
});
