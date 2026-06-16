import { forwardRef } from 'react';
import Box from '@mui/material/Box';
import type { Theme } from '@mui/material/styles';
import type { CodeProps } from './code.types';

const codeSx = (theme: Theme) => ({
  fontFamily: '"Roboto Mono", "Courier New", monospace',
  fontSize: '0.875em',
  color: theme.palette.text.primary,
  backgroundColor: theme.palette.action.hover,
  border: `1px solid ${theme.palette.dividers?.divider ?? theme.palette.divider}`,
  borderRadius: theme.shape.borderRadius,
  padding: '0.1em 0.3em',
  whiteSpace: 'break-spaces',
  wordBreak: 'break-word',
});

const Code = forwardRef<HTMLElement, CodeProps>(({ sx, ...rest }, ref) => (
  <Box ref={ref} component="code" sx={[codeSx, ...(Array.isArray(sx) ? sx : [sx])]} {...rest} />
));

Code.displayName = 'Code';

export default Code;
