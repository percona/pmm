import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import MenuItem from '@mui/material/MenuItem';
import { TextInput, SwitchInput, SelectInput } from '@percona/percona-ui';
import { SEVERITY } from 'lib/constants';
import { zodResolver } from '@hookform/resolvers/zod';
import { FC, useEffect } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import { useUpdateSettings } from 'hooks/api/useSettings';
import { Messages } from '../../Settings.messages';
import { MAX_DAYS, MIN_DAYS } from '../advanced/Advanced.constants';
import {
  ADVISOR_SEVERITY_OPTIONS,
  MIN_ADVISOR_CHECK_INTERVAL,
  STT_CHECK_INTERVALS,
} from './Advisors.constants';
import { AdvisorsFormProps } from './AdvisorsForm.types';
import { AdvisorsFormValues, advisorsSchema } from './AdvisorsForm.schema';
import { toFormValues, toPayload } from './AdvisorsForm.utils';
import { SettingsFieldLabel } from '../settings-field-label';
import { SettingsSubmitButton } from '../settings-submit-button';
import { formControlClasses } from '@mui/material/FormControl';

export const AdvisorsForm: FC<AdvisorsFormProps> = ({ settings }) => {
  const { mutateAsync: updateSettings } = useUpdateSettings();

  const methods = useForm<AdvisorsFormValues>({
    resolver: zodResolver(advisorsSchema),
    defaultValues: toFormValues(settings),
    mode: 'onChange',
  });

  const { handleSubmit, reset, watch } = methods;

  const sttEnabled = watch('stt');
  const advisorNotificationsEnabled = watch('advisorNotifications');

  useEffect(() => {
    reset(toFormValues(settings));
  }, [settings, reset]);

  const onSubmit = async (values: AdvisorsFormValues) => {
    await updateSettings(toPayload(values), {
      onSuccess: () => {
        enqueueSnackbar(Messages.service.success, { variant: 'success' });
        reset(values);
      },
      onError: (error) => {
        enqueueSnackbar(
          error instanceof Error ? error.message : Messages.unauthorized,
          { variant: 'error' }
        );
      },
    });
  };

  const m = Messages.advisors;

  return (
    <FormProvider {...methods}>
      <Stack
        component="form"
        onSubmit={handleSubmit(onSubmit)}
        gap={5}
        sx={{
          [`.${formControlClasses.root}`]: {
            margin: 0,
          },
        }}
      >
        <Stack gap={1} data-testid="advisors-settings">
          <SettingsFieldLabel
            label={m.advisorsLabel}
            description={m.advisorsTooltip}
            readMoreLink={m.advisorsLink}
            data-testid="advisors-label"
          />
          <Stack gap={1}>
            <SwitchInput name="stt" label={m.advisorsLabel} />
          </Stack>
          {sttEnabled && (
            <Stack gap={2}>
              <SettingsFieldLabel
                label={m.checkIntervalLabel}
                description={m.checkIntervalTooltip}
                data-testid="check-intervals-label"
              />
              <Stack direction="row" columnGap={2} rowGap={3} flexWrap="wrap">
                {STT_CHECK_INTERVALS.map(({ name, label }) => (
                  <TextInput
                    key={name}
                    name={name}
                    label={label}
                    textFieldProps={{
                      type: 'number',
                      slotProps: {
                        htmlInput: {
                          min: MIN_ADVISOR_CHECK_INTERVAL,
                          step: 0.1,
                          'data-testid': `${name}-number-input`,
                        },
                      },
                      size: 'small',
                      sx: { minWidth: 80, maxWidth: 120 },
                    }}
                  />
                ))}
              </Stack>

              <Stack gap={1} data-testid="advisor-retention">
                <SettingsFieldLabel
                  label={m.advisorRetentionLabel}
                  description={m.advisorRetentionTooltip}
                  data-testid="advisor-retention-label"
                />
                <Stack direction="row" alignItems="baseline" gap={1}>
                  <TextInput
                    name="advisorRetention"
                    textFieldProps={{
                      type: 'number',
                      slotProps: {
                        htmlInput: {
                          min: MIN_DAYS,
                          max: MAX_DAYS,
                          step: 1,
                          'data-testid': 'advisorRetention-number-input',
                        },
                      },
                      sx: { minWidth: 120, maxWidth: 240 },
                      size: 'small',
                    }}
                  />
                  <Typography variant="body1" color="text.secondary">
                    {m.retentionUnits}
                  </Typography>
                </Stack>
              </Stack>

              <Stack gap={1} data-testid="advisor-notifications">
                <SettingsFieldLabel
                  label={m.advisorNotificationsLabel}
                  description={m.advisorNotificationsTooltip}
                  data-testid="advisor-notifications-label"
                />
                <SwitchInput
                  name="advisorNotifications"
                  label={m.advisorNotificationsLabel}
                  formControlLabelProps={{ sx: { mr: 0 } }}
                />
                {advisorNotificationsEnabled && (
                  // wider gap than the enclosing Stack: both fields carry helper
                  // text, which otherwise sits flush against the next field
                  <Stack gap={2}>
                    <SelectInput
                      name="advisorSeverityThreshold"
                      label={m.advisorSeverityThresholdLabel}
                      helperText={m.advisorSeverityThresholdTooltip}
                      formControlProps={{
                        sx: { minWidth: 240, maxWidth: 320 },
                        size: 'small',
                      }}
                      selectFieldProps={{
                        // @ts-expect-error data-testid is passed through to the DOM
                        'data-testid': 'advisorSeverityThreshold-select-input',
                      }}
                    >
                      {ADVISOR_SEVERITY_OPTIONS.map((severity) => (
                        <MenuItem key={severity} value={severity}>
                          {SEVERITY[severity]}
                        </MenuItem>
                      ))}
                    </SelectInput>
                    <TextInput
                      name="advisorNotificationEmails"
                      label={m.advisorEmailsLabel}
                      textFieldProps={{
                        helperText: m.advisorEmailsTooltip,
                        placeholder: 'dba@example.com, oncall@example.com',
                        size: 'small',
                        sx: { minWidth: 240, maxWidth: 520 },
                        slotProps: {
                          htmlInput: {
                            'data-testid':
                              'advisorNotificationEmails-text-input',
                          },
                        },
                      }}
                    />
                  </Stack>
                )}
              </Stack>
            </Stack>
          )}
        </Stack>

        <SettingsSubmitButton testId="advisors-button" />
      </Stack>
    </FormProvider>
  );
};
