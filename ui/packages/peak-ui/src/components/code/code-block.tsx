import { forwardRef } from 'react';
import Box from '@mui/material/Box';
import { useTheme } from '@mui/material/styles';
import type { Theme } from '@mui/material/styles';
import { Highlight, themes } from 'prism-react-renderer';
import type { PrismTheme } from 'prism-react-renderer';
import CopyToClipboardButton from '../buttons/copy-to-clipboard-button';
import type { CodeBlockProps } from './code-block.types';

// Background and text color always come from the prism scheme (applied via the inline `style`),
// so every block — highlighted or plain — shares the picked scheme's canvas.
const preSx = (wrap: boolean) => (theme: Theme) => ({
  margin: 0,
  ...theme.typography.code,
  fontSize: '0.875rem',
  lineHeight: 1.5,
  border: `1px solid ${theme.palette.dividers?.contour ?? theme.palette.divider}`,
  borderRadius: '5px',
  padding: theme.spacing(1.25, 1.5),
  ...(wrap ? { whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' } : { overflowX: 'auto' }),
});

// Scheme follows the app color mode — locked to brand-aligned light/dark for consistency.
const resolveScheme = (mode: Theme['palette']['mode']): PrismTheme =>
  mode === 'dark' ? themes.okaidia : themes.nightOwlLight;

const CodeBlock = forwardRef<HTMLPreElement, CodeBlockProps>(
  (
    {
      content,
      copyable = false,
      showCopyButtonText = false,
      value,
      language,
      wrap = false,
      sx,
      ...rest
    },
    ref
  ) => {
    const theme = useTheme();
    const codeText = value ?? (typeof content === 'string' ? content : '');
    const highlighted = !!language && !!codeText;
    const prismTheme = resolveScheme(theme.palette.mode);

    const copyButton = copyable && (
      <Box
        sx={{
          position: 'absolute',
          display: 'flex',
          borderRadius: showCopyButtonText ? 1.5 : '50%',
          backgroundColor: prismTheme.plain.backgroundColor,
          ...(showCopyButtonText
            ? { top: 12, right: 12 } // labeled
            : { top: 2, right: 2 }), // icon-only
        }}
      >
        <CopyToClipboardButton
          textToCopy={value ?? codeText}
          showCopyButtonText={showCopyButtonText}
          buttonProps={{ size: 'small', color: 'primary' }}
        />
      </Box>
    );

    if (highlighted) {
      return (
        <Box sx={{ position: 'relative' }}>
          <Highlight code={codeText.replace(/\n$/, '')} language={language} theme={prismTheme}>
            {({ style, tokens, getLineProps, getTokenProps }) => (
              <Box
                ref={ref}
                component="pre"
                style={style}
                sx={[preSx(wrap), ...(Array.isArray(sx) ? sx : [sx])]}
                {...rest}
              >
                <code>
                  {tokens.map((line, i) => {
                    const lineProps = getLineProps({ line });
                    return (
                      <span key={i} {...lineProps} style={{ ...lineProps.style, display: 'block' }}>
                        {line.map((token, key) => (
                          <span key={key} {...getTokenProps({ token })} />
                        ))}
                      </span>
                    );
                  })}
                </code>
              </Box>
            )}
          </Highlight>
          {copyButton}
        </Box>
      );
    }

    return (
      <Box sx={{ position: 'relative' }}>
        <Box
          ref={ref}
          component="pre"
          style={prismTheme.plain}
          sx={[preSx(wrap), ...(Array.isArray(sx) ? sx : [sx])]}
          {...rest}
        >
          <code>{content}</code>
        </Box>
        {copyButton}
      </Box>
    );
  }
);

CodeBlock.displayName = 'CodeBlock';

export default CodeBlock;
