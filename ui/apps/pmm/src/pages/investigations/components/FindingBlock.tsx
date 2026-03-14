import { Card, CardContent, Typography } from '@mui/material';
import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';

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
        <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap' }}>
          {text || '(No content)'}
        </Typography>
      </CardContent>
    </Card>
  );
};
