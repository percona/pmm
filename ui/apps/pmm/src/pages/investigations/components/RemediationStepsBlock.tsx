import { Card, CardContent, Typography } from '@mui/material';
import { FC } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import type { InvestigationBlock } from 'api/investigations';
import { getMarkdownComponents } from 'components/adre/adre-chat-markdown';

export const RemediationStepsBlock: FC<{ block: InvestigationBlock }> = ({ block }) => {
  const data = (block.dataJson || {}) as { steps?: string[]; content?: string };
  const steps = Array.isArray(data.steps) ? data.steps : data.content ? [data.content] : [];
  const combined = steps.map((step, i) => `${i + 1}. ${step}`).join('\n');
  return (
    <Card variant="outlined" sx={{ mb: 2, borderLeft: 4, borderLeftColor: 'success.main' }}>
      {block.title && (
        <CardContent sx={{ pb: 0 }}>
          <Typography variant="subtitle1" fontWeight={600}>
            {block.title}
          </Typography>
        </CardContent>
      )}
      <CardContent>
        {combined ? (
          <Typography component="div" variant="body2">
            <Markdown
              remarkPlugins={[remarkGfm]}
              rehypePlugins={[rehypeRaw]}
              components={getMarkdownComponents(combined)}
            >
              {combined}
            </Markdown>
          </Typography>
        ) : (
          <Typography variant="body2" color="text.secondary">
            (No steps)
          </Typography>
        )}
      </CardContent>
    </Card>
  );
};
