import { Card, CardContent, Typography } from '@mui/material';
import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';

export const LogsViewBlock: FC<{ block: InvestigationBlock }> = ({ block }) => {
  const data = (block.dataJson || {}) as { content?: string; lines?: string[] };
  const text = data.content ?? (Array.isArray(data.lines) ? data.lines.join('\n') : '') ?? block.title ?? '';
  return (
    <Card variant="outlined" sx={{ mb: 2, bgcolor: 'grey.50' }}>
      {block.title && (
        <CardContent sx={{ pb: 0 }}>
          <Typography variant="subtitle1" fontWeight={600}>
            {block.title}
          </Typography>
        </CardContent>
      )}
      <CardContent>
        <Typography variant="body2" component="pre" sx={{ whiteSpace: 'pre-wrap', fontFamily: 'monospace', maxHeight: 400, overflow: 'auto' }}>
          {text || '(No logs)'}
        </Typography>
      </CardContent>
    </Card>
  );
};
