import { Card, CardContent, Typography } from '@mui/material';
import { FC } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import type { InvestigationBlock } from 'api/investigations';
import { getMarkdownComponents } from 'components/adre/adre-chat-markdown.helpers';

export const FindingBlock: FC<{ block: InvestigationBlock }> = ({ block }) => {
  const data = (block.dataJson || {}) as { content?: string; summary?: string };
  const text = data.content ?? data.summary ?? block.title ?? '';
  return (
    <Card variant="outlined" sx={{ mb: 2, borderLeft: 4, borderLeftColor: 'info.main' }}>
      {block.title && (
        <CardContent sx={{ pb: 0 }}>
          <Typography variant="subtitle1" fontWeight={600}>
            {block.title}
          </Typography>
        </CardContent>
      )}
      <CardContent>
        {text ? (
          <Typography component="div" variant="body2">
            <Markdown
              remarkPlugins={[remarkGfm]}
              rehypePlugins={[rehypeRaw]}
              components={getMarkdownComponents(text)}
            >
              {text}
            </Markdown>
          </Typography>
        ) : (
          <Typography variant="body2" color="text.secondary">
            (No content)
          </Typography>
        )}
      </CardContent>
    </Card>
  );
};
