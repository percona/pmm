import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import IconButton from '@mui/material/IconButton';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import { useTheme } from '@mui/material/styles';
import { FC } from 'react';
import { PrismLight as ReactSyntaxHighlighter } from 'react-syntax-highlighter';
import { enqueueSnackbar } from 'notistack';
import { getSyntaxHighlighterStyle } from './SyntaxHighlighter.utils';

// Import only used languages to reduce bundle size
// @ts-ignore
import mongodb from 'react-syntax-highlighter/dist/esm/languages/prism/mongodb';
import json from 'react-syntax-highlighter/dist/esm/languages/prism/json';
// @ts-ignore
import sql from 'react-syntax-highlighter/dist/esm/languages/prism/sql';
import { SyntaxHighlighterProps } from './SyntaxHighlighter.types';

ReactSyntaxHighlighter.registerLanguage('mongodb', mongodb);
ReactSyntaxHighlighter.registerLanguage('json', json);
ReactSyntaxHighlighter.registerLanguage('sql', sql);

const SyntaxHighlighter: FC<SyntaxHighlighterProps> = ({
  language,
  content,
  showCopyButton = false,
  disableBorder = false,
  maxHeight,
  ...props
}) => {
  const theme = useTheme();
  const highlighterStyle = getSyntaxHighlighterStyle(
    theme,
    language,
    showCopyButton
  );

  const handleCopy = async () => {
    if (navigator.clipboard && window.isSecureContext) {
      try {
        await navigator.clipboard.writeText(content);
        enqueueSnackbar('Query copied to clipboard', { variant: 'success' });
      } catch (error) {
        enqueueSnackbar('Failed to copy query to clipboard', {
          variant: 'error',
        });
      }
    } else {
      enqueueSnackbar('Clipboard is not available', { variant: 'error' });
    }
  };

  const highlighterBlock = (
    <>
      {/* @ts-expect-error - react-syntax-highlighter types are incompatible with React 18 */}
      <ReactSyntaxHighlighter
        language={language}
        style={highlighterStyle}
        {...props}
      >
        {content}
      </ReactSyntaxHighlighter>
    </>
  );

  return (
    <Stack
      sx={{
        overflow: 'hidden',
        ...(!disableBorder && {
          borderWidth: 1,
          borderStyle: 'solid',
          borderColor: theme.palette.divider,
          borderRadius: Number(theme.shape.borderRadius) / 2,
        }),
        backgroundColor: theme.palette.surfaces?.high || 'transparent',
        position: 'relative',
      }}
    >
      {maxHeight != null ? (
        <Box sx={{ maxHeight, overflow: 'auto', minHeight: 0 }}>
          {highlighterBlock}
        </Box>
      ) : (
        highlighterBlock
      )}
      {showCopyButton && (
        <IconButton
          sx={{
            position: 'absolute',
            top: theme.spacing(1.5),
            right: theme.spacing(1.8),
            padding: 0,
          }}
          onClick={handleCopy}
        >
          <ContentCopyIcon sx={{ width: 18, height: 18 }} color="disabled" />
        </IconButton>
      )}
    </Stack>
  );
};

export default SyntaxHighlighter;
