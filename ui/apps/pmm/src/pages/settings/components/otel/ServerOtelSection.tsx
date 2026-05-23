import {
  Alert,
  Box,
  Card,
  CardContent,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { SwitchInput } from '@percona/percona-ui';
import { FC, useEffect } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import { useUpdateSettings } from 'hooks/api/useSettings';
import type { Settings } from 'types/settings.types';
import { Messages } from '../../Settings.messages';
import { SettingsSubmitButton } from '../settings-submit-button';

const MIN_RETENTION = 1;
const MAX_RETENTION = 365;

type ServerOtelForm = {
  collectorEnabled: boolean;
  logsRetentionDays: number;
  tracesRetentionDays: number;
  metricsRetentionDays: number;
};

function readOtel(settings: Settings): ServerOtelForm {
  const o = settings.otel ?? {};
  return {
    collectorEnabled: o.collectorEnabled ?? o.collector_enabled ?? true,
    logsRetentionDays: o.logsRetentionDays ?? o.logs_retention_days ?? 7,
    tracesRetentionDays: o.tracesRetentionDays ?? o.traces_retention_days ?? 7,
    metricsRetentionDays: o.metricsRetentionDays ?? o.metrics_retention_days ?? 90,
  };
}

export const ServerOtelSection: FC<{ settings: Settings }> = ({ settings }) => {
  const { mutateAsync: updateSettings } = useUpdateSettings();
  const methods = useForm<ServerOtelForm>({ defaultValues: readOtel(settings) });
  const { handleSubmit, reset, register, watch } = methods;
  const collectorEnabled = watch('collectorEnabled');

  useEffect(() => {
    reset(readOtel(settings));
  }, [settings, reset]);

  const onSubmit = async (values: ServerOtelForm) => {
    await updateSettings(
      {
        otel: {
          collectorEnabled: values.collectorEnabled,
          logsRetentionDays: values.logsRetentionDays,
          tracesRetentionDays: values.tracesRetentionDays,
          metricsRetentionDays: values.metricsRetentionDays,
        },
      },
      {
        onSuccess: () => {
          enqueueSnackbar(Messages.service.success, { variant: 'success' });
          reset(values);
        },
        onError: (error) => {
          enqueueSnackbar(error instanceof Error ? error.message : Messages.unauthorized, {
            variant: 'error',
          });
        },
      }
    );
  };

  const m = Messages.otel.server;

  return (
    <Card variant="outlined">
      <CardContent>
        <Typography variant="h6" sx={{ mb: 1 }}>
          {m.title}
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {m.description}
        </Typography>
        {!collectorEnabled && (
          <Alert severity="warning" sx={{ mb: 2 }}>
            {m.disabledWarning}
          </Alert>
        )}
        <FormProvider {...methods}>
          <Stack component="form" onSubmit={handleSubmit(onSubmit)} gap={2}>
            <SwitchInput name="collectorEnabled" label={m.collectorEnabledLabel} />
            <Stack direction={{ xs: 'column', sm: 'row' }} gap={2}>
              <TextField
                label={m.logsRetentionLabel}
                type="number"
                size="small"
                inputProps={{ min: MIN_RETENTION, max: MAX_RETENTION }}
                {...register('logsRetentionDays', { valueAsNumber: true })}
              />
              <TextField
                label={m.tracesRetentionLabel}
                type="number"
                size="small"
                inputProps={{ min: MIN_RETENTION, max: MAX_RETENTION }}
                {...register('tracesRetentionDays', { valueAsNumber: true })}
              />
              <TextField
                label={m.metricsRetentionLabel}
                type="number"
                size="small"
                inputProps={{ min: MIN_RETENTION, max: MAX_RETENTION }}
                {...register('metricsRetentionDays', { valueAsNumber: true })}
              />
            </Stack>
            <Box>
              <SettingsSubmitButton testId="otel-server-button" />
            </Box>
          </Stack>
        </FormProvider>
      </CardContent>
    </Card>
  );
};
