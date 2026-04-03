import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import FormControlLabel from '@mui/material/FormControlLabel';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Switch from '@mui/material/Switch';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import ArrowOutwardIcon from '@mui/icons-material/ArrowOutward';
import CloseIcon from '@mui/icons-material/Close';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import WarningIcon from '@mui/icons-material/Warning';
import { FC, useEffect, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import { useUpdateSettings } from 'hooks/api/useSettings';
import { Messages } from '../../Settings.messages';
import {
  MAX_DAYS,
  MIN_DAYS,
  MIN_STT_CHECK_INTERVAL,
  SECONDS_IN_DAY,
  STT_CHECK_INTERVALS,
  TECHNICAL_PREVIEW_DOC_URL,
} from './Advanced.constants';
import {
  convertCheckIntervalsToHours,
  convertHoursStringToSeconds,
  convertSecondsToDays,
} from './Advanced.utils';
import {
  AdvancedSettingsFormProps,
  AdvancedSettingsFormValues,
  LabelWithTooltipProps,
} from './AdvancedSettingsForm.types';

const LabelWithTooltip: FC<LabelWithTooltipProps> = ({
  label,
  tooltip,
  link,
}) => (
  <Stack direction="row" alignItems="center">
    <Typography variant="body1">{label}</Typography>
    <Tooltip
      title={
        <Typography variant="caption">
            {tooltip}
            {' '}
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
      }
      arrow
    >
      <IconButton size="small">
        <InfoOutlinedIcon fontSize="small" />
      </IconButton>
    </Tooltip>
  </Stack>
);

export const AdvancedSettingsForm: FC<AdvancedSettingsFormProps> = ({
  settings,
}) => {
  const { mutateAsync: updateSettings, isPending } = useUpdateSettings();
  const intervals = convertCheckIntervalsToHours(settings.advisorRunIntervals);

  const {
    control,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { isDirty, errors },
  } = useForm<AdvancedSettingsFormValues>({
    defaultValues: {
      retention: String(
        convertSecondsToDays(settings.dataRetention ?? '86400s') || '1'
      ),
      telemetry: settings.telemetryEnabled ?? false,
      updates: settings.updatesEnabled ?? false,
      alerting: settings.alertingEnabled ?? false,
      backup: settings.backupManagementEnabled ?? false,
      enableInternalPgQan: settings.enableInternalPgQan ?? false,
      publicAddress: settings.pmmPublicAddress ?? '',
      stt: settings.advisorEnabled ?? false,
      rareInterval: intervals.rareInterval,
      standardInterval: intervals.standardInterval,
      frequentInterval: intervals.frequentInterval,
      azureDiscover: settings.azurediscoverEnabled ?? false,
      accessControl: settings.enableAccessControl ?? false,
    },
  });

  const sttEnabled = watch('stt');
  const [telemetryDialogOpen, setTelemetryDialogOpen] = useState(false);

  useEffect(() => {
    reset({
      retention: String(
        convertSecondsToDays(settings.dataRetention ?? '86400s') || '1'
      ),
      telemetry: settings.telemetryEnabled ?? false,
      updates: settings.updatesEnabled ?? false,
      alerting: settings.alertingEnabled ?? false,
      backup: settings.backupManagementEnabled ?? false,
      enableInternalPgQan: settings.enableInternalPgQan ?? false,
      publicAddress: settings.pmmPublicAddress ?? '',
      stt: settings.advisorEnabled ?? false,
      ...convertCheckIntervalsToHours(settings.advisorRunIntervals),
      azureDiscover: settings.azurediscoverEnabled ?? false,
      accessControl: settings.enableAccessControl ?? false,
    });
  }, [settings, reset]);

  const onSubmit = async (values: AdvancedSettingsFormValues) => {
    const retentionNum = parseFloat(values.retention);
    const dataRetention = `${Math.round(retentionNum * SECONDS_IN_DAY)}s`;
    const sttCheckIntervals = {
      rareInterval: `${convertHoursStringToSeconds(values.rareInterval)}s`,
      standardInterval: `${convertHoursStringToSeconds(values.standardInterval)}s`,
      frequentInterval: `${convertHoursStringToSeconds(values.frequentInterval)}s`,
    };

    await updateSettings(
      {
        dataRetention,
        pmmPublicAddress: values.publicAddress,
        enableTelemetry: values.telemetry,
        enableUpdates: values.updates,
        enableAlerting: values.alerting,
        enableBackupManagement: values.backup,
        enableInternalPgQan: values.enableInternalPgQan,
        enableAdvisor: values.stt,
        advisorRunIntervals: values.stt ? sttCheckIntervals : undefined,
        enableAzurediscover: values.azureDiscover,
        enableAccessControl: values.accessControl,
      },
      {
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
      }
    );
  };

  const { action } = Messages.advanced;
  const m = Messages.advanced;

  const validateRetention = (v: string) => {
    const n = parseFloat(v);
    if (isNaN(n) || v === '') return 'Required';
    if (n < MIN_DAYS || n > MAX_DAYS)
      return `Must be between ${MIN_DAYS} and ${MAX_DAYS}`;
    return true;
  };

  const validateInterval = (v: string) => {
    const n = parseFloat(v);
    if (isNaN(n) || v === '') return 'Required';
    if (n < MIN_STT_CHECK_INTERVAL) return `Min ${MIN_STT_CHECK_INTERVAL}`;
    return true;
  };

  return (
    <Stack
      component="form"
      onSubmit={handleSubmit(onSubmit)}
      gap={5}
    >
      <Stack gap={1}>
        <Stack maxWidth={640}>
          <Typography variant="h6">
            {m.publicAddressLabel}
          </Typography>
          <Typography variant="body2">
            {m.publicAddressTooltip}
          </Typography>
        </Stack>
        <Stack direction="row" flexWrap="wrap" gap={2} alignItems="center">
          <Controller
            name="publicAddress"
            control={control}
            render={({ field }) => (
              <TextField
                {...field}
                size="small"
                placeholder="https://..."
                sx={{ flex: 1, minWidth: 240 }}
              />
            )}
          />
          <Button
            type="button"
            variant="text"
            startIcon={<ContentCopyIcon />}
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
        <Stack maxWidth={640}>
          <Typography variant="h6">
            {m.retentionLabel}
          </Typography>
          <Typography variant="body2">
            {m.retentionTooltip}
            {' '}
            <Link
              href={m.retentionLink}
              target="_blank"
              rel="noopener noreferrer"
            >
              {Messages.tooltipLinkText}
              <ArrowOutwardIcon sx={{ fontSize: 14 }} />
            </Link>
          </Typography>
        </Stack>
        <Stack direction="row" alignItems="baseline" gap={1}>
          <Controller
            name="retention"
            control={control}
            rules={{ validate: validateRetention }}
            render={({ field, fieldState }) => (
              <TextField
                {...field}
                type="number"
                error={!!fieldState.error}
                helperText={fieldState.error?.message}
                slotProps={{
                  htmlInput: { min: MIN_DAYS, max: MAX_DAYS, step: 1 },
                }}
                sx={{ minWidth: 120, maxWidth: 240 }}
                size="small"
              />
            )}
          />
          <Typography variant="body1" color="text.secondary">
            {m.retentionUnits}
          </Typography>
        </Stack>
      </Stack>

      <Stack gap={1}>
        <Stack maxWidth={640}>
          <Typography variant="h6">
            {m.telemetryLabel}
          </Typography>
          <Typography variant="body2">
            {m.telemetryTooltip}
            {' '}
            <Link
              href={m.telemetryLink}
              target="_blank"
              rel="noopener noreferrer"
            >
              {Messages.tooltipLinkText}
              <ArrowOutwardIcon sx={{ fontSize: 14 }} />
            </Link>
          </Typography>
        </Stack>
        <Stack direction="row" alignItems="center" gap={1}>
          <Controller
            name="telemetry"
            control={control}
            render={({ field }) => (
              <FormControlLabel
                control={<Switch {...field} checked={field.value} />}
                label={m.telemetryLabel}
                sx={{ mr: 0 }}
              />
            )}
          />
          <Link
            component="button"
            type="button"
            variant="body1"
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
        >
          <DialogTitle>
            {m.telemetryLabel}
            <IconButton
              aria-label="close"
              size='medium'
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
              <Typography key={s} variant="body1" component="li" sx={{ ml: 3 }}>
                {s}
              </Typography>
            ))}
          </DialogContent>
        </Dialog>
      </Stack>

      <Stack gap={2}>
        <Stack maxWidth={640}>
          <Typography variant="h6">
            Feature management
          </Typography>
          <Typography variant="body2">
            Enable or disable core PMM capabilities. Turning off unused features can help conserve system resources and simplify your navigation menu.
          </Typography>
        </Stack>
        <Stack gap={2}>
          {[
            {
              name: 'updates' as const,
              label: m.updatesLabel,
              tooltip: m.updatesTooltip,
              link: m.updatesLink,
            },
            {
              name: 'alerting' as const,
              label: m.alertingLabel,
              tooltip: m.alertingTooltip,
              link: m.alertingLink,
            },
            {
              name: 'backup' as const,
              label: m.backupLabel,
              tooltip: m.backupTooltip,
              link: m.backupLink,
            },
            {
              name: 'enableInternalPgQan' as const,
              label: m.enableInternalPgQanLabel,
              tooltip: m.enableInternalPgQanTooltip,
              link: m.enableInternalPgQanLink,
            },
          ].map(({ name, label, tooltip, link }) => (
            <Controller
              key={name}
              name={name}
              control={control}
              render={({ field }) => (
                <FormControlLabel
                  control={<Switch {...field} checked={field.value} />}
                  label={
                    <LabelWithTooltip label={label} tooltip={tooltip} link={link} />
                  }
                />
              )}
            />
          ))}
        </Stack>
      </Stack>

      <Stack gap={1}>
        <Stack gap={1}>
          <Stack maxWidth={640}>
            <Typography variant="h6">
              {m.advisorsLabel}
            </Typography>
            <Typography variant="body2">
              {m.advisorsTooltip}
              {' '}
              <Link
                href={m.advisorsLink}
                target="_blank"
                rel="noopener noreferrer"
              >
                {Messages.tooltipLinkText}
                <ArrowOutwardIcon sx={{ fontSize: 14 }} />
              </Link>
            </Typography>
          </Stack>
          <Controller
            name="stt"
            control={control}
            render={({ field }) => (
              <FormControlLabel
                control={<Switch {...field} checked={field.value} />}
                label={m.advisorsLabel}
              />
            )}
          />
        </Stack>
        {sttEnabled && (
          <Stack gap={2}>
            <Typography variant="body1" sx={{ maxWidth: 640 }}>
              {m.sttCheckIntervalTooltip}
            </Typography>
            <Stack direction="row" columnGap={2} rowGap={3} flexWrap="wrap">
              {STT_CHECK_INTERVALS.map(({ name, label }) => (
                <Controller
                  key={name}
                  name={name}
                  control={control}
                  rules={{ validate: validateInterval }}
                  render={({ field, fieldState }) => (
                    <TextField
                      {...field}
                      label={label}
                      type="number"
                      error={!!fieldState.error}
                      helperText={fieldState.error?.message}
                      slotProps={{
                        htmlInput: { min: MIN_STT_CHECK_INTERVAL, step: 0.1 },
                      }}
                      size="small"
                      sx={{ minWidth: 80, maxWidth: 120 }}
                    />
                  )}
                />
              ))}
            </Stack>
          </Stack>
        )}
      </Stack>

      <Stack gap={2}>
        <Stack maxWidth={640}>
          <Typography variant="h6">
            <WarningIcon color="warning" sx={{ fontSize: 26, verticalAlign: '-6px' }} /> {m.technicalPreviewLegend}
          </Typography>
          <Typography variant="body2">
            {m.technicalPreviewDescription}
            <strong>{m.technicalPreviewWarning}</strong>
            {m.technicalPreviewDescriptionSuffix}
            {' '}
            <Link
              href={TECHNICAL_PREVIEW_DOC_URL}
              target="_blank"
              rel="noreferrer"
            >
              {m.technicalPreviewLinkText}
              <ArrowOutwardIcon sx={{ fontSize: 14 }} />
            </Link>
            {'.'}
          </Typography>
        </Stack>
        <Stack gap={2}>
          <Controller
            name="azureDiscover"
            control={control}
            render={({ field }) => (
              <FormControlLabel
                control={<Switch {...field} checked={field.value} />}
                label={
                  <LabelWithTooltip
                    label={m.azureDiscoverLabel}
                    tooltip={m.azureDiscoverTooltip}
                    link={m.azureDiscoverLink}
                  />
                }
              />
            )}
          />
          <Controller
            name="accessControl"
            control={control}
            render={({ field }) => (
              <FormControlLabel
                control={<Switch {...field} checked={field.value} />}
                label={
                  <LabelWithTooltip
                    label={m.accessControl}
                    tooltip={m.accessControlTooltip}
                    link={m.accessControlLink}
                  />
                }
              />
            )}
          />
        </Stack>
      </Stack>

      <Stack
        sx={{
          position: 'sticky',
          bottom: 0,
          py: 2,
          bgcolor: 'background.default',
          borderTop: 1,
          borderColor: 'divider',
          mt: 'auto',
          zIndex: 1,
          boxShadow: (theme) =>
            `-8px 0 0 0 ${theme.palette.background.default}, 30px 0 0 0 ${theme.palette.background.default}`,
        }}
      >
        <Button
          type="submit"
          variant="contained"
          disabled={!isDirty || isPending || Object.keys(errors).length > 0}
          data-testid="advanced-settings-submit"
          sx={{ alignSelf: 'flex-start' }}
        >
          {isPending ? 'Applying...' : action}
        </Button>
      </Stack>
    </Stack>
  );
};
