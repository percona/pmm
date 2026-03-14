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
import { useNavigate } from 'react-router-dom';
import { getAdreAlerts } from 'api/adre';
import { useCreateInvestigation } from 'hooks/api/useInvestigations';
import { useSnackbar } from 'notistack';
import { PMM_NEW_NAV_PATH } from 'lib/constants';

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
  const navigate = useNavigate();
  const { enqueueSnackbar } = useSnackbar();
  const createMutation = useCreateInvestigation();
  const [alerts, setAlerts] = useState<AlertItem[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [loadingAlerts, setLoadingAlerts] = useState(true);

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

  const extractFromLabels = (labels?: Record<string, string>) => {
    if (!labels) return {};
    return {
      nodeName: labels.node ?? labels.instance ?? labels.nodename ?? undefined,
      serviceName: labels.service_name ?? labels.service ?? undefined,
      clusterName: labels.cluster ?? undefined,
    };
  };

  const handleStartInvestigation = () => {
    const items = alerts.filter((a, i) => selected.has(getAlertKey(a, i)));
    if (items.length === 0) return;
    const titles = items
      .map((a) => a.labels?.alertname ?? a.annotations?.summary ?? 'Alert')
      .filter(Boolean);
    const title = titles[0] ?? 'Alerts';
    const sourceRef = items
      .map((a) => a.fingerprint ?? getAlertKey(a, alerts.indexOf(a)))
      .filter(Boolean)
      .join(',');
    const first = items[0];
    const { nodeName, serviceName, clusterName } = extractFromLabels(first?.labels);
    createMutation.mutate(
      {
        title,
        sourceType: 'alert',
        sourceRef: sourceRef || undefined,
        ...(nodeName && { nodeName }),
        ...(serviceName && { serviceName }),
        ...(clusterName && { clusterName }),
      },
      {
        onSuccess: (inv) => {
          navigate(`${PMM_NEW_NAV_PATH}/investigations/${inv.id}`);
        },
        onError: (err) => {
          enqueueSnackbar(
            err instanceof Error ? err.message : 'Failed to create investigation',
            { variant: 'error' }
          );
        },
      }
    );
  };

  return (
    <Card variant="outlined" sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <CardContent sx={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}>
        <Typography variant="h6" gutterBottom>
          Firing alerts
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
            onClick={handleStartInvestigation}
            disabled={createMutation.isPending || selected.size === 0}
          >
            {createMutation.isPending ? 'Creating...' : 'Start investigation'}
          </Button>
        </Stack>
      </CardContent>
    </Card>
  );
};
