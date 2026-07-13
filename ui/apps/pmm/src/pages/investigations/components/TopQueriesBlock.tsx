import { Card, CardContent, Typography } from '@mui/material';
import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';

export const TopQueriesBlock: FC<{ block: InvestigationBlock }> = ({ block }) => {
  const data = (block.dataJson || {}) as { content?: string; queries?: unknown[] };
  const text = data.content ?? (Array.isArray(data.queries) ? JSON.stringify(data.queries, null, 2) : '') ?? block.title ?? '';
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
        <Typography variant="body2" component="pre" sx={{ whiteSpace: 'pre-wrap', overflow: 'auto', maxHeight: 400 }}>
          {text || '(No queries)'}
        </Typography>
      </CardContent>
    </Card>
  );
};
