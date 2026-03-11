import { Card, CardContent, List, ListItem, ListItemText, Typography } from '@mui/material';
import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';

export const RemediationStepsBlock: FC<{ block: InvestigationBlock }> = ({ block }) => {
  const data = (block.dataJson || {}) as { steps?: string[]; content?: string };
  const steps = Array.isArray(data.steps) ? data.steps : data.content ? [data.content] : [];
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
        {steps.length > 0 ? (
          <List dense disablePadding>
            {steps.map((step, i) => (
              <ListItem key={i} disablePadding sx={{ alignItems: 'flex-start' }}>
                <ListItemText primary={`${i + 1}. ${step}`} primaryTypographyProps={{ variant: 'body2' }} />
              </ListItem>
            ))}
          </List>
        ) : (
          <Typography variant="body2" color="text.secondary">
            (No steps)
          </Typography>
        )}
      </CardContent>
    </Card>
  );
};
