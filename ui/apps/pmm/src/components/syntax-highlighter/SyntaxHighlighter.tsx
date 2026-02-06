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
import { SyntaxHighlighterProps } from './SyntaxHighlighter.types';


ReactSyntaxHighlighter.registerLanguage('mongodb', mongodb);
ReactSyntaxHighlighter.registerLanguage('json', json);

const SyntaxHighlighter: FC<SyntaxHighlighterProps> = ({
  language,
  content,
  showCopyButton = false,
  ...props
}) => {
  const theme = useTheme();
  const highlighterStyle = getSyntaxHighlighterStyle(theme, language, showCopyButton);

  const handleCopy = () => {
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(content);
      enqueueSnackbar('Query copied to clipboard', { variant: 'success' });
    } else {
      enqueueSnackbar('Clipboard is not available', { variant: 'error' });
    }
  };

  return (
    <Stack
      sx={{
        overflow: 'hidden',
        borderWidth: 1,
        borderStyle: 'solid',
        borderColor: theme.palette.divider,
        borderRadius: theme.shape.borderRadius / 2,
        backgroundColor: theme.palette.surfaces?.elevation1 || 'transparent',
        position: 'relative',
      }}
    >
      {/* @ts-expect-error - react-syntax-highlighter types are incompatible with React 18 */}
      <ReactSyntaxHighlighter
        language={language}
        style={highlighterStyle}
        {...props}
      >
        {content}
      </ReactSyntaxHighlighter>
      {showCopyButton && (
        <IconButton sx={{ position: 'absolute', top: theme.spacing(1.5), right: theme.spacing(1.8), padding: 0 }} onClick={handleCopy}>
          <ContentCopyIcon sx={{ width: 18, height: 18 }} color='disabled' />
        </IconButton>
      )}
    </Stack>
  );
};

export default SyntaxHighlighter;
