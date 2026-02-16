import Stack from '@mui/material/Stack';
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { FC } from 'react';
import { CodeBlockProps } from './CodeBlock.types';
import { mergeSx } from 'utils/styles.utils';

const CodeBlock: FC<CodeBlockProps> = ({ code, language, containerProps }) => (
  <Stack
    {...containerProps}
    sx={mergeSx([
      {
        flex: 1,

        '*': {
          textOverflow: 'ellipsis',
        },
      },
      containerProps?.sx,
    ])}
  >
    <SyntaxHighlighter language={language} content={code} />
  </Stack>
);

export default CodeBlock;
