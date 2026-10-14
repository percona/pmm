import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Table } from '@percona/peak-ui';
import { Modal } from 'components/modal';
import {
  useBatchUpdateNodeThresholds,
  useNodeThresholds,
} from 'hooks/api/useNodeThresholds';
import { usePrometheusAlertRules } from 'hooks/api/usePrometheusAlertRules';
import { enqueueSnackbar } from 'notistack';
import { type FC, useEffect, useMemo } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { ALERT_THRESHOLDS_COLUMNS } from './AlertThresholds.constants';
import { Messages } from './AlertThresholds.messages';
import type {
  AlertThresholdRow,
  AlertThresholdsFormValues,
  AlertThresholdsNode,
} from './AlertThresholds.types';
import type {
  ListThresholdsResponse,
  PrometheusAlertRulesResponse,
} from 'types/alerting.types';
import {
  buildThresholdUpdates,
  getInitialValues,
  getRows,
  getRuleTitles,
} from './AlertThresholds.utils';

interface Props {
  node: AlertThresholdsNode | null;
  onClose: () => void;
}

const AlertThresholds: FC<Props> = ({ node, onClose }) => {
  const nodeId = node?.nodeId;

  const { data, isLoading, isError } = useNodeThresholds(nodeId ?? '', {
    enabled: !!nodeId,
  });

  const {
    data: rulesData,
    isLoading: rulesLoading,
    isError: rulesError,
  } = usePrometheusAlertRules({
    enabled: !!nodeId,
  });

  // Rule titles live in Grafana, not in the thresholds response, so they are joined
  // on the identity label PMM stamps on every rule it creates.
  const ruleTitles = useMemo(
    () => getRuleTitles(rulesData as PrometheusAlertRulesResponse),
    [rulesData]
  );

  // A row whose rule is missing from Grafana belongs to a deleted rule not yet swept, or
  // to another org, so it is hidden rather than offered for editing.
  const rows = useMemo<AlertThresholdRow[]>(
    () =>
      getRows(data as ListThresholdsResponse, ruleTitles).filter((row) =>
        ruleTitles.has(row.ruleId)
      ),
    [data, ruleTitles]
  );

  // Depends on `data` only - deliberately not on `rows`, which carries rule titles.
  const initialValues = useMemo<AlertThresholdsFormValues>(
    () => getInitialValues(data as ListThresholdsResponse),
    [data]
  );

  const methods = useForm<AlertThresholdsFormValues>({
    defaultValues: initialValues,
  });
  const { mutateAsync: applyThresholds } = useBatchUpdateNodeThresholds(
    nodeId ?? ''
  );

  // Seeded here rather than through useForm's defaultValues, which this always-mounted
  // component would only see once. A refetch keeps what the user has typed; handleClose
  // resets the form, so a reopened modal starts clean.
  useEffect(() => {
    methods.reset(initialValues, { keepDirtyValues: true });
  }, [nodeId, initialValues, methods]);

  const handleClose = () => {
    methods.reset({});
    onClose();
  };

  const handleSubmit = async (values: AlertThresholdsFormValues) => {
    const updates = buildThresholdUpdates(
      rows,
      values,
      'THRESHOLD_SCOPE_NODE',
      nodeId ?? ''
    );

    if (updates.length > 0) {
      try {
        // One transactional call: either every row lands or none does.
        await applyThresholds(updates);
      } catch {
        // The axios interceptor reports the reason; keep the modal open with the edits.
        return;
      }

      enqueueSnackbar(Messages.success.updated, { variant: 'success' });
    }

    handleClose();
  };

  if (!node) {
    return null;
  }

  return (
    <Modal open onClose={handleClose} title={Messages.title(node.nodeName)}>
      <FormProvider {...methods}>
        <Stack component="form" onSubmit={methods.handleSubmit(handleSubmit)}>
          {isError || rulesError ? (
            <Typography variant="body2" color="error">
              {Messages.error}
            </Typography>
          ) : isLoading || rulesLoading ? (
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
              getRowId={(row) => row.id}
              enableHiding={false}
              enableStickyHeader
              muiTableContainerProps={{
                sx: {
                  // Keeps the title and actions on screen when a node has many rules.
                  maxHeight: 'calc(100vh - 320px)',
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
              disabled={
                isError ||
                rulesError ||
                rows.length === 0 ||
                methods.formState.isSubmitting
              }
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
