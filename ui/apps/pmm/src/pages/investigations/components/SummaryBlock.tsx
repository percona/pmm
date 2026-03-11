import { Card, CardContent, Typography } from '@mui/material';
import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';

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
        <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap' }}>
          {text}
        </Typography>
      </CardContent>
    </Card>
  );
};
