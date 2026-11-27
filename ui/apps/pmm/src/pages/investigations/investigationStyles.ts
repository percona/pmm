import { adreColors } from 'components/adre/adreAnswerSx';

/** Status colors on the dark pages: amber while open, accent while running, green when done. */
export const STATUS_STYLES: Record<string, { color: string; bg: string }> = {
  open: { color: '#FCD34D', bg: 'rgba(252,211,77,0.12)' },
  in_progress: { color: '#FCD34D', bg: 'rgba(252,211,77,0.12)' },
  investigating: { color: '#FCD34D', bg: 'rgba(252,211,77,0.12)' },
  running: { color: '#CFCBFF', bg: 'rgba(182,178,255,0.14)' },
  completed: { color: '#86EFAC', bg: 'rgba(134,239,172,0.12)' },
  resolved: { color: '#86EFAC', bg: 'rgba(134,239,172,0.12)' },
  failed: { color: '#FCA5A5', bg: 'rgba(248,113,113,0.14)' },
  archived: { color: '#A9A7B0', bg: 'rgba(255,255,255,0.06)' },
};

export const pillButtonSx = {
  height: 36,
  px: 2,
  borderRadius: 999,
  textTransform: 'none',
  fontSize: 13,
  fontWeight: 500,
  color: adreColors.text,
  borderColor: 'rgba(255,255,255,0.16)',
} as const;

export const accentButtonSx = {
  ...pillButtonSx,
  border: 0,
  bgcolor: adreColors.accent,
  color: adreColors.panel,
  '&:hover': { bgcolor: '#C9C6FF' },
} as const;

export const sourceLabel = (sourceType?: string) =>
  sourceType === 'alert'
    ? 'Alert'
    : sourceType === 'chat'
      ? 'Assistant chat'
      : 'User request';
