import {
  Alert,
  Box,
  Button,
  FormControlLabel,
  IconButton,
  Link,
  Stack,
  Switch,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import LinkIcon from '@mui/icons-material/Link';
import { FC, useEffect } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import { useUpdateSettings } from 'hooks/api/useSettings';
import { Settings } from 'types/settings.types';
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

interface AdvancedSettingsFormProps {
  settings: Settings;
}

interface FormValues {
  retention: string;
  telemetry: boolean;
  updates: boolean;
  alerting: boolean;
  backup: boolean;
  enableInternalPgQan: boolean;
  publicAddress: string;
  stt: boolean;
  rareInterval: string;
  standardInterval: string;
  frequentInterval: string;
  azureDiscover: boolean;
  accessControl: boolean;
}

const LabelWithTooltip: FC<{
  label: string;
  tooltip: React.ReactNode;
  link?: string;
}> = ({ label, tooltip, link }) => (
  <Stack direction="row" alignItems="center" gap={0.5}>
    <Typography variant="body1">{label}</Typography>
    <Tooltip
      title={
        <Typography variant="body2">
          <Stack gap={0.5} p={1}>
            <Box>{tooltip}</Box>
            {link && (
              <Link
                href={link}
                target="_blank"
                rel="noopener noreferrer"
                color="inherit"
                variant="body2"
              >
                {Messages.tooltipLinkText}
              </Link>
            )}
          </Stack>
        </Typography>
      }
      arrow
    >
      <IconButton size="small" sx={{ p: 0.25 }}>
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
  } = useForm<FormValues>({
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

  const onSubmit = async (values: FormValues) => {
    try {
      const retentionNum = parseFloat(values.retention);
      const dataRetention = `${Math.round(retentionNum * SECONDS_IN_DAY)}s`;
      const sttCheckIntervals = {
        rareInterval: `${convertHoursStringToSeconds(values.rareInterval)}s`,
        standardInterval: `${convertHoursStringToSeconds(values.standardInterval)}s`,
        frequentInterval: `${convertHoursStringToSeconds(values.frequentInterval)}s`,
      };

      await updateSettings({
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
      });
      enqueueSnackbar(Messages.service.success, { variant: 'success' });
      reset(values);
    } catch (error) {
      enqueueSnackbar(
        error instanceof Error ? error.message : Messages.unauthorized,
        { variant: 'error' }
      );
    }
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
      gap={2}
      maxWidth={{ xs: '100%', md: 600 }}
    >
      <Stack direction="row" alignItems="center" gap={1}>
        <LabelWithTooltip
          label={m.retentionLabel}
          tooltip={m.retentionTooltip}
          link={m.retentionLink}
        />
      </Stack>
      <Stack direction="row" alignItems="center" gap={2}>
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
              inputProps={{ min: MIN_DAYS, max: MAX_DAYS, step: 1 }}
              sx={{ width: 120 }}
            />
          )}
        />
        <Typography variant="body2" color="text.secondary">
          {m.retentionUnits}
        </Typography>
      </Stack>

      {[
        {
          name: 'telemetry' as const,
          label: m.telemetryLabel,
          tooltip: (
            <Stack
              gap={1}
              maxHeight={300}
              sx={{
                mr: -1,
                overflowY: 'scroll',
                scrollbarColor: 'auto',
              }}
            >
              <Typography variant="body2">{m.telemetryTooltip}</Typography>
              <Typography variant="body2">{m.telemetrySummaryTitle}</Typography>
              <Box
                component="ul"
                sx={{
                  m: 0,
                  p: 0,
                  pl: 2,
                }}
              >
                {(settings.telemetrySummaries ?? []).map((s) => (
                  <Typography key={s} variant="body2" component="li">
                    {s}
                  </Typography>
                ))}
              </Box>
            </Stack>
          ),
          link: m.telemetryLink,
        },
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

      <Stack direction="row" alignItems="center" gap={1}>
        <LabelWithTooltip
          label={m.publicAddressLabel}
          tooltip={m.publicAddressTooltip}
        />
      </Stack>
      <Stack direction="row" gap={1} alignItems="center">
        <Controller
          name="publicAddress"
          control={control}
          render={({ field }) => (
            <TextField
              {...field}
              fullWidth
              size="small"
              placeholder="https://..."
            />
          )}
        />
        <Button
          type="button"
          variant="outlined"
          startIcon={<LinkIcon />}
          onClick={() =>
            setValue('publicAddress', window.location.host, {
              shouldDirty: true,
            })
          }
          sx={{
            width: 250,
          }}
        >
          {m.publicAddressButton}
        </Button>
      </Stack>

      <Controller
        name="stt"
        control={control}
        render={({ field }) => (
          <FormControlLabel
            control={<Switch {...field} checked={field.value} />}
            label={
              <LabelWithTooltip
                label={m.advisorsLabel}
                tooltip={m.advisorsTooltip}
                link={m.advisorsLink}
              />
            }
          />
        )}
      />

      <Stack direction="row" alignItems="center" gap={1}>
        <LabelWithTooltip
          label={m.sttCheckIntervalsLabel}
          tooltip={m.sttCheckIntervalTooltip}
        />
      </Stack>
      <Stack direction="row" gap={2} flexWrap="wrap">
        {STT_CHECK_INTERVALS.map(({ name, label }) => (
          <Controller
            key={name}
            name={name}
            control={control}
            rules={{ validate: validateInterval }}
            render={({ field, fieldState }) => (
              <Stack direction="row" alignItems="center" gap={1}>
                <TextField
                  {...field}
                  label={label}
                  type="number"
                  disabled={!sttEnabled}
                  error={!!fieldState.error}
                  helperText={fieldState.error?.message}
                  inputProps={{ min: MIN_STT_CHECK_INTERVAL, step: 0.1 }}
                  sx={{ width: 100 }}
                />
                <Typography variant="body2" color="text.secondary">
                  {m.sttCheckIntervalUnit}
                </Typography>
              </Stack>
            )}
          />
        ))}
      </Stack>

      <Box
        component="fieldset"
        sx={{
          border: 1,
          borderColor: 'divider',
          borderRadius: 1,
          p: 2,
        }}
      >
        <Typography component="legend" variant="subtitle2" sx={{ px: 1 }}>
          {m.technicalPreviewLegend}
        </Typography>
        <Alert severity="info" sx={{ mt: 1, mb: 2 }}>
          {m.technicalPreviewDescription}{' '}
          <Link
            href={TECHNICAL_PREVIEW_DOC_URL}
            target="_blank"
            rel="noreferrer"
          >
            {m.technicalPreviewLinkText}
          </Link>
        </Alert>
        <Stack gap={1}>
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
      </Box>

      <Button
        type="submit"
        variant="contained"
        disabled={!isDirty || isPending || Object.keys(errors).length > 0}
        data-testid="advanced-settings-submit"
      >
        {isPending ? 'Applying...' : action}
      </Button>
    </Stack>
  );
};
