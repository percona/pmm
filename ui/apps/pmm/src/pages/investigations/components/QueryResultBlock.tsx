import { Card, CardContent, Typography } from '@mui/material';
import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';

export const QueryResultBlock: FC<{ block: InvestigationBlock }> = ({ block }) => {
  const data = (block.dataJson || {}) as { query?: string; result?: string; rows?: unknown[] };
  const text = data.result ?? (Array.isArray(data.rows) ? JSON.stringify(data.rows, null, 2) : '') ?? block.title ?? '';
  return (
    <Card variant="outlined" sx={{ mb: 2 }}>
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
        <Typography variant="body2" component="pre" sx={{ whiteSpace: 'pre-wrap', overflow: 'auto', maxHeight: 400 }}>
          {text || '(No result)'}
        </Typography>
      </CardContent>
    </Card>
  );
};
