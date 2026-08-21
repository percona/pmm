import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Table } from '@percona/peak-ui';
import { OpenAlertThresholdsModalMessage } from '@pmm/shared';
import { Modal } from 'components/modal';
import {
  useDeleteNodeThreshold,
  useNodeThresholds,
  useSetNodeThreshold,
} from 'hooks/api/useNodeThresholds';
import messenger from 'lib/messenger';
import { enqueueSnackbar } from 'notistack';
import { useEffect, useMemo, useState } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { ALERT_THRESHOLDS_COLUMNS } from './AlertThresholds.constants';
import {
  AlertThresholdRow,
  AlertThresholdsFormValues,
  thresholdRowId,
} from './AlertThresholds.types';

const AlertThresholds = () => {
  const [nodeId, setNodeId] = useState<string>();
  const [nodeName, setNodeName] = useState<string>();
  const [open, setIsOpen] = useState(false);

  const { data, isLoading } = useNodeThresholds(nodeId ?? '', {
    enabled: open && !!nodeId,
  });

  const rows = useMemo<AlertThresholdRow[]>(
    () =>
      (data?.thresholds ?? []).map((t) => ({ ...t, id: thresholdRowId(t) })),
    [data]
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
  const { mutateAsync: setThreshold } = useSetNodeThreshold(nodeId ?? '');
  const { mutateAsync: deleteThreshold } = useDeleteNodeThreshold(nodeId ?? '');

  useEffect(() => {
    methods.reset(initialValues);
  }, [initialValues, methods]);

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
    const operations: Promise<unknown>[] = [];

    for (const row of rows) {
      const raw = values[row.id];
      const parsed =
        raw === undefined || (raw as unknown) === '' ? undefined : Number(raw);
      const cleared = parsed === undefined || Number.isNaN(parsed);

      // Clearing the field or setting it to the default reverts the node to the
      // template default (delete the override); only needed if one exists.
      if (cleared || parsed === row.defaultValue) {
        if (row.isOverridden) {
          operations.push(
            deleteThreshold({ ruleId: row.ruleId, paramName: row.paramName })
          );
        }
        continue;
      }

      if (parsed !== row.effectiveValue) {
        operations.push(
          setThreshold({
            ruleId: row.ruleId,
            paramName: row.paramName,
            value: parsed,
          })
        );
      }
    }

    if (operations.length > 0) {
      await Promise.all(operations);
      enqueueSnackbar('Alert thresholds updated', { variant: 'success' });
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
      title={`Alert thresholds: ${nodeName ?? ''}`}
    >
      <FormProvider {...methods}>
        <Stack component="form" onSubmit={methods.handleSubmit(handleSubmit)}>
          {isLoading ? (
            <Typography variant="body2" color="text.secondary">
              Loading thresholds…
            </Typography>
          ) : rows.length === 0 ? (
            <Typography variant="body2" color="text.secondary">
              No overridable thresholds for this node.
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
              Cancel and close
            </Button>
            <Button
              type="submit"
              variant="contained"
              disabled={rows.length === 0}
            >
              Submit changes
            </Button>
          </Stack>
        </Stack>
      </FormProvider>
    </Modal>
  );
};

export default AlertThresholds;
