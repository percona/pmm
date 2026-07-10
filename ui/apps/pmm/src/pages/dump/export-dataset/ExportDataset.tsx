import { zodResolver } from '@hookform/resolvers/zod';
import {
  Alert,
  Autocomplete,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { DateTimePicker } from '@mui/x-date-pickers';
import { SwitchInput, TextInput } from '@percona/percona-ui';
import { FC, useMemo } from 'react';
import { Controller, FormProvider, useForm, useWatch } from 'react-hook-form';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import { enqueueSnackbar } from 'notistack';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { useStartDump } from 'hooks/api/useDump';
import { useManagedServices } from 'hooks/api/useServices';
import { OrgRole } from 'types/user.types';
import { Messages } from './ExportDataset.messages';
import {
  ExportDatasetFormValues,
  exportDatasetSchema,
} from './ExportDataset.schema';
import {
  getDefaultValues,
  getServiceOptions,
  toStartDumpPayload,
} from './ExportDataset.utils';

export const ExportDataset: FC = () => {
  const { user } = useUser();
  const navigate = useNavigate();
  const {
    data,
    isLoading: isLoadingServices,
    isError: isServicesError,
    refetch: refetchServices,
  } = useManagedServices({}, { enabled: !!user?.isPMMAdmin });
  const { mutateAsync: startDump, isPending } = useStartDump();
  const serviceOptions = useMemo(
    () => getServiceOptions(data?.services ?? []),
    [data]
  );
  const methods = useForm<ExportDatasetFormValues>({
    resolver: zodResolver(exportDatasetSchema),
    defaultValues: getDefaultValues(),
    mode: 'onChange',
  });
  const {
    control,
    formState: { isValid },
    handleSubmit,
  } = methods;
  const enableEncryption = useWatch({ control, name: 'enableEncryption' });

  const onSubmit = async (values: ExportDatasetFormValues) => {
    await startDump(toStartDumpPayload(values), {
      onSuccess: () => {
        enqueueSnackbar(Messages.success, { variant: 'success' });
        navigate('/pmm-dump');
      },
    });
  };

  return (
    <Page
      title={Messages.pageTitle}
      surface="paper"
      roles={user?.isPMMAdmin ? undefined : [OrgRole.Admin]}
    >
      <Card variant="outlined">
        <CardContent>
          <FormProvider {...methods}>
            <Stack component="form" onSubmit={handleSubmit(onSubmit)} gap={2}>
              <Typography>{Messages.summary}</Typography>
              <Typography variant="h6">{Messages.title}</Typography>
              {isServicesError && (
                <Alert
                  severity="error"
                  action={
                    <Button color="inherit" onClick={() => refetchServices()}>
                      {Messages.retry}
                    </Button>
                  }
                >
                  {Messages.servicesError}
                </Alert>
              )}

              <Controller
                name="serviceNames"
                control={control}
                render={({ field }) => (
                  <Autocomplete
                    multiple
                    disableCloseOnSelect
                    options={serviceOptions}
                    value={serviceOptions.filter(({ value }) =>
                      field.value.includes(value)
                    )}
                    onChange={(_, selected) =>
                      field.onChange(selected.map(({ value }) => value))
                    }
                    loading={isLoadingServices}
                    noOptionsText={Messages.noServices}
                    getOptionLabel={({ label }) => label}
                    isOptionEqualToValue={(option, value) =>
                      option.value === value.value
                    }
                    renderInput={(params) => (
                      <TextField
                        {...params}
                        label={Messages.services}
                        placeholder={Messages.allServices}
                        slotProps={{
                          input: {
                            ...params.InputProps,
                            endAdornment: (
                              <>
                                {isLoadingServices && (
                                  <CircularProgress size={20} />
                                )}
                                {params.InputProps.endAdornment}
                              </>
                            ),
                          },
                          htmlInput: {
                            ...params.inputProps,
                            'data-testid': 'service-select-input',
                          },
                        }}
                      />
                    )}
                  />
                )}
              />

              <Stack direction={{ xs: 'column', sm: 'row' }} gap={2}>
                <Controller
                  name="startTime"
                  control={control}
                  render={({ field, fieldState }) => (
                    <DateTimePicker
                      label={Messages.startTime}
                      value={field.value}
                      onChange={(value) => value && field.onChange(value)}
                      maxDateTime={new Date()}
                      slotProps={{
                        textField: {
                          error: !!fieldState.error,
                          helperText: fieldState.error?.message,
                          inputProps: {
                            'data-testid': 'dump-start-time-input',
                          },
                        },
                      }}
                    />
                  )}
                />
                <Controller
                  name="endTime"
                  control={control}
                  render={({ field, fieldState }) => (
                    <DateTimePicker
                      label={Messages.endTime}
                      value={field.value}
                      onChange={(value) => value && field.onChange(value)}
                      maxDateTime={new Date()}
                      slotProps={{
                        textField: {
                          error: !!fieldState.error,
                          helperText: fieldState.error?.message,
                          inputProps: {
                            'data-testid': 'dump-end-time-input',
                          },
                        },
                      }}
                    />
                  )}
                />
              </Stack>

              <Stack gap={1}>
                <span data-testid="pmm-dump-export-qan">
                  <SwitchInput name="exportQan" label={Messages.exportQan} />
                </span>
                <Typography variant="body2" color="text.secondary">
                  {Messages.exportQanHelp}
                </Typography>
                <span data-testid="pmm-dump-ignore-load">
                  <SwitchInput name="ignoreLoad" label={Messages.ignoreLoad} />
                </span>
                <Typography variant="body2" color="text.secondary">
                  {Messages.ignoreLoadHelp}
                </Typography>
                <span data-testid="pmm-dump-enable-encryption">
                  <SwitchInput
                    name="enableEncryption"
                    label={Messages.enableEncryption}
                  />
                </span>
                <Typography variant="body2" color="text.secondary">
                  {Messages.enableEncryptionHelp}
                </Typography>
              </Stack>

              {enableEncryption && (
                <TextInput
                  name="encryptionPassword"
                  label={Messages.encryptionPassword}
                  textFieldProps={{
                    type: 'password',
                    placeholder: Messages.encryptionPasswordPlaceholder,
                    slotProps: {
                      htmlInput: {
                        'data-testid': 'dump-encryption-password',
                      },
                    },
                  }}
                />
              )}

              <Stack direction="row" gap={1} justifyContent="flex-end">
                <Button
                  component={RouterLink}
                  to="/pmm-dump"
                  variant="outlined"
                  data-testid="cancel-button"
                  disabled={isPending}
                >
                  {Messages.cancel}
                </Button>
                <Button
                  type="submit"
                  variant="contained"
                  data-testid="create-dataset-submit-button"
                  disabled={
                    isPending ||
                    !isValid ||
                    serviceOptions.length === 0 ||
                    isServicesError
                  }
                >
                  {isPending ? Messages.creating : Messages.create}
                </Button>
              </Stack>
            </Stack>
          </FormProvider>
        </CardContent>
      </Card>
    </Page>
  );
};
