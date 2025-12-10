import Stack from '@mui/material/Stack';
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { FC } from 'react';
import { CodeBlockProps } from './CodeBlock.types';

const CodeBlock: FC<CodeBlockProps> = ({ code, language }) => (
  <Stack sx={{ flex: 1 }}>
    <SyntaxHighlighter language={language}>{code}</SyntaxHighlighter>
  </Stack>
);

export default CodeBlock;
