import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import { Modal } from 'components/modal';
import { usePrometheusAlertRules } from 'hooks/api/usePrometheusAlertRules';
import { useEffect, useMemo, useState } from 'react';
import { getInitialFormValues, getNodeAlerts } from './AlertThresholds.utils';
import { Table } from '@percona/percona-ui';
import { ALERT_THRESHOLDS_COLUMNS } from './AlertThresholds.constants';
import { FormProvider, useForm } from 'react-hook-form';
import { AlertThresholdsFormValues } from './AlertThresholds.types';
import {
  UpdateAlertThresholdData,
  useUpdateAlertThresholds,
} from 'hooks/api/useUpdateAlertThreshold';
import messenger from 'lib/messenger';
import { useGrafana } from 'contexts/grafana';
import { OpenAlertThresholdsModalMessage } from '@pmm/shared';

const AlertThresholds = () => {
  const [nodeName, setNodeName] = useState<string>();
  const [open, setIsOpen] = useState(false);
  const { data, isLoading } = usePrometheusAlertRules({
    enabled: open,
  });
  const rules = useMemo(
    () => (data && nodeName ? getNodeAlerts(nodeName, data) : []),
    [data, nodeName]
  );
  const methods = useForm<AlertThresholdsFormValues>({
    defaultValues: getInitialFormValues(rules),
  });
  const { mutateAsync } = useUpdateAlertThresholds();
  const { isFrameLoaded } = useGrafana();

  const handleSubmit = async (values: AlertThresholdsFormValues) => {
    const initial = getInitialFormValues(rules);
    const changedValues = Object.entries(values).reduce(
      (acc, [key, value]) => {
        if (initial[key] !== value || true) {
          acc[key] = value;
        }
        return acc;
      },
      {} as Record<string, number | undefined>
    );

    console.log('changedValues:', changedValues);

    const payloads = Object.entries(changedValues)
      .filter(([, threshold]) => threshold !== undefined && nodeName)
      .map<UpdateAlertThresholdData>(
        ([uid, threshold]) =>
          ({
            uid,
            nodeName,
            threshold,
          }) as UpdateAlertThresholdData
      );

    await mutateAsync(payloads);

    handleClose();
  };

  const handleClose = () => {
    setNodeName(undefined);
    setIsOpen(false);
  };

  useEffect(() => {
    console.log('reset');
    methods.reset(getInitialFormValues(rules));
  }, [rules]);

  useEffect(() => {
    if (isFrameLoaded) {
      console.log('[ALERT_THRESHOLDS] adding listener');
      messenger.addListener({
        type: 'OPEN_ALERT_THRESHOLDS_MODAL',
        onMessage: (msg: OpenAlertThresholdsModalMessage) => {
          setNodeName(msg.payload?.nodeName);
          setIsOpen(true);
        },
      });
    }
  }, [isFrameLoaded]);

  if (isLoading || !nodeName) {
    return null;
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={`Alert thresholds: ${nodeName}`}
    >
      <FormProvider {...methods}>
        <Stack
          component="form"
          onSubmit={methods.handleSubmit(handleSubmit, (errors) =>
            console.error('Validation errors:', errors)
          )}
        >
          <Table
            tableName="alert-thresholds"
            columns={ALERT_THRESHOLDS_COLUMNS}
            data={rules || []}
          />
          <Stack
            direction="row"
            justifyContent="end"
            sx={{ gap: 1, pt: 2, alignSelf: 'flex-end' }}
          >
            <Button type="button" variant="text" onClick={handleClose}>
              Cancel and close
            </Button>
            <Button type="submit" variant="contained">
              Submit changes
            </Button>
          </Stack>
        </Stack>
      </FormProvider>
    </Modal>
  );
};

export default AlertThresholds;
