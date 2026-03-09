import {
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  CircularProgress,
  Collapse,
  FormControlLabel,
  FormGroup,
  IconButton,
  Stack,
  Typography,
} from '@mui/material';
import ExpandLess from '@mui/icons-material/ExpandLess';
import ExpandMore from '@mui/icons-material/ExpandMore';
import { FC, useState, useEffect } from 'react';
import { getAdreAlerts, adreInvestigateStream } from 'api/adre';
import { useSnackbar } from 'notistack';

interface AlertItem {
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  fingerprint?: string;
  [k: string]: unknown;
}

/** Unique key for an alert (fingerprint if set, else label+index). */
function getAlertKey(a: AlertItem, index: number): string {
  const fp = String(a.fingerprint ?? a.labels?.alertname ?? '');
  return a.fingerprint ? fp : `${fp || 'alert'}-${index}`;
}

export const AdreAlertsPanel: FC = () => {
  const { enqueueSnackbar } = useSnackbar();
  const [alerts, setAlerts] = useState<AlertItem[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [result, setResult] = useState('');
  const [reasoning, setReasoning] = useState('');
  const [loading, setLoading] = useState(false);
  const [loadingAlerts, setLoadingAlerts] = useState(true);
  const [reasoningExpanded, setReasoningExpanded] = useState(false);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      try {
        const data = (await getAdreAlerts()) as {
          data?: { alerts?: AlertItem[] };
          alerts?: AlertItem[];
        };
        const list = data?.data?.alerts ?? data?.alerts ?? [];
        const arr = Array.isArray(list) ? list : [];
        if (!cancelled) setAlerts(arr);
      } catch (err) {
        if (!cancelled) {
          setAlerts([]);
          enqueueSnackbar('Failed to load alerts', { variant: 'warning' });
        }
      } finally {
        if (!cancelled) setLoadingAlerts(false);
      }
    };
    load();
    return () => { cancelled = true; };
  }, [enqueueSnackbar]);

  const toggle = (key: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const buildInvestigatePayload = (): {
    source: string;
    title: string;
    description: string;
    subject?: Record<string, unknown>;
    context?: Record<string, unknown>;
  } => {
    const items = alerts.filter((a, i) => selected.has(getAlertKey(a, i)));
    if (items.length === 0) {
      return {
        source: 'prometheus',
        title: 'No alerts selected',
        description: 'Select one or more alerts to investigate.',
        subject: {},
        context: {},
      };
    }
    const titles = items.map((a) => a.labels?.alertname ?? a.annotations?.summary ?? 'Alert').filter(Boolean);
    const descs = items.map((a) => a.annotations?.description ?? a.annotations?.summary ?? JSON.stringify(a.labels ?? {})).filter(Boolean);
    const alertsPayload = items.map((a) => ({ labels: a.labels, annotations: a.annotations }));
    return {
      source: 'prometheus',
      title: titles[0] ?? 'Alerts',
      description: descs.join('\n\n') || JSON.stringify(items),
      subject: { alerts: alertsPayload },
      context: { source: 'prometheus', alertCount: items.length },
    };
  };

  const handleInvestigate = async () => {
    const payload = buildInvestigatePayload();
    setLoading(true);
    setResult('');
    setReasoning('');
    try {
      await adreInvestigateStream(
        { ...payload, stream: true },
        (contentChunk, reasoningChunk) => {
          if (contentChunk) setResult((prev) => prev + contentChunk);
          if (reasoningChunk) setReasoning((prev) => prev + reasoningChunk);
        }
      );
    } catch (err) {
      enqueueSnackbar(
        err instanceof Error ? err.message : 'Investigation failed',
        { variant: 'error' }
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card variant="outlined" sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <CardContent sx={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}>
        <Typography variant="h6" gutterBottom>
          Investigate current alerts
        </Typography>
        <Stack gap={1} sx={{ flex: 1, minHeight: 0 }}>
          {loadingAlerts ? (
            <Typography color="text.secondary">Loading alerts...</Typography>
          ) : alerts.length === 0 ? (
            <Typography color="text.secondary">
              No firing alerts. Alerts from Alertmanager will appear here.
            </Typography>
          ) : (
            <FormGroup>
              <Box sx={{ maxHeight: 150, overflow: 'auto' }}>
                {alerts.map((a, index) => {
                  const key = getAlertKey(a, index);
                  const label = (a.labels?.alertname ?? a.annotations?.summary) ?? (a.fingerprint ? String(a.fingerprint) : key);
                  return (
                    <FormControlLabel
                      key={key}
                      control={
                        <Checkbox
                          checked={selected.has(key)}
                          onChange={() => toggle(key)}
                        />
                      }
                      label={label}
                    />
                  );
                })}
              </Box>
            </FormGroup>
          )}
          <Button
            variant="contained"
            onClick={handleInvestigate}
            disabled={loading || selected.size === 0}
          >
            Investigate
          </Button>
          {(loading || result || reasoning) && (
            <Box
              sx={{
                flex: 1,
                minHeight: 100,
                maxHeight: 250,
                overflow: 'auto',
                p: 1,
                bgcolor: 'action.hover',
                borderRadius: 1,
              }}
            >
              {loading && (
                <Stack direction="row" alignItems="center" gap={1} sx={{ mb: 1 }}>
                  <CircularProgress size={16} />
                  <Typography variant="body2" color="text.secondary">
                    {result ? 'Streaming...' : 'Investigating...'}
                  </Typography>
                </Stack>
              )}
              {reasoning && (
                <>
                  <IconButton
                    size="small"
                    onClick={() => setReasoningExpanded((p) => !p)}
                    sx={{ p: 0, mr: 0.5 }}
                  >
                    {reasoningExpanded ? <ExpandLess /> : <ExpandMore />}
                  </IconButton>
                  <Typography
                    component="span"
                    variant="caption"
                    color="text.secondary"
                    sx={{ cursor: 'pointer' }}
                    onClick={() => setReasoningExpanded((p) => !p)}
                  >
                    Reasoning
                  </Typography>
                  <Collapse in={reasoningExpanded}>
                    <Typography
                      component="pre"
                      variant="body2"
                      color="text.secondary"
                      sx={{ whiteSpace: 'pre-wrap', fontStyle: 'italic', mt: 0.5 }}
                    >
                      {reasoning}
                    </Typography>
                  </Collapse>
                  {result && <Box sx={{ mt: 1 }} />}
                </>
              )}
              {result && (
                <Typography component="pre" sx={{ whiteSpace: 'pre-wrap', fontFamily: 'inherit' }}>
                  {result}
                </Typography>
              )}
            </Box>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
};
