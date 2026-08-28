import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Table } from '@percona/peak-ui';
import type { OpenAlertThresholdsModalMessage } from '@pmm/shared';
import { Modal } from 'components/modal';
import {
  NODE_SCOPE,
  useBatchUpdateNodeThresholds,
  useNodeThresholds,
} from 'hooks/api/useNodeThresholds';
import { usePrometheusAlertRules } from 'hooks/api/usePrometheusAlertRules';
import messenger from 'lib/messenger';
import { enqueueSnackbar } from 'notistack';
import { useEffect, useMemo, useState } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { ALERT_THRESHOLDS_COLUMNS } from './AlertThresholds.constants';
import { Messages } from './AlertThresholds.messages';
import type {
  AlertThresholdRow,
  AlertThresholdsFormValues,
} from './AlertThresholds.types';
import type {
  ListThresholdsResponse,
  PrometheusAlertRulesResponse,
  ThresholdUpdate,
} from 'types/alerting.types';
import { getRows, getRuleTitles } from './AlertThresholds.utils';

const AlertThresholds = () => {
  const [nodeId, setNodeId] = useState<string>();
  const [nodeName, setNodeName] = useState<string>();
  const [open, setIsOpen] = useState(false);

  const { data, isLoading } = useNodeThresholds(nodeId ?? '', {
    enabled: open && !!nodeId,
  });

  const { data: rulesData } = usePrometheusAlertRules({
    enabled: open && !!nodeId,
  });

  // Rule titles live in Grafana, not in the thresholds response, so they are joined
  // on the identity label PMM stamps on every rule it creates.
  const ruleTitles = useMemo(
    () => getRuleTitles(rulesData as PrometheusAlertRulesResponse),
    [rulesData]
  );

  const rows = useMemo<AlertThresholdRow[]>(
    () => getRows(data as ListThresholdsResponse, ruleTitles),
    [data, ruleTitles]
  );

  const initialValues = useMemo<AlertThresholdsFormValues>(
    () =>
      rows.reduce((acc, row) => {
        acc[row.id] = row.effectiveValue;
        return acc;
      }, {} as AlertThresholdsFormValues),
    [rows]
  );

  const methods = useForm<AlertThresholdsFormValues>({
    defaultValues: initialValues,
  });
  const { mutateAsync: applyThresholds } = useBatchUpdateNodeThresholds(
    nodeId ?? ''
  );

  useEffect(() => {
    methods.reset(initialValues);
  }, [initialValues, methods]);

  // Deliberately has no dependency array, so it re-subscribes after every render.
  //
  // GrafanaProvider's cleanup calls messenger.unregister(), which empties the shared
  // listener array rather than removing only its own listeners, and its effect depends
  // on `navigate` - so an ordinary navigation discards this subscription along with
  // everyone else's. Re-registering on each render is what puts it back; with `[]` the
  // modal would work until the first navigation and then silently stop opening.
  //
  // The costs are a subscribe/unsubscribe cycle per render, and a narrow window between
  // cleanup and re-subscribe in which an OPEN_ALERT_THRESHOLDS_MODAL message would be
  // dropped. Both go away once unregister() is made the true inverse of register() -
  // detaching the window listener and leaving the array to each component's own cleanup.
  // Tracked as a follow-up; fixing it here would mean changing shared messenger
  // behaviour that every other consumer relies on.
  useEffect(() => {
    const handler = messenger.addListener({
      type: 'OPEN_ALERT_THRESHOLDS_MODAL',
      onMessage: (msg: OpenAlertThresholdsModalMessage) => {
        setNodeId(msg.payload?.nodeId);
        setNodeName(msg.payload?.nodeName);
        setIsOpen(true);
      },
    });

    return () => messenger.removeListener(handler);
  });

  const handleClose = () => {
    setNodeId(undefined);
    setNodeName(undefined);
    setIsOpen(false);
  };

  const handleSubmit = async (values: AlertThresholdsFormValues) => {
    const updates: ThresholdUpdate[] = [];

    for (const row of rows) {
      const raw = values[row.id];
      const parsed =
        raw === undefined || (raw as unknown) === '' ? undefined : Number(raw);
      const cleared = parsed === undefined || Number.isNaN(parsed);

      const base = {
        scope: NODE_SCOPE,
        target: nodeId ?? '',
        ruleId: row.ruleId,
        paramName: row.paramName,
      };

      // Emptying the field, or typing the default back in, returns the node to the
      // rule default. That is only a change if an override exists today; omitting
      // the value clears it rather than writing the default as a new override.
      if (cleared || parsed === row.defaultValue) {
        if (row.isOverridden) {
          updates.push(base);
        }
        continue;
      }

      if (parsed !== row.effectiveValue) {
        updates.push({ ...base, value: parsed });
      }
    }

    if (updates.length > 0) {
      // One transactional call: either every row lands or none does.
      await applyThresholds(updates);
      enqueueSnackbar(Messages.success.updated, { variant: 'success' });
    }

    handleClose();
  };

  if (!open || !nodeId) {
    return null;
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={Messages.title(nodeName ?? '')}
    >
      <FormProvider {...methods}>
        <Stack component="form" onSubmit={methods.handleSubmit(handleSubmit)}>
          {isLoading ? (
            <Typography variant="body2" color="text.secondary">
              {Messages.loading}
            </Typography>
          ) : rows.length === 0 ? (
            <Typography variant="body2" color="text.secondary">
              {Messages.empty}
            </Typography>
          ) : (
            <Table
              tableName="alert-thresholds"
              columns={ALERT_THRESHOLDS_COLUMNS}
              data={rows}
              enableHiding={false}
              muiTableContainerProps={{
                sx: {
                  border: '1px solid',
                  borderColor: 'divider',
                  borderRadius: '8px',
                },
              }}
            />
          )}
          <Stack
            direction="row"
            justifyContent="end"
            sx={{ gap: 1, pt: 2, alignSelf: 'flex-end' }}
          >
            <Button type="button" variant="text" onClick={handleClose}>
              {Messages.actions.cancel}
            </Button>
            <Button
              type="submit"
              variant="contained"
              disabled={rows.length === 0}
            >
              {Messages.actions.submit}
            </Button>
          </Stack>
        </Stack>
      </FormProvider>
    </Modal>
  );
};

export default AlertThresholds;
