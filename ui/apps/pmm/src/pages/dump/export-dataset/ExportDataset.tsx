import { zodResolver } from '@hookform/resolvers/zod';
import Alert from '@mui/material/Alert';
import Autocomplete from '@mui/material/Autocomplete';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker';
import { RadioGroup, SwitchInput } from '@percona/peak-ui';
import { SecretField } from 'components/secret-field';
import { SectionHeader } from 'components/section-header';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { useDumps, useStartDump } from 'hooks/api/useDump';
import { useServices } from 'hooks/api/useServices';
import { PMM_DUMP_PATH } from 'lib/constants';
import { enqueueSnackbar } from 'notistack';
import { type FC, useMemo } from 'react';
import { Controller, FormProvider, useForm, useWatch } from 'react-hook-form';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import { OrgRole } from 'types/user.types';
import { PMM_DUMP_DOCS_URL } from '../DumpPage.constants';
import { Messages as DumpMessages } from '../DumpPage.messages';
import { isDumpCreating } from '../DumpPage.utils';
import { Messages } from './ExportDataset.messages';
import {
  type ExportDatasetFormValues,
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
  const isAdmin = !!user?.isPMMAdmin;
  const {
    data: services,
    isLoading: isLoadingServices,
    isError: isServicesError,
    refetch: refetchServices,
  } = useServices({}, { enabled: isAdmin });
  const { data: dumps } = useDumps({ enabled: isAdmin });
  const { mutate: startDump, isPending } = useStartDump();
  const serviceOptions = useMemo(() => getServiceOptions(services), [services]);
  const serviceCount =
    isLoadingServices || isServicesError ? undefined : serviceOptions.length;
  const isAnotherCreating = !!dumps?.dumps.some(isDumpCreating);
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
  const [serviceScope, enableEncryption] = useWatch({
    control,
    name: ['serviceScope', 'enableEncryption'],
  });

  const onSubmit = (values: ExportDatasetFormValues) =>
    startDump(toStartDumpPayload(values), {
      onSuccess: () => {
        enqueueSnackbar(Messages.success, { variant: 'success' });
        navigate(PMM_DUMP_PATH);
      },
    });

  return (
    <Page
      title={Messages.pageTitle}
      roles={isAdmin ? undefined : [OrgRole.Admin]}
    >
      <FormProvider {...methods}>
        <Stack
          component="form"
          onSubmit={handleSubmit(onSubmit)}
          gap={4}
          sx={{ flex: 1 }}
          data-testid="create-dataset-form"
        >
          <Typography>
            {DumpMessages.description}{' '}
            <Link
              href={PMM_DUMP_DOCS_URL}
              target="_blank"
              rel="noopener noreferrer"
            >
              {DumpMessages.learnMore}
            </Link>
          </Typography>
          {isAnotherCreating && (
            <Alert severity="info">{DumpMessages.creatingBlocked}</Alert>
          )}

          <Stack gap={2}>
            <SectionHeader title={Messages.sections.services} />
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
            <RadioGroup
              name="serviceScope"
              radioGroupFieldProps={{ row: false }}
              options={[
                { label: Messages.allServices(serviceCount), value: 'all' },
                { label: Messages.selectedServices, value: 'selected' },
              ]}
            />
            {serviceScope === 'selected' && (
              <Controller
                name="serviceNames"
                control={control}
                render={({ field, fieldState }) => (
                  <Autocomplete
                    multiple
                    disableCloseOnSelect
                    options={serviceOptions}
                    value={field.value}
                    onChange={(_, selected) => field.onChange(selected)}
                    loading={isLoadingServices}
                    noOptionsText={Messages.noServices}
                    renderInput={(params) => (
                      <TextField
                        {...params}
                        label={Messages.services}
                        error={!!fieldState.error}
                        helperText={fieldState.error?.message}
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
            )}
          </Stack>

          <Stack gap={2}>
            <SectionHeader
              title={Messages.sections.timeRange}
              description={Messages.timeRangeDescription}
            />
            <Stack direction={{ xs: 'column', sm: 'row' }} gap={2}>
              <Controller
                name="startTime"
                control={control}
                rules={{ deps: 'endTime' }}
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
                rules={{ deps: 'startTime' }}
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
          </Stack>

          <Stack gap={2}>
            <SectionHeader title={Messages.sections.data} />
            <SwitchInput
              name="exportQan"
              label={Messages.exportQan}
              labelCaption={Messages.exportQanHelp}
            />
          </Stack>

          <Stack gap={2}>
            <SectionHeader title={Messages.sections.security} />
            <SwitchInput
              name="enableEncryption"
              label={Messages.enableEncryption}
              labelCaption={Messages.enableEncryptionHelp}
            />
            {enableEncryption && (
              <>
                <SecretField
                  name="encryptionPassword"
                  label={Messages.encryptionPassword}
                  helperText={Messages.passwordRules}
                  testId="dump-encryption-password"
                  isRequired
                />
                <Alert severity="warning">{Messages.passwordWarning}</Alert>
              </>
            )}
          </Stack>

          <Stack gap={2}>
            <SectionHeader title={Messages.sections.serverLoad} />
            <SwitchInput
              name="ignoreLoad"
              label={Messages.ignoreLoad}
              labelCaption={Messages.ignoreLoadHelp}
            />
          </Stack>

          <Stack
            direction="row"
            gap={2}
            sx={{
              position: 'sticky',
              bottom: 0,
              py: 2,
              mt: 'auto',
              zIndex: 1,
              bgcolor: 'background.paper',
              borderTop: 1,
              borderColor: 'divider',
            }}
          >
            <Button
              type="submit"
              variant="contained"
              data-testid="create-dataset-submit-button"
              disabled={isPending || !isValid || isAnotherCreating}
            >
              {isPending ? Messages.creating : Messages.create}
            </Button>
            <Button
              component={RouterLink}
              to={PMM_DUMP_PATH}
              data-testid="cancel-button"
              disabled={isPending}
            >
              {Messages.cancel}
            </Button>
          </Stack>
        </Stack>
      </FormProvider>
    </Page>
  );
};
