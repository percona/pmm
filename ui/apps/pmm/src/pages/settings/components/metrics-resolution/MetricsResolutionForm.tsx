import Stack from '@mui/material/Stack';
import { TextInput, RadioGroup } from '@percona/percona-ui';
import { FC, useEffect, useMemo } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import { useUpdateSettings } from 'hooks/api/useSettings';
import { MetricsResolutions } from 'types/settings.types';
import { Messages } from '../../Settings.messages';
import {
  defaultResolutions,
  resolutionOptions,
  RESOLUTION_MAX,
  RESOLUTION_MIN,
  DEFAULT_METRICS,
} from './MetricsResolution.constants';
import {
  addUnits,
  getResolutionPreset,
  removeUnits,
} from './MetricsResolution.utils';
import {
  MetricsResolutionFormProps,
  MetricsResolutionFormValues,
} from './MetricsResolutionForm.types';
import { SettingsFieldLabel } from '../settings-field-label';
import { SettingsSubmitButton } from '../settings-submit-button';
import { formControlClasses } from '@mui/material/FormControl';

export const MetricsResolutionForm: FC<MetricsResolutionFormProps> = ({
  settings,
}) => {
  const { mutateAsync: updateSettings } = useUpdateSettings();
  const metricsResolutions = useMemo(
    () => settings?.metricsResolutions ?? DEFAULT_METRICS,
    [settings?.metricsResolutions]
  );
  const preset = getResolutionPreset(metricsResolutions);
  const raw = removeUnits(metricsResolutions);

  const methods = useForm<MetricsResolutionFormValues>({
    defaultValues: {
      preset,
      lr: raw.lr,
      mr: raw.mr,
      hr: raw.hr,
    },
  });

  const currentPreset = methods.watch('preset');

  useEffect(() => {
    methods.reset({
      preset: getResolutionPreset(metricsResolutions),
      ...removeUnits(metricsResolutions),
    });
  }, [metricsResolutions, methods.reset]);

  useEffect(() => {
    if (currentPreset && currentPreset !== 'custom') {
      const idx = resolutionOptions.findIndex((o) => o.value === currentPreset);
      if (idx >= 0) {
        const def = removeUnits(defaultResolutions[idx]);
        methods.setValue('lr', def.lr, { shouldDirty: false });
        methods.setValue('mr', def.mr, { shouldDirty: false });
        methods.setValue('hr', def.hr, { shouldDirty: false });
      }
    }
  }, [currentPreset, methods.setValue]);

  const onSubmit = async (values: MetricsResolutionFormValues) => {
    const payload: MetricsResolutions = addUnits({
      lr: values.lr,
      mr: values.mr,
      hr: values.hr,
    });
    await updateSettings(
      { metricsResolutions: payload },
      {
        onSuccess: () => {
          enqueueSnackbar(Messages.service.success, { variant: 'success' });
          methods.reset({
            preset: values.preset,
            lr: values.lr,
            mr: values.mr,
            hr: values.hr,
          });
        },
        onError: (error) => {
          enqueueSnackbar(
            error instanceof Error ? error.message : Messages.unauthorized,
            { variant: 'error' }
          );
        },
      }
    );
  };

  const validateNumber = (v: string) => {
    const n = parseInt(v, 10);

    if (isNaN(n) || v === '') {
      return validation.required;
    }

    if (n < RESOLUTION_MIN || n > RESOLUTION_MAX) {
      return validation.minMax(RESOLUTION_MIN, RESOLUTION_MAX);
    }

    return true;
  };

  const { label, link, tooltip, intervals, validation } = Messages.metrics;

  return (
    <FormProvider {...methods}>
      <Stack component="form" onSubmit={methods.handleSubmit(onSubmit)} gap={2}>
        <SettingsFieldLabel
          label={label}
          description={tooltip}
          readMoreLink={link}
        />

        <RadioGroup
          name="preset"
          options={resolutionOptions}
          radioGroupFieldProps={{
            row: true,
            sx: { columnGap: 2, rowGap: 1, mb: 1 },
          }}
        />

        <Stack
          direction="row"
          columnGap={2}
          rowGap={3}
          flexWrap="wrap"
          mb={2}
          sx={{
            [`.${formControlClasses.root}`]: {
              margin: 0,
            },
          }}
        >
          <TextInput
            name="lr"
            label={intervals.low}
            controllerProps={{ rules: { validate: validateNumber } }}
            textFieldProps={{
              type: 'number',
              disabled: currentPreset !== 'custom',
              slotProps: {
                htmlInput: { min: RESOLUTION_MIN, max: RESOLUTION_MAX },
              },
              sx: { minWidth: 80, maxWidth: 120 },
              size: 'small',
            }}
          />
          <TextInput
            name="mr"
            label={intervals.medium}
            controllerProps={{ rules: { validate: validateNumber } }}
            textFieldProps={{
              type: 'number',
              disabled: currentPreset !== 'custom',
              slotProps: {
                htmlInput: { min: RESOLUTION_MIN, max: RESOLUTION_MAX },
              },
              sx: { minWidth: 80, maxWidth: 120 },
              size: 'small',
            }}
          />
          <TextInput
            name="hr"
            label={intervals.high}
            controllerProps={{ rules: { validate: validateNumber } }}
            textFieldProps={{
              type: 'number',
              disabled: currentPreset !== 'custom',
              slotProps: {
                htmlInput: { min: RESOLUTION_MIN, max: RESOLUTION_MAX },
              },
              sx: { minWidth: 80, maxWidth: 120 },
              size: 'small',
            }}
          />
        </Stack>
        <SettingsSubmitButton />
      </Stack>
    </FormProvider>
  );
};
