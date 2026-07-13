import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Stack,
  Typography,
} from '@mui/material';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import { FC } from 'react';
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { useQanPanelActions, useQanPanelState } from '../../hooks/useQanPanelState';
import { QanMetricsTab } from './QanMetricsTab';

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
        <Card
          variant="outlined"
          sx={{
            flex: { xs: '1 1 auto', md: '0 0 280px' },
            bgcolor: 'action.hover',
          }}
        >
          <CardContent>
            <Stack spacing={1}>
              <Stack direction="row" alignItems="center" spacing={0.5}>
                <AutoAwesomeIcon color="primary" fontSize="small" />
                <Typography variant="subtitle2">AI insights</Typography>
              </Stack>
              <Typography variant="body2" color="text.secondary">
                Run batch analysis for this fingerprint or ask the chat aside.
                Recommendations are advisory only — copy SQL and apply manually.
              </Typography>
              <Button
                size="small"
                variant="outlined"
                startIcon={<AutoAwesomeIcon />}
                onClick={() => actions.setTab('aiInsights')}
              >
                Get AI Insights
              </Button>
            </Stack>
          </CardContent>
        </Card>
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
