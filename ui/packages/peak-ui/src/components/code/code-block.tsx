import { forwardRef } from 'react';
import Box from '@mui/material/Box';
import type { Theme } from '@mui/material/styles';
import CopyToClipboardButton from '../buttons/copy-to-clipboard-button';
import type { CodeBlockProps } from './code-block.types';

const preSx = (copyable: boolean) => (theme: Theme) => ({
  margin: 0,
  fontFamily: '"Roboto Mono", "Courier New", monospace',
  fontSize: '0.875rem',
  lineHeight: 1.6,
  color: theme.palette.text.primary,
  backgroundColor: theme.palette.surfaces?.low ?? theme.palette.background.paper,
  border: `1px solid ${theme.palette.dividers?.divider ?? theme.palette.divider}`,
  borderRadius: theme.shape.borderRadius,
  padding: theme.spacing(2),
  paddingRight: copyable ? theme.spacing(7) : theme.spacing(2),
  overflowX: 'auto',
});

const CodeBlock = forwardRef<HTMLPreElement, CodeBlockProps>(
  ({ children, copyable = false, showCopyButtonText = false, value, sx, ...rest }, ref) => {
    const textToCopy = value ?? (typeof children === 'string' ? children : '');

    return (
      <Box sx={{ position: 'relative' }}>
        <Box
          ref={ref}
          component="pre"
          sx={[preSx(copyable), ...(Array.isArray(sx) ? sx : [sx])]}
          {...rest}
        >
          <code>{children}</code>
        </Box>
        {copyable && (
          <Box sx={{ position: 'absolute', top: 4, right: 4 }}>
            <CopyToClipboardButton
              textToCopy={textToCopy}
              showCopyButtonText={showCopyButtonText}
              buttonProps={{ size: 'small', color: 'primary' }}
            />
          </Box>
        )}
      </Box>
    );
  }
);

CodeBlock.displayName = 'CodeBlock';

export default CodeBlock;
