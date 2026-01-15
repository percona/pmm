import { SxProps, Theme } from '@mui/material';

/**
 * External documentation and feedback URLs
 */
export const DOCS_URL =
  'https://docs.percona.com/percona-monitoring-and-management/3/get-started/query-analytics.html';
export const FORUM_URL =
  'https://forums.percona.com/c/percona-monitoring-and-management-pmm/percona-monitoring-and-management-pmm-v3';

/**
 * Reusable link styles for documentation and feedback links
 */
export const linkStyles: SxProps<Theme> = (theme) => ({
  fontFamily: 'Roboto, sans-serif',
  fontSize: '14px',
  fontWeight: 400,
  lineHeight: 1.5,
  color: theme.palette.info.light,
  textAlign: 'center',
  textDecoration: 'underline solid',
  textDecorationSkipInk: 'none',
  textUnderlinePosition: 'from-font',
  fontVariationSettings: "'wdth' 100",
  '&:hover': {
    color: theme.palette.info.main,
  },
});

/**
 * Typography styles for page title
 */
export const titleStyles: SxProps<Theme> = {
  fontFamily: 'Poppins, sans-serif',
  fontWeight: 600,
  fontSize: '23px',
  lineHeight: 1.125,
  textAlign: 'center',
};

/**
 * Typography styles for page description
 */
export const descriptionStyles: SxProps<Theme> = {
  fontFamily: 'Roboto, sans-serif',
  fontWeight: 400,
  fontSize: '16px',
  lineHeight: 1.375,
  textAlign: 'center',
  fontVariationSettings: "'wdth' 100",
};

/**
 * Typography styles for secondary text
 */
export const secondaryTextStyles: SxProps<Theme> = {
  fontFamily: 'Roboto, sans-serif',
  fontWeight: 400,
  fontSize: '14px',
  lineHeight: 1.5,
  textAlign: 'center',
  fontVariationSettings: "'wdth' 100",
};
