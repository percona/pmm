import { Card, CardContent, Typography } from '@mui/material';
import { FC } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import type { InvestigationBlock } from 'api/investigations';
import { getMarkdownComponents } from 'components/adre/adre-chat-markdown.helpers';

export const SummaryBlock: FC<{ block: InvestigationBlock }> = ({ block }) => {
  const data = block.dataJson as { summary?: string } | undefined;
  const text = data?.summary ?? block.title ?? '';
  return (
    <Card variant="outlined" sx={{ mb: 2 }}>
      {block.title && (
        <CardContent sx={{ pb: 0 }}>
          <Typography variant="subtitle1" fontWeight={600}>
            {block.title}
          </Typography>
        </CardContent>
      )}
      <CardContent>
        <Typography component="div" variant="body2">
          <Markdown
            remarkPlugins={[remarkGfm]}
            rehypePlugins={[rehypeRaw]}
            components={getMarkdownComponents(text)}
          >
            {text}
          </Markdown>
        </Typography>
      </CardContent>
    </Card>
  );
};
