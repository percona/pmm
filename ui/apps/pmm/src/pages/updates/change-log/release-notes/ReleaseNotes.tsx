import { FC } from 'react';
import Markdown from 'react-markdown';
import { ReleaseNotesProps } from './ReleaseNotes.types';
import rehypeRaw from 'rehype-raw';
import remarkGfm from 'remark-gfm';
import { Link, Stack } from '@mui/material';
import { CodeBlock } from '../code-block';
import { IconMap } from './ReleaseNotes.constants';

export const ReleaseNotes: FC<ReleaseNotesProps> = ({ content }) => {
  if (!content) {
    return null;
  }

  return (
    <Stack
      sx={(theme) => ({
        img: {
          height: 'auto',
          maxWidth: '100%',
        },
        'table, tr, th, td': {
          p: 1,
          textAlign: 'left',
          borderWidth: 1,
          borderStyle: 'solid',
          borderColor: theme.palette.divider,
          borderCollapse: 'collapse',
        },
        '.alert': {
          borderWidth: 1,
          borderStyle: 'solid',
          borderColor: theme.palette.divider,
          borderRadius: theme.shape.borderRadius,
          overflow: 'hidden',
          p: 2,
          mb: 2,

          '& > *': {
            m: 0,
          },

          '& h4': {
            p: 2,
            margin: -2,
            mb: 1,
            backgroundColor: theme.palette.surfaces?.low,
            gap: 2,
            display: 'flex',
          },

          '&.note, &.caution, &.hint, &.summary, &.seealso': {
            borderColor: '#448aff',

            h4: {
              backgroundColor: '#448aff1a',
            },
          },

          '&.danger': {
            borderColor: '#ff1744',

            h4: {
              backgroundColor: '#ff17441a',
            },
          },
        },
      })}
    >
      <Markdown
        rehypePlugins={[rehypeRaw, remarkGfm]}
        components={{
          // eslint-disable-next-line @typescript-eslint/no-unused-vars
          a: ({ ref, ...props }) => <Link {...props} />,
          i: ({ className }) => (className ? IconMap[className] || null : null),
          code: ({ children }) => <CodeBlock>{children}</CodeBlock>,
        }}
      >
        {content}
      </Markdown>
    </Stack>
  );
};
