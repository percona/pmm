import { Card, CardContent, Typography } from '@mui/material';
import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';

export const MarkdownBlock: FC<{ block: InvestigationBlock }> = ({ block }) => {
  const data = block.dataJson as { content?: string } | undefined;
  const content = data?.content ?? '';
  return (
    <Card variant="outlined" sx={{ mb: 2, bgcolor: 'action.hover' }}>
      {block.title && (
        <CardContent sx={{ pb: 0 }}>
          <Typography variant="subtitle1" fontWeight={600}>
            {block.title}
          </Typography>
        </CardContent>
      )}
      <CardContent>
        <Typography
          component="div"
          variant="body2"
          sx={{ whiteSpace: 'pre-wrap' }}
        >
          {content}
        </Typography>
      </CardContent>
    </Card>
  );
};
