import { Box, Button, Checkbox, Link, Stack, Typography } from '@mui/material';
import { FC, Fragment, useState, useEffect } from 'react';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import { getAdreAlerts, getAlertMetadataFromLabels } from 'api/adre';
import {
  useCreateInvestigation,
  useSimilarToIncident,
} from 'hooks/api/useInvestigations';
import { formatSeenDate } from 'pages/investigations/components/seenBefore.utils';
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

/** The color of an alert's severity label. */
const severityColor = (severity: string) => {
  const s = severity.toLowerCase();
  if (s === 'critical' || s === 'error' || s === 'emergency' || s === 'alert') {
    return '#FCA5A5';
  }
  if (s === 'warning') return '#FCD34D';
  return '#A9A7B0';
};

/** The past investigation an alert looks like, if any. */
const AlertSeenBefore: FC<{ labels?: Record<string, string> }> = ({
  labels,
}) => {
  const { nodeName, serviceName, clusterName } =
    getAlertMetadataFromLabels(labels);
  const query = {
    alertname: labels?.alertname,
    service_name: serviceName,
    node_name: nodeName,
    cluster: clusterName,
  };
  const { data = [] } = useSimilarToIncident(
    query,
    !!(query.alertname || query.service_name || query.node_name)
  );
  const [top] = data;
  if (!top) return null;
  return (
    <Typography sx={{ fontSize: 12, color: '#A9A7B0' }}>
      Seen before:{' '}
      <Link
        component={RouterLink}
        to={`${PMM_NEW_NAV_PATH}/investigations/${top.id}`}
        sx={{ color: '#CFCBFF' }}
      >
        {top.title}
      </Link>
      {` · ${formatSeenDate(top.createdAt)}`}
      {data.length > 1 ? ` · +${data.length - 1} more` : ''}
    </Typography>
  );
};

/** Unique key for an alert (fingerprint if set, else label+index). */
function getAlertKey(a: AlertItem, index: number): string {
  const fp = String(a.fingerprint ?? a.labels?.alertname ?? '');
  return a.fingerprint ? fp : `${fp || 'alert'}-${index}`;
}

export const AdreAlertsPanel: FC<AdreAlertsPanelProps> = ({
  alerts: alertsProp,
}) => {
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
    return () => {
      cancelled = true;
    };
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
    const { nodeName, serviceName, clusterName } = getAlertMetadataFromLabels(
      first?.labels
    );
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
            err instanceof Error
              ? err.message
              : 'Failed to create investigation',
            { variant: 'error' }
          );
        },
      }
    );
  };

  const selectedCount = selected.size;
  return (
    <Stack component="section" aria-label="Firing alerts" gap={1.25}>
      <Stack direction="row" alignItems="center" justifyContent="space-between">
        <Typography
          component="h2"
          sx={{
            fontFamily: 'Poppins, sans-serif',
            fontSize: 14,
            fontWeight: 600,
          }}
        >
          Firing alerts
        </Typography>
        {alerts.length > 0 && (
          <Box
            component="span"
            sx={{
              px: 1,
              borderRadius: 999,
              bgcolor: 'rgba(248,113,113,0.14)',
              color: '#FCA5A5',
              fontSize: 12,
              fontWeight: 500,
            }}
          >
            {alerts.length}
          </Box>
        )}
      </Stack>
      {loadingAlerts ? (
        <Typography variant="caption" color="text.secondary">
          Loading...
        </Typography>
      ) : alerts.length === 0 ? (
        <Typography variant="caption" color="text.secondary">
          No firing alerts.
        </Typography>
      ) : (
        alerts.map((a, index) => {
          const key = getAlertKey(a, index);
          const label =
            a.labels?.alertname ??
            a.annotations?.summary ??
            (a.fingerprint ? String(a.fingerprint) : key);
          const { nodeName, serviceName } = getAlertMetadataFromLabels(
            a.labels
          );
          const severity = a.labels?.severity ?? a.labels?.Severity ?? '';
          const fields = [
            ['Node', nodeName],
            ['Service', serviceName],
            ['Database', a.labels?.schema ?? a.labels?.database],
            ['Query ID', a.labels?.query_id ?? a.labels?.queryid],
            ['Fingerprint', a.labels?.fingerprint ?? a.fingerprint],
          ].filter((f): f is [string, string] => !!f[1]);
          const checked = selected.has(key);
          return (
            <Box
              key={key}
              component="label"
              sx={{
                display: 'flex',
                gap: 1,
                p: 1.5,
                pl: 0.75,
                borderRadius: '12px',
                bgcolor: '#1F1E23',
                border: 1,
                borderColor: checked
                  ? 'rgba(182,178,255,0.45)'
                  : 'rgba(255,255,255,0.08)',
                cursor: 'pointer',
              }}
            >
              <Checkbox
                size="small"
                checked={checked}
                onChange={() => toggle(key)}
                sx={{ p: 0.5, alignSelf: 'flex-start' }}
              />
              <Stack gap={1} sx={{ flex: 1, minWidth: 0 }}>
                <Stack
                  direction="row"
                  alignItems="center"
                  justifyContent="space-between"
                  gap={1}
                >
                  <Typography
                    sx={{
                      fontSize: 14,
                      fontWeight: 500,
                      overflowWrap: 'anywhere',
                    }}
                  >
                    {label}
                  </Typography>
                  {severity && (
                    <Stack
                      direction="row"
                      alignItems="center"
                      gap={0.75}
                      sx={{
                        flexShrink: 0,
                        fontSize: 12,
                        color: severityColor(severity),
                      }}
                    >
                      <Box
                        component="span"
                        sx={{
                          width: 7,
                          height: 7,
                          borderRadius: 999,
                          bgcolor: 'currentColor',
                        }}
                      />
                      {severity}
                    </Stack>
                  )}
                </Stack>
                {fields.length > 0 && (
                  <Box
                    component="dl"
                    sx={{
                      m: 0,
                      display: 'grid',
                      gridTemplateColumns: '76px minmax(0, 1fr)',
                      rowGap: 0.5,
                      fontSize: 12,
                      '& dt': { color: '#8E8C95' },
                      '& dd': {
                        m: 0,
                        color: '#D9D7DE',
                        overflowWrap: 'anywhere',
                      },
                    }}
                  >
                    {fields.map(([name, value]) => (
                      <Fragment key={name}>
                        <dt>{name}</dt>
                        <Box
                          component="dd"
                          sx={
                            name === 'Fingerprint'
                              ? {
                                  fontFamily: 'Roboto Mono, monospace',
                                  color: '#A9A7B0 !important',
                                  overflow: 'hidden',
                                  textOverflow: 'ellipsis',
                                  whiteSpace: 'nowrap',
                                }
                              : undefined
                          }
                          title={name === 'Fingerprint' ? value : undefined}
                        >
                          {value}
                        </Box>
                      </Fragment>
                    ))}
                  </Box>
                )}
                <AlertSeenBefore labels={a.labels} />
              </Stack>
            </Box>
          );
        })
      )}
      {alerts.length > 0 && (
        <Button
          variant="outlined"
          onClick={handleStartInvestigation}
          disabled={createMutation.isPending || selectedCount === 0}
        >
          {createMutation.isPending
            ? 'Creating...'
            : selectedCount === 0
              ? 'Start investigation'
              : `Start investigation (${selectedCount} ${selectedCount === 1 ? 'alert' : 'alerts'})`}
        </Button>
      )}
    </Stack>
  );
};
