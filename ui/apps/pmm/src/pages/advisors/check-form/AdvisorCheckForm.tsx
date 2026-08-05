import { FC, useEffect, useMemo } from 'react';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import CloseIcon from '@mui/icons-material/Close';
import DeleteOutlineOutlinedIcon from '@mui/icons-material/DeleteOutlineOutlined';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Drawer from '@mui/material/Drawer';
import { formControlClasses } from '@mui/material/FormControl';
import IconButton from '@mui/material/IconButton';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { AutoCompleteInput, SelectInput, TextInput } from '@percona/percona-ui';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useFieldArray, useForm } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import {
  DRAWER_CLOSED_WIDTH,
  DRAWER_WIDTH,
} from 'components/sidebar/drawer/Drawer.constants';
import { useNavigation } from 'contexts/navigation/navigation.hooks';
import {
  useAdvisorCheck,
  useAdvisors,
  useCreateAdvisorCheck,
  useUpdateAdvisorCheck,
} from 'hooks/api/useAdvisors';
import { ADVISOR_TECHNOLOGY, ADVISOR_INTERVAL } from 'lib/constants';
import { flattenAdvisorChecks } from 'utils/advisors.utils';
import { CheckTestControls } from '../check-test/CheckTestControls';
import { CheckTestResults } from '../check-test/CheckTestResults';
import { useCheckTest } from '../check-test/useCheckTest';
import { Messages } from './AdvisorCheckForm.messages';
import {
  TECHNOLOGY_OPTIONS,
  INTERVAL_OPTIONS,
  QUERY_TYPES_BY_TECHNOLOGY,
  SCRIPT_FIELD_MIN_HEIGHT,
} from './AdvisorCheckForm.constants';
import {
  AdvisorCheckFormValues,
  advisorCheckFormSchema,
  emptyFormValues,
  toFormValues,
  toInput,
} from './AdvisorCheckForm.schema';
import { ScriptEditorInput } from './ScriptEditorInput';

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

  const technology = watch('technology');
  const queryTypeOptions = QUERY_TYPES_BY_TECHNOLOGY[technology] ?? [];

  // suggest the categories already in use; the query is shared with the list
  // page, so opening the form costs no extra request
  const { data: advisors = [] } = useAdvisors();
  const { categoryOptions, subcategoryOptions } = useMemo(() => {
    const rows = flattenAdvisorChecks(advisors);
    return {
      categoryOptions: [...new Set(rows.map((row) => row.category))].sort(),
      subcategoryOptions: [
        ...new Set(rows.map((row) => row.subcategory)),
      ].sort(),
    };
  }, [advisors]);

  const test = useCheckTest({ technology, enabled: open, resetKey: open });

  const { mutateAsync: create, isPending: isCreating } =
    useCreateAdvisorCheck();
  const { mutateAsync: update, isPending: isUpdating } =
    useUpdateAdvisorCheck();
  const isSaving = isCreating || isUpdating;

  const handleTest = async () => {
    const valid = await trigger();
    if (!valid) {
      return;
    }
    await test.runTest(toInput(getValues()));
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
              />
              <TextInput
                name="summary"
                label={Messages.fields.summary}
                textFieldProps={{
                  slotProps: { htmlInput: { 'data-testid': 'check-summary' } },
                }}
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
              />
              <Stack direction="row" gap={2} flexWrap="wrap">
                <AutoCompleteInput
                  name="category"
                  label={Messages.fields.category}
                  options={categoryOptions}
                  autoCompleteProps={{
                    // existing values are suggestions, not a closed list, and
                    // autoSelect commits a typed-in value on blur (freeSolo
                    // alone only commits on Enter)
                    freeSolo: true,
                    autoSelect: true,
                    // percona-ui defaults the Autocomplete to mt: 3
                    sx: { mt: 0, flex: 1, minWidth: 160 },
                  }}
                />
                <AutoCompleteInput
                  name="subcategory"
                  label={Messages.fields.subcategory}
                  options={subcategoryOptions}
                  autoCompleteProps={{
                    freeSolo: true,
                    autoSelect: true,
                    sx: { mt: 0, flex: 1, minWidth: 160 },
                  }}
                />
                <SelectInput
                  name="technology"
                  label={Messages.fields.technology}
                  formControlProps={{ sx: { flex: 1, minWidth: 160 } }}
                  selectFieldProps={{
                    // @ts-expect-error data-testid is passed through to the DOM
                    'data-testid': 'check-technology-select',
                  }}
                >
                  {TECHNOLOGY_OPTIONS.map((option) => (
                    <MenuItem key={option} value={option}>
                      {ADVISOR_TECHNOLOGY[option]}
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
                  sx: {
                    // fill the leftover height of the scroll area instead of
                    // sitting at the editor's minimum, so the field matches the
                    // details pane; a long script scrolls inside it
                    flex: 1,
                    '& .MuiInputBase-root': {
                      flex: 1,
                      // the editor's own minHeight lives inside an overflow:auto
                      // wrapper, so it never reaches this box as a min-content
                      // floor — state it here, or a long queries list squeezes
                      // the editor away instead of scrolling the form
                      minHeight: SCRIPT_FIELD_MIN_HEIGHT,
                      // the root centres its flex items, which would float a
                      // short script in the middle once it can outgrow content
                      alignItems: 'flex-start',
                    },
                    // the outlined root already signals focus; drop the
                    // browser's native focus ring on the editor's inner textarea
                    '& textarea': { outline: 'none' },
                  },
                  slotProps: {
                    // syntax-highlighted editor in place of the plain textarea
                    input: { inputComponent: ScriptEditorInput },
                    htmlInput: { 'data-testid': 'check-script' },
                  },
                }}
              />
            </Box>

            <CheckTestResults test={test} />

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
              <CheckTestControls test={test} onTest={handleTest} />
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
