import {
  Box,
  Button,
  Checkbox,
  FormControlLabel,
  FormGroup,
  Stack,
  Typography,
} from '@mui/material';
import { FC, useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { getAdreAlerts, getAlertMetadataFromLabels } from 'api/adre';
import { useCreateInvestigation } from 'hooks/api/useInvestigations';
import { useSnackbar } from 'notistack';
import { PMM_NEW_NAV_PATH } from 'lib/constants';

export interface AlertItem {
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  fingerprint?: string;
  [k: string]: unknown;
}

export interface AdreAlertsPanelProps {
  alerts?: AlertItem[];
}

/** Unique key for an alert (fingerprint if set, else label+index). */
function getAlertKey(a: AlertItem, index: number): string {
  const fp = String(a.fingerprint ?? a.labels?.alertname ?? '');
  return a.fingerprint ? fp : `${fp || 'alert'}-${index}`;
}

export const AdreAlertsPanel: FC<AdreAlertsPanelProps> = ({ alerts: alertsProp }) => {
  const navigate = useNavigate();
  const { enqueueSnackbar } = useSnackbar();
  const createMutation = useCreateInvestigation();
  const [internalAlerts, setInternalAlerts] = useState<AlertItem[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [loadingAlerts, setLoadingAlerts] = useState(!alertsProp);

  const alerts = alertsProp ?? internalAlerts;

  useEffect(() => {
    if (alertsProp != null) return;
    let cancelled = false;
    const load = async () => {
      try {
        const data = (await getAdreAlerts()) as {
          data?: { alerts?: AlertItem[] };
          alerts?: AlertItem[];
        };
        const list = data?.data?.alerts ?? data?.alerts ?? [];
        const arr = Array.isArray(list) ? list : [];
        if (!cancelled) setInternalAlerts(arr);
      } catch (err) {
        if (!cancelled) {
          setInternalAlerts([]);
          enqueueSnackbar('Failed to load alerts', { variant: 'warning' });
        }
      } finally {
        if (!cancelled) setLoadingAlerts(false);
      }
    };
    load();
    return () => { cancelled = true; };
  }, [alertsProp, enqueueSnackbar]);

  const toggle = (key: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const handleStartInvestigation = () => {
    const items = alerts.filter((a, i) => selected.has(getAlertKey(a, i)));
    if (items.length === 0) return;
    const titles = items
      .map((a) => a.labels?.alertname ?? a.annotations?.summary ?? 'Alert')
      .filter(Boolean);
    const title = `Alert: ${titles[0] ?? 'Alerts'}`;
    const sourceRef = items
      .map((a) => a.fingerprint ?? getAlertKey(a, alerts.indexOf(a)))
      .filter(Boolean)
      .join(',');
    const first = items[0];
    const { nodeName, serviceName, clusterName } = getAlertMetadataFromLabels(first?.labels);
    createMutation.mutate(
      {
        title,
        sourceType: 'alert',
        sourceRef: sourceRef || undefined,
        ...(nodeName && { nodeName }),
        ...(serviceName && { serviceName }),
        ...(clusterName && { clusterName }),
        alertSnapshot: items,
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
    <Box
      sx={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        p: 1.5,
        borderLeft: 1,
        borderColor: 'rgba(255,255,255,0.12)',
      }}
    >
      <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
        Firing alerts ({alerts.length})
      </Typography>
      <Stack gap={1} sx={{ flex: 1, minHeight: 0 }}>
        {loadingAlerts ? (
          <Typography variant="caption" color="text.secondary">
            Loading...
          </Typography>
        ) : alerts.length === 0 ? (
          <Typography variant="caption" color="text.secondary">
            No firing alerts.
          </Typography>
        ) : (
          <FormGroup sx={{ gap: 0 }}>
            <Box sx={{ maxHeight: 150, overflow: 'auto' }}>
              {alerts.map((a, index) => {
                const key = getAlertKey(a, index);
                const label = (a.labels?.alertname ?? a.annotations?.summary) ?? (a.fingerprint ? String(a.fingerprint) : key);
                return (
                  <FormControlLabel
                    key={key}
                    control={
                      <Checkbox
                        size="small"
                        checked={selected.has(key)}
                        onChange={() => toggle(key)}
                      />
                    }
                    label={
                      <Typography variant="caption" sx={{ fontSize: '0.75rem' }}>
                        {String(label).length > 40 ? `${String(label).slice(0, 40)}…` : label}
                      </Typography>
                    }
                    sx={{ m: 0, py: 0.25 }}
                  />
                );
              })}
            </Box>
          </FormGroup>
        )}
        <Button
          variant="contained"
          size="small"
          onClick={handleStartInvestigation}
          disabled={createMutation.isPending || selected.size === 0}
          sx={{ mt: 0.5 }}
        >
          {createMutation.isPending ? 'Creating...' : 'Start investigation'}
        </Button>
      </Stack>
    </Box>
  );
};
