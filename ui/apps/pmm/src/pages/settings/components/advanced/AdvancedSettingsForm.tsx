import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import CloseIcon from '@mui/icons-material/Close';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import WarningIcon from '@mui/icons-material/Warning';
import { TextInput, SwitchInput } from '@percona/percona-ui';
import { zodResolver } from '@hookform/resolvers/zod';
import { FC, useEffect, useState } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import { useUpdateSettings } from 'hooks/api/useSettings';
import { Messages } from '../../Settings.messages';
import {
  FEATURE_MANAGEMENT_SETTINGS,
  MAX_DAYS,
  MIN_DAYS,
  MIN_STT_CHECK_INTERVAL,
  STT_CHECK_INTERVALS,
  TECHNICAL_PREVIEW_DOC_URL,
} from './Advanced.constants';
import { MAX_LABEL_WIDTH } from '../../Settings.constants';
import { AdvancedSettingsFormProps } from './AdvancedSettingsForm.types';
import {
  AdvancedSettingsFormValues,
  advancedSettingsSchema,
} from './AdvancedSettingsForm.schema';
import { toFormValues, toPayload } from './AdvancedSettingsForm.utils';
import { SettingsFieldLabel } from '../settings-field-label';
import { SettingsSubmitButton } from '../settings-submit-button';
import { formControlClasses } from '@mui/material/FormControl';
import { formControlLabelClasses } from '@mui/material/FormControlLabel';
import { helperTextTestId } from 'utils/mui.utils';

export const AdvancedSettingsForm: FC<AdvancedSettingsFormProps> = ({
  settings,
}) => {
  const { mutateAsync: updateSettings } = useUpdateSettings();

  const methods = useForm<AdvancedSettingsFormValues>({
    resolver: zodResolver(advancedSettingsSchema),
    defaultValues: toFormValues(settings),
    mode: 'onChange',
  });

  const { handleSubmit, reset, watch, setValue } = methods;

  const sttEnabled = watch('stt');
  const [telemetryDialogOpen, setTelemetryDialogOpen] = useState(false);

  useEffect(() => {
    reset(toFormValues(settings));
  }, [settings, reset]);

  const onSubmit = async (values: AdvancedSettingsFormValues) => {
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

  const m = Messages.advanced;

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
        <Stack gap={1}>
          <SettingsFieldLabel
            label={m.publicAddressLabel}
            description={m.publicAddressTooltip}
            data-testid="public-address-label"
          />
          <Stack direction="row" flexWrap="wrap" gap={2} alignItems="center">
            <TextInput
              name="publicAddress"
              textFieldProps={{
                size: 'small',
                placeholder: m.publicAddressPlaceholder,
                sx: { flex: 1, minWidth: 240 },
                slotProps: {
                  htmlInput: { 'data-testid': 'publicAddress-text-input' },
                },
              }}
            />
            <Button
              type="button"
              variant="text"
              startIcon={<ContentCopyIcon />}
              data-testid="public-address-button"
              onClick={() =>
                setValue('publicAddress', window.location.host, {
                  shouldDirty: true,
                })
              }
            >
              {m.publicAddressButton}
            </Button>
          </Stack>
        </Stack>

        <Stack gap={1}>
          <SettingsFieldLabel
            label={m.retentionLabel}
            description={m.retentionTooltip}
            readMoreLink={m.retentionLink}
            data-testid="advanced-label"
          />
          <Stack direction="row" alignItems="baseline" gap={1}>
            <TextInput
              name="retention"
              textFieldProps={{
                type: 'number',
                slotProps: {
                  htmlInput: {
                    min: MIN_DAYS,
                    max: MAX_DAYS,
                    step: 1,
                    'data-testid': 'retention-number-input',
                  },
                },
                sx: { minWidth: 120, maxWidth: 240 },
                size: 'small',
              }}
              formHelperTextProps={helperTextTestId(
                'retention-field-error-message'
              )}
            />
            <Typography variant="body1" color="text.secondary">
              {m.retentionUnits}
            </Typography>
          </Stack>
        </Stack>

        <Stack gap={1} data-testid="advanced-telemetry">
          <SettingsFieldLabel
            label={m.telemetryLabel}
            description={m.telemetryTooltip}
            readMoreLink={m.telemetryLink}
            data-testid="advanced-telemetry-label"
          />
          <Stack direction="row" alignItems="center" gap={0.5}>
            <SwitchInput
              name="telemetry"
              label={m.telemetryLabel}
              formControlLabelProps={{ sx: { mr: 0 } }}
            />
            {'—'}
            <Link
              component="button"
              type="button"
              variant="body1"
              data-testid="telemetry-summaries-link"
              onClick={() => setTelemetryDialogOpen(true)}
            >
              {m.telemetryDialogLink}
            </Link>
          </Stack>
          <Dialog
            open={telemetryDialogOpen}
            onClose={() => setTelemetryDialogOpen(false)}
            maxWidth="sm"
            fullWidth
            data-testid="telemetry-summaries-dialog"
            slotProps={{ paper: { elevation: 1 } }}
          >
            <DialogTitle>
              {m.telemetryLabel}
              <IconButton
                aria-label="close"
                size="medium"
                onClick={() => setTelemetryDialogOpen(false)}
                sx={{ position: 'absolute', right: 8, top: 8 }}
              >
                <CloseIcon />
              </IconButton>
            </DialogTitle>
            <DialogContent>
              <Typography variant="body1" mb={2}>
                {m.telemetrySummaryTitle}
              </Typography>
              {(settings.telemetrySummaries ?? []).map((s) => (
                <Typography
                  key={s}
                  variant="body1"
                  component="li"
                  sx={{ ml: 3 }}
                >
                  {s}
                </Typography>
              ))}
            </DialogContent>
          </Dialog>
        </Stack>

        <Stack gap={2}>
          <SettingsFieldLabel
            label={m.featureManagementLabel}
            description={m.featureManagementDescription}
          />
          <Stack
            gap={2}
            sx={{
              [`.${formControlLabelClasses.root}`]: {
                marginRight: 0,
              },
            }}
          >
            {FEATURE_MANAGEMENT_SETTINGS.map(
              ({ name, label, tooltip, link, testId }) => (
                <Stack
                  key={name}
                  direction="row"
                  alignItems="center"
                  data-testid={testId}
                >
                  <SwitchInput name={name} label={label} />
                  <Tooltip
                    title={
                      <Box data-testid="info-tooltip">
                        <Typography variant="caption">
                          {tooltip}{' '}
                          {link && (
                            <Link
                              href={link}
                              target="_blank"
                              rel="noopener noreferrer"
                              color="inherit"
                              sx={{ textDecorationColor: 'inherit' }}
                            >
                              {Messages.tooltipLinkText}
                            </Link>
                          )}
                        </Typography>
                      </Box>
                    }
                    arrow
                  >
                    <IconButton size="small" data-testid="info-icon">
                      <InfoOutlinedIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                </Stack>
              )
            )}
          </Stack>
        </Stack>

        <Stack gap={1} data-testid="advanced-advisors">
          <SettingsFieldLabel
            label={m.advisorsLabel}
            description={m.advisorsTooltip}
            readMoreLink={m.advisorsLink}
            data-testid="advanced-advisors-label"
          />
          <Stack gap={1}>
            <SwitchInput name="stt" label={m.advisorsLabel} />
          </Stack>
          {sttEnabled && (
            <Stack gap={2}>
              <Typography
                variant="body1"
                sx={{ maxWidth: MAX_LABEL_WIDTH }}
                data-testid="check-intervals-label"
              >
                {m.sttCheckIntervalTooltip}
              </Typography>
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
                          min: MIN_STT_CHECK_INTERVAL,
                          step: 0.1,
                          'data-testid': `${name}-number-input`,
                        },
                      },
                      size: 'small',
                      sx: { minWidth: 80, maxWidth: 120 },
                    }}
                    formHelperTextProps={helperTextTestId(
                      `${name}-field-error-message`
                    )}
                  />
                ))}
              </Stack>
            </Stack>
          )}
        </Stack>

        <Stack gap={2}>
          <SettingsFieldLabel
            label={
              <>
                <WarningIcon
                  color="warning"
                  sx={{ fontSize: 26, verticalAlign: '-6px' }}
                />{' '}
                {m.technicalPreviewLegend}
              </>
            }
            description={
              <>
                {m.technicalPreviewDescription}
                <strong>{m.technicalPreviewWarning}</strong>
                {m.technicalPreviewDescriptionSuffix}{' '}
              </>
            }
            readMoreLink={TECHNICAL_PREVIEW_DOC_URL}
            readMoreText={m.technicalPreviewLinkText}
          />
          <Stack
            gap={2}
            sx={{
              [`.${formControlLabelClasses.root}`]: {
                marginRight: 0,
              },
            }}
          >
            <Stack
              direction="row"
              alignItems="center"
              data-testid="advanced-azure-discover"
            >
              <SwitchInput name="azureDiscover" label={m.azureDiscoverLabel} />
              <Tooltip
                title={
                  <Box data-testid="info-tooltip">
                    <Typography variant="caption">
                      {m.azureDiscoverTooltip}{' '}
                      <Link
                        href={m.azureDiscoverLink}
                        target="_blank"
                        rel="noopener noreferrer"
                        color="inherit"
                        sx={{ textDecorationColor: 'inherit' }}
                      >
                        {Messages.tooltipLinkText}
                      </Link>
                    </Typography>
                  </Box>
                }
                arrow
              >
                <IconButton size="small" data-testid="info-icon">
                  <InfoOutlinedIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            </Stack>
            <Stack
              direction="row"
              alignItems="center"
              data-testid="access-control"
            >
              <SwitchInput name="accessControl" label={m.accessControl} />
              <Tooltip
                title={
                  <Box data-testid="info-tooltip">
                    <Typography variant="caption">
                      {m.accessControlTooltip}{' '}
                      <Link
                        href={m.accessControlLink}
                        target="_blank"
                        rel="noopener noreferrer"
                        color="inherit"
                        sx={{ textDecorationColor: 'inherit' }}
                      >
                        {Messages.tooltipLinkText}
                      </Link>
                    </Typography>
                  </Box>
                }
                arrow
              >
                <IconButton size="small" data-testid="info-icon">
                  <InfoOutlinedIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            </Stack>
          </Stack>
        </Stack>

        <SettingsSubmitButton testId="advanced-button" />
      </Stack>
    </FormProvider>
  );
};
