import { FC, useEffect, useMemo, useState } from 'react';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import CloseIcon from '@mui/icons-material/Close';
import DeleteOutlineOutlinedIcon from '@mui/icons-material/DeleteOutlineOutlined';
import Autocomplete from '@mui/material/Autocomplete';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Drawer from '@mui/material/Drawer';
import { formControlClasses } from '@mui/material/FormControl';
import IconButton from '@mui/material/IconButton';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { CodeBlock, SelectInput, TextInput } from '@percona/percona-ui';
import { zodResolver } from '@hookform/resolvers/zod';
import { AxiosError } from 'axios';
import { FormProvider, useFieldArray, useForm } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import {
  DRAWER_CLOSED_WIDTH,
  DRAWER_WIDTH,
} from 'components/sidebar/drawer/Drawer.constants';
import { useNavigation } from 'contexts/navigation/navigation.hooks';
import {
  useAdvisorCheck,
  useCreateAdvisorCheck,
  useTestAdvisorCheck,
  useUpdateAdvisorCheck,
} from 'hooks/api/useAdvisors';
import { useServices } from 'hooks/api/useServices';
import { ADVISOR_FAMILY, ADVISOR_INTERVAL } from 'lib/constants';
import { TestAdvisorCheckResult } from 'types/advisors.types';
import { VersionedService } from 'types/services.types';
import { ADVISOR_FAMILY_SERVICE_TYPE } from 'utils/advisors.utils';
import { helperTextTestId } from 'utils/mui.utils';
import { Messages } from './AdvisorCheckForm.messages';
import {
  FAMILY_OPTIONS,
  INTERVAL_OPTIONS,
  QUERY_TYPES_BY_FAMILY,
} from './AdvisorCheckForm.constants';
import {
  AdvisorCheckFormValues,
  advisorCheckFormSchema,
  emptyFormValues,
  toFormValues,
  toInput,
} from './AdvisorCheckForm.schema';

export type AdvisorCheckFormMode = 'create' | 'edit' | 'clone';

interface AdvisorCheckFormProps {
  open: boolean;
  mode: AdvisorCheckFormMode;
  // name of the source check to prefill from (edit/clone)
  checkName?: string;
  onClose: () => void;
}

export const AdvisorCheckForm: FC<AdvisorCheckFormProps> = ({
  open,
  mode,
  checkName,
  onClose,
}) => {
  const { navOpen } = useNavigation();
  // the overlay never covers the main navigation
  const sidebarWidth = navOpen ? DRAWER_WIDTH : DRAWER_CLOSED_WIDTH;

  const isEdit = mode === 'edit';
  const needsSource = mode === 'edit' || mode === 'clone';

  const {
    data: sourceCheck,
    isLoading: isSourceLoading,
    isError: isSourceError,
  } = useAdvisorCheck(needsSource && open ? checkName : undefined);

  const methods = useForm<AdvisorCheckFormValues>({
    resolver: zodResolver(advisorCheckFormSchema),
    defaultValues: emptyFormValues,
    mode: 'onChange',
  });
  const { control, handleSubmit, reset, watch, trigger, getValues } = methods;
  const { fields, append, remove } = useFieldArray({
    control,
    name: 'queries',
  });

  const [testServiceId, setTestServiceId] = useState<string | null>(null);
  const [testResults, setTestResults] = useState<
    TestAdvisorCheckResult[] | null
  >(null);
  const [testScriptOutput, setTestScriptOutput] = useState<string | null>(null);
  const [testError, setTestError] = useState<string | null>(null);

  // drop stale test state whenever the overlay opens
  useEffect(() => {
    if (open) {
      setTestServiceId(null);
      setTestResults(null);
      setTestScriptOutput(null);
      setTestError(null);
    }
  }, [open]);

  // (re)initialize the form whenever the overlay opens or its source loads
  useEffect(() => {
    if (!open) {
      return;
    }
    if (mode === 'create') {
      reset(emptyFormValues);
      return;
    }
    if (sourceCheck) {
      reset(toFormValues(sourceCheck, mode === 'clone'));
    }
  }, [open, mode, sourceCheck, reset]);

  const family = watch('family');
  const queryTypeOptions = QUERY_TYPES_BY_FAMILY[family] ?? [];

  // the test target must match the check's family
  const serviceType = ADVISOR_FAMILY_SERVICE_TYPE[family];
  const { data: servicesResponse } = useServices(
    { serviceType },
    { enabled: open && !!serviceType }
  );
  const serviceOptions = useMemo(
    () =>
      (Object.values(servicesResponse ?? {}).flat() as VersionedService[]).map(
        (s) => ({ id: s.serviceId, label: s.serviceName })
      ),
    [servicesResponse]
  );

  // a previously picked service is likely of the wrong type after a family change
  useEffect(() => {
    setTestServiceId(null);
  }, [family]);

  const { mutateAsync: create, isPending: isCreating } =
    useCreateAdvisorCheck();
  const { mutateAsync: update, isPending: isUpdating } =
    useUpdateAdvisorCheck();
  const isSaving = isCreating || isUpdating;
  const { mutateAsync: testCheck, isPending: isTesting } =
    useTestAdvisorCheck();

  const handleTest = async () => {
    const valid = await trigger();
    if (!valid || !testServiceId) {
      return;
    }
    setTestResults(null);
    setTestScriptOutput(null);
    setTestError(null);
    try {
      const { results, scriptOutput } = await testCheck({
        check: toInput(getValues()),
        serviceId: testServiceId,
      });
      setTestResults(results ?? []);
      setTestScriptOutput(scriptOutput || null);
    } catch (error) {
      const message =
        error instanceof AxiosError
          ? (error.response?.data?.message ?? error.message)
          : Messages.testFailed;
      setTestError(message);
    }
  };

  const onSubmit = async (values: AdvisorCheckFormValues) => {
    const input = toInput(values);
    try {
      if (isEdit && checkName) {
        await update({ name: checkName, check: input });
        enqueueSnackbar(Messages.success.updated(input.name), {
          variant: 'success',
        });
      } else {
        await create(input);
        enqueueSnackbar(Messages.success.created(input.name), {
          variant: 'success',
        });
      }
      onClose();
    } catch {
      // the request error is surfaced by the global interceptor; keep the form open
    }
  };

  const title =
    mode === 'edit'
      ? Messages.editTitle
      : mode === 'clone'
        ? Messages.cloneTitle
        : Messages.createTitle;

  const loadingSource = needsSource && isSourceLoading;

  return (
    <Drawer
      anchor="bottom"
      open={open}
      onClose={onClose}
      slotProps={{
        paper: {
          // @ts-expect-error data-testid is passed through to the DOM
          'data-testid': 'advisor-check-form',
          sx: {
            height: '100vh',
            left: { xs: 0, md: sidebarWidth },
            p: 2,
            display: 'flex',
            flexDirection: 'column',
          },
        },
      }}
    >
      <Stack direction="row" justifyContent="space-between" alignItems="center">
        <Typography variant="h6">{title}</Typography>
        <IconButton
          size="small"
          aria-label={Messages.close}
          onClick={onClose}
          data-testid="advisor-check-form-close"
        >
          <CloseIcon fontSize="small" />
        </IconButton>
      </Stack>

      {loadingSource ? (
        <CircularProgress
          size={24}
          sx={{ mt: 4, alignSelf: 'center' }}
          data-testid="advisor-check-form-loading"
        />
      ) : isSourceError ? (
        <Typography variant="body2" color="error" sx={{ mt: 2 }}>
          {Messages.loadError}
        </Typography>
      ) : (
        <FormProvider {...methods}>
          <Stack
            component="form"
            onSubmit={handleSubmit(onSubmit)}
            sx={{
              flex: 1,
              minHeight: 0,
              mt: 2,
              gap: 2,
              // consistent margins so fields align cleanly within rows
              [`.${formControlClasses.root}`]: { margin: 0 },
            }}
          >
            <Box
              sx={{
                flex: 1,
                minHeight: 0,
                overflow: 'auto',
                display: 'flex',
                flexDirection: 'column',
                gap: 2,
                // top padding so the first field's floating label isn't clipped,
                // right padding so the scrollbar doesn't overlay the inputs
                pt: 1,
                pr: 2,
              }}
            >
              <TextInput
                name="name"
                label={Messages.fields.name}
                textFieldProps={{
                  disabled: isEdit,
                  helperText: Messages.fields.nameHelper,
                  slotProps: { htmlInput: { 'data-testid': 'check-name' } },
                }}
                formHelperTextProps={helperTextTestId(
                  'name-field-error-message'
                )}
              />
              <TextInput
                name="summary"
                label={Messages.fields.summary}
                textFieldProps={{
                  slotProps: { htmlInput: { 'data-testid': 'check-summary' } },
                }}
                formHelperTextProps={helperTextTestId(
                  'summary-field-error-message'
                )}
              />
              <TextInput
                name="description"
                label={Messages.fields.description}
                textFieldProps={{
                  multiline: true,
                  minRows: 2,
                  slotProps: {
                    htmlInput: { 'data-testid': 'check-description' },
                  },
                }}
                formHelperTextProps={helperTextTestId(
                  'description-field-error-message'
                )}
              />
              <Stack direction="row" gap={2} flexWrap="wrap">
                <TextInput
                  name="category"
                  label={Messages.fields.category}
                  textFieldProps={{
                    sx: { flex: 1, minWidth: 160 },
                    slotProps: {
                      htmlInput: { 'data-testid': 'check-category' },
                    },
                  }}
                  formHelperTextProps={helperTextTestId(
                    'category-field-error-message'
                  )}
                />
                <TextInput
                  name="subcategory"
                  label={Messages.fields.subcategory}
                  textFieldProps={{
                    sx: { flex: 1, minWidth: 160 },
                    slotProps: {
                      htmlInput: { 'data-testid': 'check-subcategory' },
                    },
                  }}
                  formHelperTextProps={helperTextTestId(
                    'subcategory-field-error-message'
                  )}
                />
                <SelectInput
                  name="family"
                  label={Messages.fields.family}
                  formControlProps={{ sx: { flex: 1, minWidth: 160 } }}
                  selectFieldProps={{
                    // @ts-expect-error data-testid is passed through to the DOM
                    'data-testid': 'check-family-select',
                  }}
                >
                  {FAMILY_OPTIONS.map((option) => (
                    <MenuItem key={option} value={option}>
                      {ADVISOR_FAMILY[option]}
                    </MenuItem>
                  ))}
                </SelectInput>
                <SelectInput
                  name="interval"
                  label={Messages.fields.interval}
                  formControlProps={{ sx: { flex: 1, minWidth: 160 } }}
                  selectFieldProps={{
                    // @ts-expect-error data-testid is passed through to the DOM
                    'data-testid': 'check-interval-select',
                  }}
                >
                  {INTERVAL_OPTIONS.map((option) => (
                    <MenuItem key={option} value={option}>
                      {ADVISOR_INTERVAL[option]}
                    </MenuItem>
                  ))}
                </SelectInput>
              </Stack>

              <Stack gap={1}>
                <Typography variant="subtitle2">
                  {Messages.fields.queries}
                </Typography>
                {fields.map((field, index) => (
                  <Stack
                    key={field.id}
                    direction="row"
                    gap={2}
                    alignItems="flex-start"
                    data-testid={`check-query-${index}`}
                  >
                    <SelectInput
                      name={`queries.${index}.type`}
                      label={Messages.fields.queryType}
                      formControlProps={{ sx: { minWidth: 220 } }}
                      selectFieldProps={{
                        // @ts-expect-error data-testid is passed through to the DOM
                        'data-testid': `check-query-${index}-type-select`,
                      }}
                    >
                      {queryTypeOptions.map((option) => (
                        <MenuItem key={option} value={option}>
                          {option}
                        </MenuItem>
                      ))}
                    </SelectInput>
                    <TextInput
                      name={`queries.${index}.query`}
                      label={Messages.fields.query}
                      textFieldProps={{
                        multiline: true,
                        minRows: 1,
                        helperText: Messages.fields.queryHelper,
                        sx: { flex: 1 },
                        slotProps: {
                          htmlInput: {
                            'data-testid': `check-query-${index}-text`,
                          },
                        },
                      }}
                    />
                    <Tooltip title={Messages.removeQuery} arrow>
                      <Box component="span">
                        <IconButton
                          aria-label={Messages.removeQuery}
                          disabled={fields.length <= 1}
                          onClick={() => remove(index)}
                          data-testid={`check-query-${index}-remove`}
                        >
                          <DeleteOutlineOutlinedIcon />
                        </IconButton>
                      </Box>
                    </Tooltip>
                  </Stack>
                ))}
                <Button
                  size="small"
                  startIcon={<AddOutlinedIcon />}
                  onClick={() =>
                    append({ type: queryTypeOptions[0] ?? '', query: '' })
                  }
                  sx={{ alignSelf: 'flex-start' }}
                  data-testid="check-query-add"
                >
                  {Messages.addQuery}
                </Button>
              </Stack>

              <TextInput
                name="script"
                label={Messages.fields.script}
                textFieldProps={{
                  multiline: true,
                  minRows: 8,
                  sx: {
                    '& textarea': { fontFamily: 'monospace', fontSize: 13 },
                  },
                  slotProps: { htmlInput: { 'data-testid': 'check-script' } },
                }}
                formHelperTextProps={helperTextTestId(
                  'script-field-error-message'
                )}
              />
            </Box>

            {(testResults !== null || testError !== null) && (
              <Stack gap={1} data-testid="advisor-check-form-test-results">
                <Stack
                  direction="row"
                  justifyContent="space-between"
                  alignItems="center"
                >
                  <Typography variant="subtitle2">
                    {testError
                      ? Messages.testFailed
                      : Messages.testResults(testResults?.length ?? 0)}
                  </Typography>
                  <IconButton
                    size="small"
                    aria-label={Messages.closeResults}
                    onClick={() => {
                      setTestResults(null);
                      setTestScriptOutput(null);
                      setTestError(null);
                    }}
                    data-testid="advisor-check-form-test-results-close"
                  >
                    <CloseIcon fontSize="small" />
                  </IconButton>
                </Stack>
                {testError ? (
                  <Typography
                    variant="body2"
                    color="error"
                    sx={{ whiteSpace: 'pre-wrap' }}
                    data-testid="advisor-check-form-test-error"
                  >
                    {testError}
                  </Typography>
                ) : (
                  <CodeBlock
                    language="json"
                    copyable
                    content={JSON.stringify(testResults, null, 2)}
                    maxHeight="30vh"
                    sx={{ overflow: 'auto', m: 0 }}
                    data-testid="advisor-check-form-test-output"
                  />
                )}
                {testScriptOutput && (
                  <>
                    <Typography variant="subtitle2">
                      {Messages.scriptOutput}
                    </Typography>
                    <CodeBlock
                      copyable
                      content={testScriptOutput}
                      maxHeight="20vh"
                      sx={{ overflow: 'auto', m: 0 }}
                      data-testid="advisor-check-form-test-script-output"
                    />
                  </>
                )}
              </Stack>
            )}

            <Stack
              direction="row"
              alignItems="center"
              gap={1}
              // full-bleed toolbar: cancel the drawer padding so the darker
              // strip visually closes the form at the bottom edge
              sx={{
                mx: -2,
                mb: -2,
                px: 2,
                py: 1.5,
                borderTop: 1,
                borderColor: 'divider',
                bgcolor: 'background.default',
              }}
            >
              <Autocomplete
                size="small"
                sx={{ width: 280 }}
                options={serviceOptions}
                value={
                  serviceOptions.find((o) => o.id === testServiceId) ?? null
                }
                onChange={(_, option) => setTestServiceId(option?.id ?? null)}
                renderInput={(params) => (
                  <TextField {...params} label={Messages.testService} />
                )}
                data-testid="advisor-check-form-test-service"
              />
              <Button
                variant="outlined"
                disabled={!testServiceId || isTesting}
                onClick={handleTest}
                data-testid="advisor-check-form-test"
              >
                {isTesting ? Messages.testing : Messages.test}
              </Button>
              <Box sx={{ flex: 1 }} />
              <Button
                color="inherit"
                onClick={onClose}
                data-testid="advisor-check-form-cancel"
              >
                {Messages.cancel}
              </Button>
              <Button
                type="submit"
                variant="contained"
                disabled={isSaving}
                data-testid="advisor-check-form-save"
              >
                {Messages.save}
              </Button>
            </Stack>
          </Stack>
        </FormProvider>
      )}
    </Drawer>
  );
};
