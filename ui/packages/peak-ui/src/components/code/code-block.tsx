import { forwardRef } from 'react';
import Box from '@mui/material/Box';
import { useTheme } from '@mui/material/styles';
import type { Theme } from '@mui/material/styles';
import { Highlight, themes } from 'prism-react-renderer';
import type { PrismTheme } from 'prism-react-renderer';
import CopyToClipboardButton from '../buttons/copy-to-clipboard-button';
import type { CodeBlockProps, CodeColorScheme } from './code-block.types';

const preSx = (copyable: boolean, highlighted: boolean) => (theme: Theme) => ({
  margin: 0,
  fontFamily: '"Roboto Mono", "Courier New", monospace',
  fontSize: '0.875rem',
  lineHeight: 1.6,
  border: `1px solid ${theme.palette.dividers?.divider ?? theme.palette.divider}`,
  borderRadius: theme.shape.borderRadius,
  padding: theme.spacing(2),
  paddingRight: copyable ? theme.spacing(7) : theme.spacing(2),
  overflowX: 'auto',
  ...(highlighted
    ? {}
    : {
        color: theme.palette.text.primary,
        backgroundColor: theme.palette.surfaces?.low ?? theme.palette.background.paper,
      }),
});

const resolveScheme = (
  colorScheme: CodeColorScheme | PrismTheme | undefined,
  mode: Theme['palette']['mode']
): PrismTheme => {
  if (colorScheme == null) {
    return mode === 'dark' ? themes.vsDark : themes.github;
  }
  return typeof colorScheme === 'string' ? themes[colorScheme] : colorScheme;
};

const CodeBlock = forwardRef<HTMLPreElement, CodeBlockProps>(
  (
    {
      content,
      copyable = false,
      showCopyButtonText = false,
      value,
      language,
      colorScheme,
      sx,
      ...rest
    },
    ref
  ) => {
    const theme = useTheme();
    const codeText = value ?? (typeof content === 'string' ? content : '');
    const highlighted = !!language && !!codeText;

    const copyButton = copyable && (
      <Box sx={{ position: 'absolute', top: 4, right: 4 }}>
        <CopyToClipboardButton
          textToCopy={value ?? codeText}
          showCopyButtonText={showCopyButtonText}
          buttonProps={{ size: 'small', color: 'primary' }}
        />
      </Box>
    );

    if (highlighted) {
      const prismTheme = resolveScheme(colorScheme, theme.palette.mode);

      return (
        <Box sx={{ position: 'relative' }}>
          <Highlight code={codeText.replace(/\n$/, '')} language={language} theme={prismTheme}>
            {({ style, tokens, getLineProps, getTokenProps }) => (
              <Box
                ref={ref}
                component="pre"
                style={style}
                sx={[preSx(copyable, true), ...(Array.isArray(sx) ? sx : [sx])]}
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
          sx={[preSx(copyable, false), ...(Array.isArray(sx) ? sx : [sx])]}
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
