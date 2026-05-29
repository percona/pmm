import { Stack, Typography } from '@mui/material';
import { FC } from 'react';
import { useQanPanelState } from '../../hooks/useQanPanelState';
import { QanMetricsTab } from './QanMetricsTab';

/** Details tab combines metrics summary + metadata (Grafana Details tab). */
export const QanDetailsTabPanel: FC = () => {
  const state = useQanPanelState();

  return (
    <Stack spacing={2}>
      {state.fingerprint ? (
        <Typography variant="body2" sx={{ fontFamily: 'monospace', whiteSpace: 'pre-wrap' }}>
          {state.fingerprint}
        </Typography>
      ) : null}
      <QanMetricsTab />
    </Stack>
  );
};
