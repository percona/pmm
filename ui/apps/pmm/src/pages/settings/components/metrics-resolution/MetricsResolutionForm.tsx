import Button from '@mui/material/Button';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Radio from '@mui/material/Radio';
import RadioGroup from '@mui/material/RadioGroup';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { FC, useEffect, useMemo } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import { useUpdateSettings } from 'hooks/api/useSettings';
import { MetricsResolutions } from 'types/settings.types';
import { Messages } from '../../Settings.messages';
import {
  defaultResolutions,
  resolutionOptions,
  RESOLUTION_MAX,
  RESOLUTION_MIN,
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

const DEFAULT_METRICS = { hr: '5s', mr: '10s', lr: '60s' } as const;

export const MetricsResolutionForm: FC<MetricsResolutionFormProps> = ({
  settings,
}) => {
  const { mutateAsync: updateSettings, isPending } = useUpdateSettings();
  const metricsResolutions = useMemo(
    () => settings.metricsResolutions ?? DEFAULT_METRICS,
    [settings.metricsResolutions]
  );
  const preset = getResolutionPreset(metricsResolutions);
  const raw = removeUnits(metricsResolutions);

  const {
    control,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { isDirty, errors },
  } = useForm<MetricsResolutionFormValues>({
    defaultValues: {
      preset,
      lr: raw.lr,
      mr: raw.mr,
      hr: raw.hr,
    },
  });

  const currentPreset = watch('preset');

  useEffect(() => {
    reset({
      preset: getResolutionPreset(metricsResolutions),
      ...removeUnits(metricsResolutions),
    });
  }, [metricsResolutions, reset]);

  useEffect(() => {
    if (currentPreset && currentPreset !== 'custom') {
      const idx = resolutionOptions.findIndex((o) => o.value === currentPreset);
      if (idx >= 0) {
        const def = removeUnits(defaultResolutions[idx]);
        setValue('lr', def.lr, { shouldDirty: false });
        setValue('mr', def.mr, { shouldDirty: false });
        setValue('hr', def.hr, { shouldDirty: false });
      }
    }
  }, [currentPreset, setValue]);

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
          reset({
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

  const { label, link, tooltip, action, intervals } = Messages.metrics;
  const { tooltipLinkText } = Messages;

  const validateNumber = (v: string) => {
    const n = parseInt(v, 10);
    if (isNaN(n) || v === '') return 'Required';
    if (n < RESOLUTION_MIN || n > RESOLUTION_MAX)
      return `Must be between ${RESOLUTION_MIN} and ${RESOLUTION_MAX}`;
    return true;
  };

  return (
    <Stack
      component="form"
      onSubmit={handleSubmit(onSubmit)}
      gap={2}
      maxWidth={600}
    >
      <Stack direction="row" alignItems="center" gap={1}>
        <Typography variant="body1" fontWeight={500}>
          {label}
        </Typography>
        <Tooltip
          title={
            <Stack gap={0.5}>
              <Typography variant="body2">{tooltip}</Typography>
              <Link
                href={link}
                target="_blank"
                rel="noopener noreferrer"
                color="inherit"
                sx={{ textDecoration: 'underline' }}
              >
                {tooltipLinkText}
              </Link>
            </Stack>
          }
          arrow
        >
          <IconButton size="small" aria-label={tooltip} sx={{ p: 0.5 }}>
            <InfoOutlinedIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      </Stack>

      <Controller
        name="preset"
        control={control}
        render={({ field }) => (
          <FormControl>
            <RadioGroup {...field} row>
              {resolutionOptions.map((opt) => (
                <FormControlLabel
                  key={opt.value}
                  value={opt.value}
                  control={<Radio />}
                  label={opt.label}
                />
              ))}
            </RadioGroup>
          </FormControl>
        )}
      />

      <Stack direction="row" gap={2} flexWrap="wrap">
        <Controller
          name="lr"
          control={control}
          rules={{ validate: validateNumber }}
          render={({ field, fieldState }) => (
            <TextField
              {...field}
              label={intervals.low}
              type="number"
              disabled={currentPreset !== 'custom'}
              error={!!fieldState.error}
              helperText={fieldState.error?.message}
              slotProps={{
                htmlInput: { min: RESOLUTION_MIN, max: RESOLUTION_MAX },
              }}
              data-testid="metrics-resolution-lr"
              sx={{ minWidth: 120 }}
            />
          )}
        />
        <Controller
          name="mr"
          control={control}
          rules={{ validate: validateNumber }}
          render={({ field, fieldState }) => (
            <TextField
              {...field}
              label={intervals.medium}
              type="number"
              disabled={currentPreset !== 'custom'}
              error={!!fieldState.error}
              helperText={fieldState.error?.message}
              slotProps={{
                htmlInput: { min: RESOLUTION_MIN, max: RESOLUTION_MAX },
              }}
              data-testid="metrics-resolution-mr"
              sx={{ minWidth: 120 }}
            />
          )}
        />
        <Controller
          name="hr"
          control={control}
          rules={{ validate: validateNumber }}
          render={({ field, fieldState }) => (
            <TextField
              {...field}
              label={intervals.high}
              type="number"
              disabled={currentPreset !== 'custom'}
              error={!!fieldState.error}
              helperText={fieldState.error?.message}
              slotProps={{
                htmlInput: { min: RESOLUTION_MIN, max: RESOLUTION_MAX },
              }}
              data-testid="metrics-resolution-hr"
              sx={{ minWidth: 120 }}
            />
          )}
        />
      </Stack>

      <Button
        type="submit"
        variant="contained"
        disabled={!isDirty || isPending || Object.keys(errors).length > 0}
        data-testid="metrics-resolution-submit"
      >
        {isPending ? 'Applying...' : action}
      </Button>
    </Stack>
  );
};
