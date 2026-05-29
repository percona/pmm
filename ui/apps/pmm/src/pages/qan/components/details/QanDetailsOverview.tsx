import {
  Alert,
  Box,
  Button,
  Stack,
  Typography,
} from '@mui/material';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import { FC } from 'react';
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { useQanPanelActions, useQanPanelState } from '../../hooks/useQanPanelState';
import { QanMetricsTab } from './QanMetricsTab';

const ANOMALY_GRADIENT =
  'linear-gradient(135deg, rgba(18, 122, 232, 0.85) 0%, rgba(120, 60, 200, 0.9) 55%, rgba(180, 80, 220, 0.85) 100%)';

export const QanDetailsOverview: FC = () => {
  const state = useQanPanelState();
  const actions = useQanPanelActions();

  return (
    <Stack spacing={2} data-testid="qan-details-overview">
      <Stack
        direction={{ xs: 'column', md: 'row' }}
        spacing={2}
        sx={{ alignItems: 'stretch' }}
      >
        <Box
          sx={{
            flex: 1,
            minWidth: 0,
            bgcolor: 'background.paper',
            border: 1,
            borderColor: 'divider',
            borderRadius: 0.5,
            p: 2,
            position: 'relative',
          }}
        >
          {state.fingerprint ? (
            <SyntaxHighlighter language="text" content={state.fingerprint} showCopyButton />
          ) : (
            <Typography variant="body2" color="text.secondary">
              No fingerprint available for this query.
            </Typography>
          )}
        </Box>
        <Box
          sx={{
            flex: { xs: '1 1 auto', md: '0 0 320px' },
            borderRadius: 0.5,
            p: 2,
            color: 'common.white',
            background: ANOMALY_GRADIENT,
            boxShadow: 3,
            display: 'flex',
            flexDirection: 'column',
            gap: 1,
          }}
        >
          <Stack direction="row" alignItems="center" spacing={0.5}>
            <WarningAmberIcon sx={{ color: 'warning.light', fontSize: 20 }} />
            <Typography
              variant="overline"
              sx={{
                fontWeight: 700,
                letterSpacing: '0.08em',
                color: 'warning.light',
              }}
            >
              Anomaly detected
            </Typography>
          </Stack>
          <Typography variant="h6" sx={{ fontWeight: 600, lineHeight: 1.25 }}>
            Review query performance
          </Typography>
          <Typography variant="body2" sx={{ opacity: 0.95, lineHeight: 1.5 }}>
            Run batch analysis for this fingerprint or ask the AI aside. PMM surfaces
            advisory insights only — copy SQL and apply changes manually.
          </Typography>
          <Button
            size="small"
            variant="contained"
            startIcon={<AutoAwesomeIcon />}
            onClick={() => actions.setTab('aiInsights')}
            sx={{
              alignSelf: 'flex-start',
              mt: 0.5,
              bgcolor: 'rgba(255,255,255,0.15)',
              color: 'common.white',
              '&:hover': { bgcolor: 'rgba(255,255,255,0.25)' },
            }}
          >
            Get AI Insights
          </Button>
        </Box>
      </Stack>
      <Box>
        <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 600 }}>
          Metrics
        </Typography>
        <QanMetricsTab />
      </Box>
      <Alert severity="info" sx={{ display: { xs: 'flex', md: 'none' } }}>
        Use the Get AI Insights tab or the chat aside for advisory analysis.
      </Alert>
    </Stack>
  );
};
