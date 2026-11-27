import { Box, Typography } from '@mui/material';
import { FC, ReactNode } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import { getMarkdownComponents } from 'components/adre/adre-chat-markdown.helpers';
import {
  adreColors,
  answerBodySx,
  headingFont,
} from 'components/adre/adreAnswerSx';

/** A report section: its title, then its content in the Assistant's answer typography. */
export const BlockFrame: FC<{ title?: string; children: ReactNode }> = ({
  title,
  children,
}) => (
  <Box>
    {title && (
      <Typography
        component="h3"
        sx={{
          fontFamily: headingFont,
          fontSize: 18,
          fontWeight: 600,
          lineHeight: 1.4,
          color: adreColors.text,
          mb: 1.25,
          // Room for the section's move and delete buttons.
          pr: 14,
        }}
      >
        {title}
      </Typography>
    )}
    <Box sx={answerBodySx}>{children}</Box>
  </Box>
);

/** Markdown in a report section, rendered like an Assistant answer. */
export const BlockMarkdown: FC<{ text: string }> = ({ text }) => (
  <Markdown
    remarkPlugins={[remarkGfm]}
    rehypePlugins={[rehypeRaw]}
    components={getMarkdownComponents(text)}
  >
    {text}
  </Markdown>
);

/** Output kept as it was written: logs, query results, schemas. */
export const BlockPre: FC<{ children: ReactNode; caption?: boolean }> = ({
  children,
  caption,
}) => (
  <Box
    component="pre"
    sx={{
      m: 0,
      mb: caption ? 1 : 0,
      p: 1.75,
      maxHeight: caption ? 160 : 400,
      overflow: 'auto',
      borderRadius: '10px',
      border: `1px solid ${adreColors.border}`,
      bgcolor: '#0D0C0F',
      color: caption ? adreColors.secondary : '#E6E4EA',
      fontFamily: 'Roboto Mono, monospace',
      fontSize: 13,
      lineHeight: 1.6,
      whiteSpace: 'pre-wrap',
      overflowWrap: 'anywhere',
    }}
  >
    {children}
  </Box>
);

/** What a section says when it has nothing to show. */
export const BlockEmpty: FC<{ children: ReactNode }> = ({ children }) => (
  <Typography sx={{ fontSize: 14, color: adreColors.muted }}>
    {children}
  </Typography>
);
