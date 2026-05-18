import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import { TextInput, RadioGroup } from '@percona/percona-ui';
import { FC, useEffect, useMemo } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
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
import { MetricsResolutionFormProps } from './MetricsResolutionForm.types';
import {
  metricsResolutionSchema,
  MetricsResolutionFormValues,
} from './MetricsResolutionForm.schema';
import { SettingsFieldLabel } from '../settings-field-label';
import { SettingsSubmitButton } from '../settings-submit-button';
import { formControlClasses } from '@mui/material/FormControl';
import { helperTextTestId } from 'utils/mui.utils';

export const MetricsResolutionForm: FC<MetricsResolutionFormProps> = ({
  settings,
}) => {
  const { mutateAsync: updateSettings } = useUpdateSettings();
  const metricsResolutions = useMemo(
    () => settings.metricsResolutions ?? DEFAULT_METRICS,
    [settings.metricsResolutions]
  );
  const preset = getResolutionPreset(metricsResolutions);
  const { lr, mr, hr } = removeUnits(metricsResolutions);

  const methods = useForm<MetricsResolutionFormValues>({
    resolver: zodResolver(metricsResolutionSchema),
    defaultValues: { preset, lr, mr, hr },
  });

  const currentPreset = methods.watch('preset');

  useEffect(() => {
    const raw = removeUnits(metricsResolutions);
    methods.reset({ preset: getResolutionPreset(metricsResolutions), ...raw });
  }, [metricsResolutions, methods]);

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
  }, [currentPreset, methods]);

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

  const { label, link, tooltip, intervals } = Messages.metrics;

  return (
    <FormProvider {...methods}>
      <Stack component="form" onSubmit={methods.handleSubmit(onSubmit)} gap={2}>
        <SettingsFieldLabel
          label={label}
          description={tooltip}
          readMoreLink={link}
          data-testid="metrics-resolution-label"
        />

        <Box data-testid="metrics-resolution-radio-group">
          <RadioGroup
            name="preset"
            options={resolutionOptions}
            radioGroupFieldProps={{
              row: true,
              sx: { columnGap: 2, rowGap: 1, mb: 1 },
            }}
          />
        </Box>

        <Stack
          direction="row"
          columnGap={2}
          rowGap={3}
          flexWrap="wrap"
          mb={2}
          sx={{
            [`.${formControlClasses.root}`]: {
              margin: -0,
            },
          }}
        >
          <TextInput
            name="lr"
            label={intervals.low}
            textFieldProps={{
              type: 'number',
              disabled: currentPreset !== 'custom',
              slotProps: {
                htmlInput: {
                  min: RESOLUTION_MIN,
                  max: RESOLUTION_MAX,
                  'data-testid': 'lr-number-input',
                },
              },
              sx: { minWidth: 80, maxWidth: 120 },
              size: 'small',
            }}
            formHelperTextProps={helperTextTestId('lr-field-error-message')}
          />
          <TextInput
            name="mr"
            label={intervals.medium}
            textFieldProps={{
              type: 'number',
              disabled: currentPreset !== 'custom',
              slotProps: {
                htmlInput: {
                  min: RESOLUTION_MIN,
                  max: RESOLUTION_MAX,
                  'data-testid': 'mr-number-input',
                },
              },
              sx: { minWidth: 80, maxWidth: 120 },
              size: 'small',
            }}
            formHelperTextProps={helperTextTestId('mr-field-error-message')}
          />
          <TextInput
            name="hr"
            label={intervals.high}
            textFieldProps={{
              type: 'number',
              disabled: currentPreset !== 'custom',
              slotProps: {
                htmlInput: {
                  min: RESOLUTION_MIN,
                  max: RESOLUTION_MAX,
                  'data-testid': 'hr-number-input',
                },
              },
              sx: { minWidth: 80, maxWidth: 120 },
              size: 'small',
            }}
            formHelperTextProps={helperTextTestId('hr-field-error-message')}
          />
        </Stack>
        <SettingsSubmitButton testId="metrics-resolution-button" />
      </Stack>
    </FormProvider>
  );
};
