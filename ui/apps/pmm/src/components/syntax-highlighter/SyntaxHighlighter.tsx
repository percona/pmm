import Stack from '@mui/material/Stack';
import { useTheme } from '@mui/material/styles';
import { FC } from 'react';
import { PrismLight as ReactSyntaxHighlighter } from 'react-syntax-highlighter';
import { getSyntaxHighlighterStyle } from './SyntaxHighlighter.utils';

// Import only used languages to reduce bundle size
// @ts-ignore
import mongodb from 'react-syntax-highlighter/dist/esm/languages/prism/mongodb';
import { SyntaxHighlighterProps } from './SyntaxHighlighter.types';

ReactSyntaxHighlighter.registerLanguage('mongodb', mongodb);

const SyntaxHighlighter: FC<SyntaxHighlighterProps> = ({
  language,
  children,
  ...props
}) => {
  const theme = useTheme();
  const highlighterStyle = getSyntaxHighlighterStyle(theme, language);

  return (
    <Stack
      sx={{
        overflow: 'hidden',
        borderWidth: 1,
        borderStyle: 'solid',
        borderColor: theme.palette.divider,
        borderRadius: theme.shape.borderRadius / 2,
        backgroundColor: theme.palette.surfaces?.elevation1 || 'transparent',
      }}
    >
      {/* @ts-expect-error - react-syntax-highlighter types are incompatible with React 18 */}
      <ReactSyntaxHighlighter
        language={language}
        style={highlighterStyle}
        {...props}
      >
        {children}
      </ReactSyntaxHighlighter>
    </Stack>
  );
};

export default SyntaxHighlighter;
