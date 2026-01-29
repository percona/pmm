import Stack from '@mui/material/Stack';
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { FC } from 'react';
import { CodeBlockProps } from './CodeBlock.types';

const CodeBlock: FC<CodeBlockProps> = ({ code, language, containerProps }) => (
  <Stack
    {...containerProps}
    sx={[
      {
        flex: 1,

        '*': {
          textOverflow: 'ellipsis',
        },
      },
      ...(Array.isArray(containerProps?.sx)
        ? containerProps?.sx
        : [containerProps?.sx]),
    ]}
  >
    <SyntaxHighlighter language={language}>{code}</SyntaxHighlighter>
  </Stack>
);

export default CodeBlock;
