import { Card, CardContent, Typography } from '@mui/material';
import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';

export const SlowQueryAnalysisBlock: FC<{ block: InvestigationBlock }> = ({ block }) => {
  const data = (block.dataJson || {}) as { content?: string; summary?: string; query?: string };
  const text = data.content ?? data.summary ?? block.title ?? '';
  return (
    <Card variant="outlined" sx={{ mb: 2, borderLeft: 4, borderLeftColor: 'warning.main' }}>
      {block.title && (
        <CardContent sx={{ pb: 0 }}>
          <Typography variant="subtitle1" fontWeight={600}>
            {block.title}
          </Typography>
        </CardContent>
      )}
      {data.query && (
        <CardContent sx={{ py: 0.5 }}>
          <Typography variant="caption" color="text.secondary" component="pre" sx={{ overflow: 'auto' }}>
            {data.query}
          </Typography>
        </CardContent>
      )}
      <CardContent>
        <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap' }}>
          {text || '(No analysis)'}
        </Typography>
      </CardContent>
    </Card>
  );
};
