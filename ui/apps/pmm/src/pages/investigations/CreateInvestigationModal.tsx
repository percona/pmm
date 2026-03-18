import {
  Box,
  Button,
  Checkbox,
  DialogActions,
  DialogContent,
  FormControl,
  FormControlLabel,
  FormGroup,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker';
import { Dialog, DialogTitle } from '@percona/percona-ui';
import { FC, useEffect, useState } from 'react';
import { getAdreAlerts, getAlertMetadataFromLabels } from 'api/adre';
import type { CreateInvestigationBody } from 'api/investigations';

interface AlertItem {
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  fingerprint?: string;
  [k: string]: unknown;
}

function getAlertKey(a: AlertItem, index: number): string {
  const fp = String(a.fingerprint ?? a.labels?.alertname ?? '');
  return a.fingerprint ? fp : `${fp || 'alert'}-${index}`;
}

function parseISODate(s: string): Date | null {
  if (!s.trim()) return null;
  const d = new Date(s);
  return Number.isNaN(d.getTime()) ? null : d;
}

export interface CreateInvestigationModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (body: CreateInvestigationBody) => void;
  isPending?: boolean;
  /** Prefill from URL params or alert context */
  initial?: Partial<CreateInvestigationBody>;
}

export const CreateInvestigationModal: FC<CreateInvestigationModalProps> = ({
  open,
  onClose,
  onSubmit,
  isPending = false,
  initial,
}) => {
  const [summary, setSummary] = useState(initial?.summary ?? '');
  const [title, setTitle] = useState(initial?.title ?? '');
  const [sourceType, setSourceType] = useState<string>(initial?.sourceType ?? 'manual');
  const [sourceRef, setSourceRef] = useState(initial?.sourceRef ?? '');
  const [timeFrom, setTimeFrom] = useState(initial?.timeFrom ?? '');
  const [timeTo, setTimeTo] = useState(initial?.timeTo ?? '');
  const [alerts, setAlerts] = useState<AlertItem[]>([]);
  const [loadingAlerts, setLoadingAlerts] = useState(false);
  const [selectedAlertKeys, setSelectedAlertKeys] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (open) {
      setSummary(initial?.summary ?? '');
      setTitle(initial?.title ?? '');
      setSourceType(initial?.sourceType ?? 'manual');
      setSourceRef(initial?.sourceRef ?? '');
      setTimeFrom(initial?.timeFrom ?? '');
      setTimeTo(initial?.timeTo ?? '');
      setSelectedAlertKeys(new Set());
    }
  }, [open, initial?.summary, initial?.title, initial?.sourceType, initial?.sourceRef, initial?.timeFrom, initial?.timeTo]);

  useEffect(() => {
    if (!open || sourceType !== 'alert') return;
    let cancelled = false;
    setLoadingAlerts(true);
    getAdreAlerts()
      .then((data: unknown) => {
        const raw = data as { data?: { alerts?: AlertItem[] }; alerts?: AlertItem[] };
        const list = raw?.data?.alerts ?? raw?.alerts ?? [];
        const arr = Array.isArray(list) ? list : [];
        if (!cancelled) setAlerts(arr);
      })
      .catch(() => {
        if (!cancelled) setAlerts([]);
      })
      .finally(() => {
        if (!cancelled) setLoadingAlerts(false);
      });
    return () => { cancelled = true; };
  }, [open, sourceType]);

  // When creating from alert, set default title to "Alert: <alertname>" when selection changes (only if title is empty or already a default).
  useEffect(() => {
    if (sourceType !== 'alert' || selectedAlertKeys.size === 0) return;
    const firstIndex = alerts.findIndex((a, i) => selectedAlertKeys.has(getAlertKey(a, i)));
    if (firstIndex === -1) return;
    const first = alerts[firstIndex];
    const alertname =
      first?.labels?.alertname ?? first?.annotations?.summary ?? 'Alert';
    const defaultTitle = `Alert: ${alertname}`;
    setTitle((prev) => {
      if (prev === '' || prev.startsWith('Alert: ')) return defaultTitle;
      return prev;
    });
  }, [sourceType, selectedAlertKeys, alerts]);

  const toggleAlert = (key: string) => {
    setSelectedAlertKeys((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const handleSubmit = () => {
    let finalSourceRef = sourceRef.trim();
    let alertMeta: { nodeName?: string; serviceName?: string; clusterName?: string } = {};
    let selectedAlerts: AlertItem[] = [];
    if (sourceType === 'alert' && selectedAlertKeys.size > 0) {
      const refs = alerts
        .map((a, i) => (selectedAlertKeys.has(getAlertKey(a, i)) ? (a.fingerprint ?? getAlertKey(a, i)) : null))
        .filter(Boolean) as string[];
      finalSourceRef = refs.join(',');
      const firstSelected = alerts.find((a, i) => selectedAlertKeys.has(getAlertKey(a, i)));
      alertMeta = getAlertMetadataFromLabels(firstSelected?.labels);
      selectedAlerts = alerts.filter((a, i) => selectedAlertKeys.has(getAlertKey(a, i)));
    }
    const body: CreateInvestigationBody = {
      title: title.trim() || 'New investigation',
      summary: summary.trim() || undefined,
      sourceType: sourceType === 'manual' ? undefined : sourceType,
      sourceRef: finalSourceRef || undefined,
      timeFrom: timeFrom.trim() || undefined,
      timeTo: timeTo.trim() || undefined,
      ...(alertMeta.nodeName && { nodeName: alertMeta.nodeName }),
      ...(alertMeta.serviceName && { serviceName: alertMeta.serviceName }),
      ...(alertMeta.clusterName && { clusterName: alertMeta.clusterName }),
      ...(selectedAlerts.length > 0 && { alertSnapshot: selectedAlerts }),
    };
    onSubmit(body);
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      aria-labelledby="create-investigation-modal-title"
      maxWidth="sm"
      fullWidth
    >
      <DialogTitle id="create-investigation-modal-title" onClose={onClose}>
        Create investigation
      </DialogTitle>
      <DialogContent>
        <Stack gap={2} sx={{ mt: 0.5 }}>
          <TextField
            label="What would you like to investigate?"
            value={summary}
            onChange={(e) => setSummary(e.target.value)}
            placeholder="Describe what you want to investigate..."
            size="small"
            fullWidth
            multiline
            minRows={2}
          />
          <TextField
            label="Title"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Short title for this investigation"
            size="small"
            fullWidth
            required
          />
          <FormControl size="small" fullWidth>
            <InputLabel>Source type</InputLabel>
            <Select
              value={sourceType}
              label="Source type"
              onChange={(e) => setSourceType(e.target.value)}
            >
              <MenuItem value="manual">User request</MenuItem>
              <MenuItem value="alert">Alert</MenuItem>
            </Select>
          </FormControl>
          {sourceType === 'alert' && (
            <Box>
              <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1 }}>
                Firing alerts (select one or more)
              </Typography>
              {loadingAlerts ? (
                <Typography variant="body2" color="text.secondary">
                  Loading alerts...
                </Typography>
              ) : alerts.length === 0 ? (
                <Typography variant="body2" color="text.secondary">
                  No firing alerts.
                </Typography>
              ) : (
                <FormGroup>
                  <Box sx={{ maxHeight: 180, overflow: 'auto' }}>
                    {alerts.map((a, index) => {
                      const key = getAlertKey(a, index);
                      const label =
                        a.labels?.alertname ?? a.annotations?.summary ?? a.fingerprint ?? key;
                      return (
                        <FormControlLabel
                          key={key}
                          control={
                            <Checkbox
                              checked={selectedAlertKeys.has(key)}
                              onChange={() => toggleAlert(key)}
                            />
                          }
                          label={String(label)}
                        />
                      );
                    })}
                  </Box>
                </FormGroup>
              )}
            </Box>
          )}
          <DateTimePicker
            label="Time from"
            value={parseISODate(timeFrom)}
            onChange={(newValue) =>
              setTimeFrom(newValue ? newValue.toISOString() : '')
            }
            slotProps={{
              textField: {
                size: 'small',
                fullWidth: true,
                placeholder: 'e.g. 2025-01-01T00:00:00Z',
              },
            }}
          />
          <DateTimePicker
            label="Time to"
            value={parseISODate(timeTo)}
            onChange={(newValue) =>
              setTimeTo(newValue ? newValue.toISOString() : '')
            }
            slotProps={{
              textField: {
                size: 'small',
                fullWidth: true,
                placeholder: 'e.g. 2025-01-01T23:59:59Z',
              },
            }}
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button
          variant="contained"
          onClick={handleSubmit}
          disabled={isPending}
        >
          {isPending ? 'Creating...' : 'Create'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
