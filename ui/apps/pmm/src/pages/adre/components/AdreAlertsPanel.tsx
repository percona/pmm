import {
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  FormControlLabel,
  FormGroup,
  Stack,
  Typography,
} from '@mui/material';
import { FC, useState, useEffect } from 'react';
import { getAdreAlerts, adreInvestigate, adreInvestigateStream } from 'api/adre';
import { useSnackbar } from 'notistack';

interface AlertItem {
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  fingerprint?: string;
  [k: string]: unknown;
}

export const AdreAlertsPanel: FC = () => {
  const { enqueueSnackbar } = useSnackbar();
  const [alerts, setAlerts] = useState<AlertItem[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [result, setResult] = useState('');
  const [loading, setLoading] = useState(false);
  const [loadingAlerts, setLoadingAlerts] = useState(true);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      try {
        const data = (await getAdreAlerts()) as { data?: { alerts?: AlertItem[] }; status?: string };
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

  const toggle = (fp: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(fp)) next.delete(fp);
      else next.add(fp);
      return next;
    });
  };

  const buildInvestigatePayload = (): { source: string; title: string; description: string; subject?: unknown } => {
    const items = alerts.filter((a) => selected.has(String(a.fingerprint ?? a.labels?.alertname ?? '')));
    if (items.length === 0) {
      return {
        source: 'prometheus',
        title: 'No alerts selected',
        description: 'Select one or more alerts to investigate.',
      };
    }
    const titles = items.map((a) => a.labels?.alertname ?? a.annotations?.summary ?? 'Alert').filter(Boolean);
    const descs = items.map((a) => a.annotations?.description ?? a.annotations?.summary ?? JSON.stringify(a.labels ?? {})).filter(Boolean);
    return {
      source: 'prometheus',
      title: titles[0] ?? 'Alerts',
      description: descs.join('\n\n') || JSON.stringify(items),
      subject: items.map((a) => ({ labels: a.labels, annotations: a.annotations })),
    };
  };

  const handleInvestigate = async () => {
    const payload = buildInvestigatePayload();
    setLoading(true);
    setResult('');
    try {
      await adreInvestigateStream(
        { ...payload, stream: true },
        (chunk) => setResult((prev) => prev + chunk)
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
                {alerts.map((a) => {
                  const fp = String(a.fingerprint ?? a.labels?.alertname ?? '');
                  const label = a.labels?.alertname ?? a.annotations?.summary ?? fp || 'Alert';
                  return (
                    <FormControlLabel
                      key={fp}
                      control={
                        <Checkbox
                          checked={selected.has(fp)}
                          onChange={() => toggle(fp)}
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
          {result && (
            <Box
              sx={{
                flex: 1,
                minHeight: 100,
                maxHeight: 200,
                overflow: 'auto',
                p: 1,
                bgcolor: 'action.hover',
                borderRadius: 1,
              }}
            >
              <Typography component="pre" sx={{ whiteSpace: 'pre-wrap', fontFamily: 'inherit' }}>
                {result}
              </Typography>
            </Box>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
};
