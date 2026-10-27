import ArrowBack from '@mui/icons-material/ArrowBack';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import RadioButtonUncheckedIcon from '@mui/icons-material/RadioButtonUnchecked';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Divider from '@mui/material/Divider';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import {
  AutoCompleteInput,
  DateTimePickerInput,
  RadioGroup,
  SwitchInput,
  TextInput,
  ToggleButtonGroupInputRegular,
  ToggleRegularButton,
} from '@percona/peak-ui';
import { format, formatDistanceStrict, subHours } from 'date-fns';
import {
  type FC,
  type PropsWithChildren,
  type ReactNode,
  useEffect,
} from 'react';
import { FormProvider, useForm, useWatch } from 'react-hook-form';
import { Page } from 'components/page';
import { MOCK_SERVICES } from './mockups.data';

type Scope = 'all' | 'selected';
type Preset = '1' | '6' | '12' | '24' | 'custom';

interface FormValues {
  scope: Scope;
  serviceNames: string[];
  preset: Preset;
  startTime: Date;
  endTime: Date;
  exportQan: boolean;
  ignoreLoad: boolean;
  enableEncryption: boolean;
  encryptionPassword: string;
  confirmPassword: string;
}

interface CreateDatasetMockupProps {
  variant?: 'default' | 'filled';
  onCancel: () => void;
}

const PRESETS: { value: Preset; label: string }[] = [
  { value: '1', label: 'Last hour' },
  { value: '6', label: 'Last 6 hours' },
  { value: '12', label: 'Last 12 hours' },
  { value: '24', label: 'Last 24 hours' },
  { value: 'custom', label: 'Custom' },
];

const PASSWORD_RULES: { label: string; test: (value: string) => boolean }[] = [
  { label: 'At least 8 characters', test: (value) => value.length >= 8 },
  { label: 'A letter', test: (value) => /[A-Za-z]/.test(value) },
  { label: 'A number', test: (value) => /\d/.test(value) },
  { label: 'A special character', test: (value) => /[^A-Za-z0-9]/.test(value) },
];

const getDefaults = (
  variant: CreateDatasetMockupProps['variant']
): FormValues => {
  const endTime = new Date();
  endTime.setSeconds(0, 0);
  const filled = variant === 'filled';

  return {
    scope: filled ? 'selected' : 'all',
    serviceNames: filled ? ['mysql-prod-primary', 'mysql-prod-replica-1'] : [],
    preset: '12',
    startTime: subHours(endTime, 12),
    endTime,
    exportQan: filled,
    ignoreLoad: false,
    enableEncryption: filled,
    encryptionPassword: filled ? 'support2026' : '',
    confirmPassword: '',
  };
};

const Section: FC<
  PropsWithChildren<{ title: string; description?: ReactNode }>
> = ({ title, description, children }) => (
  <Stack gap={2}>
    <Stack gap={0.5}>
      <Typography variant="h6" component="h2">
        {title}
      </Typography>
      {description && (
        <Typography variant="body2" color="text.secondary">
          {description}
        </Typography>
      )}
    </Stack>
    {children}
  </Stack>
);

const SummaryRow: FC<{ label: string; value: ReactNode }> = ({
  label,
  value,
}) => (
  <Stack direction="row" justifyContent="space-between" gap={2}>
    <Typography variant="body2" color="text.secondary">
      {label}
    </Typography>
    <Typography variant="body2" textAlign="right">
      {value}
    </Typography>
  </Stack>
);

export const CreateDatasetMockup: FC<CreateDatasetMockupProps> = ({
  variant = 'default',
  onCancel,
}) => {
  const methods = useForm<FormValues>({ defaultValues: getDefaults(variant) });
  const { control, setValue } = methods;
  const values = useWatch({ control });
  const password = values.encryptionPassword ?? '';

  useEffect(() => {
    if (values.preset && values.preset !== 'custom') {
      const endTime = new Date();
      endTime.setSeconds(0, 0);
      setValue('endTime', endTime);
      setValue('startTime', subHours(endTime, Number(values.preset)));
    }
  }, [values.preset, setValue]);

  const serviceSummary =
    values.scope === 'all'
      ? `All services (${MOCK_SERVICES.length})`
      : `${values.serviceNames?.length ?? 0} selected`;

  return (
    <Page
      title="Create dataset"
      footer={null}
      maxWidth={1200}
      topBar={
        <Link
          component="button"
          onClick={onCancel}
          underline="hover"
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 0.5,
            alignSelf: 'flex-start',
          }}
        >
          <ArrowBack fontSize="small" />
          PMM Dump
        </Link>
      }
    >
      <FormProvider {...methods}>
        <Stack
          direction={{ xs: 'column', lg: 'row' }}
          gap={3}
          alignItems="flex-start"
          component="form"
        >
          <Card variant="outlined" sx={{ flex: 1, width: '100%' }}>
            <CardContent>
              <Stack gap={4} divider={<Divider flexItem />}>
                <Section
                  title="Services"
                  description="Pick the services involved in the issue. Fewer services means a smaller archive and a faster export."
                >
                  <RadioGroup
                    name="scope"
                    options={[
                      {
                        label: `All services (${MOCK_SERVICES.length})`,
                        value: 'all',
                      },
                      { label: 'Only selected services', value: 'selected' },
                    ]}
                    radioGroupFieldProps={{ row: true }}
                  />
                  {values.scope === 'selected' && (
                    <AutoCompleteInput
                      name="serviceNames"
                      label="Services"
                      options={MOCK_SERVICES}
                      autoCompleteProps={{
                        multiple: true,
                        disableCloseOnSelect: true,
                      }}
                      textFieldProps={{ placeholder: 'Search services' }}
                    />
                  )}
                </Section>

                <Section
                  title="Time range"
                  description="Cover the time the problem happened, plus some time before it for comparison."
                >
                  <ToggleButtonGroupInputRegular
                    name="preset"
                    toggleButtonGroupProps={{ exclusive: true, size: 'small' }}
                  >
                    {PRESETS.map(({ value, label }) => (
                      <ToggleRegularButton key={value} value={value}>
                        {label}
                      </ToggleRegularButton>
                    ))}
                  </ToggleButtonGroupInputRegular>
                  <Stack direction={{ xs: 'column', sm: 'row' }} gap={2}>
                    <DateTimePickerInput
                      name="startTime"
                      label="From"
                      disabled={values.preset !== 'custom'}
                      maxDateTime={new Date()}
                      sx={{ flex: 1 }}
                    />
                    <DateTimePickerInput
                      name="endTime"
                      label="To"
                      disabled={values.preset !== 'custom'}
                      maxDateTime={new Date()}
                      sx={{ flex: 1 }}
                    />
                  </Stack>
                </Section>

                <Section
                  title="Data to include"
                  description="Metrics for the selected services are always included."
                >
                  <SwitchInput
                    name="exportQan"
                    label="Include Query Analytics (QAN)"
                    labelCaption="Adds query statistics and examples. Examples can contain data from your queries."
                  />
                </Section>

                <Section title="Security">
                  <SwitchInput
                    name="enableEncryption"
                    label="Encrypt the archive"
                    labelCaption="Only someone with the password can open it."
                  />
                  {values.enableEncryption && (
                    <Stack gap={2}>
                      <Stack direction={{ xs: 'column', sm: 'row' }} gap={2}>
                        <TextInput
                          name="encryptionPassword"
                          label="Password"
                          textFieldProps={{ type: 'password', fullWidth: true }}
                        />
                        <TextInput
                          name="confirmPassword"
                          label="Confirm password"
                          textFieldProps={{ type: 'password', fullWidth: true }}
                        />
                      </Stack>
                      <Stack
                        direction="row"
                        flexWrap="wrap"
                        columnGap={3}
                        rowGap={0.5}
                      >
                        {PASSWORD_RULES.map(({ label, test }) => {
                          const met = test(password);
                          return (
                            <Stack
                              key={label}
                              direction="row"
                              gap={0.5}
                              alignItems="center"
                            >
                              {met ? (
                                <CheckCircleOutlineIcon
                                  sx={{ fontSize: 16 }}
                                  color="success"
                                />
                              ) : (
                                <RadioButtonUncheckedIcon
                                  sx={{ fontSize: 16 }}
                                  color="disabled"
                                />
                              )}
                              <Typography
                                variant="caption"
                                color={met ? 'text.primary' : 'text.secondary'}
                              >
                                {label}
                              </Typography>
                            </Stack>
                          );
                        })}
                      </Stack>
                      <Alert severity="warning">
                        PMM doesn’t store this password and can’t recover it.
                        Share it with Percona Support in your ticket, separately
                        from the archive.
                      </Alert>
                    </Stack>
                  )}
                </Section>

                <Section title="Server load">
                  <SwitchInput
                    name="ignoreLoad"
                    label="Export even when PMM Server is busy"
                    labelCaption="By default the export pauses while PMM Server is under heavy load. Turn this on to finish faster, at the cost of slower dashboards while it runs."
                  />
                </Section>
              </Stack>
            </CardContent>
          </Card>

          <Card
            variant="outlined"
            sx={{
              width: { xs: '100%', lg: 340 },
              position: { lg: 'sticky' },
              top: 16,
            }}
          >
            <CardContent>
              <Stack gap={2}>
                <Typography variant="h6" component="h2">
                  Summary
                </Typography>
                <Stack gap={1}>
                  <SummaryRow label="Services" value={serviceSummary} />
                  {values.startTime && values.endTime && (
                    <>
                      <SummaryRow
                        label="From"
                        value={format(values.startTime, 'MMM d, HH:mm')}
                      />
                      <SummaryRow
                        label="To"
                        value={format(values.endTime, 'MMM d, HH:mm')}
                      />
                      <SummaryRow
                        label="Duration"
                        value={formatDistanceStrict(
                          values.endTime,
                          values.startTime
                        )}
                      />
                    </>
                  )}
                  <SummaryRow
                    label="Includes"
                    value={values.exportQan ? 'Metrics, QAN' : 'Metrics'}
                  />
                  <SummaryRow
                    label="Encryption"
                    value={values.enableEncryption ? 'On' : 'Off'}
                  />
                </Stack>
                <Typography variant="caption" color="text.secondary">
                  The export runs in the background. You can leave this page and
                  follow its progress in the dataset list.
                </Typography>
                <Box display="flex" flexDirection="column" gap={1}>
                  <Button variant="contained" fullWidth onClick={onCancel}>
                    Create dataset
                  </Button>
                  <Button fullWidth onClick={onCancel}>
                    Cancel
                  </Button>
                </Box>
              </Stack>
            </CardContent>
          </Card>
        </Stack>
      </FormProvider>
    </Page>
  );
};
