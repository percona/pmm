import type { Meta, StoryObj } from '@storybook/react';
import { FormProvider, useForm, useWatch } from 'react-hook-form';
import { Box, Button, Stack, Typography } from '@mui/material';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFnsV3';
import DateTimePickerInput from './date-time-picker';

const meta: Meta<typeof DateTimePickerInput> = {
  title: 'Inputs/DateTimePickerInput',
  component: DateTimePickerInput,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'A react-hook-form-bound wrapper around MUI X `<DateTimePicker>`. By default the value stored in form state is a `Date` object; pass `valueFormat="iso-string"` or `"unix-ms"` (or a custom `transform`) to bridge other shapes without a custom outer `Controller`.',
      },
    },
  },
  decorators: [
    (Story) => (
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <Box sx={{ minWidth: 360 }}>
          <Story />
        </Box>
      </LocalizationProvider>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof DateTimePickerInput>;

// ---------- helpers ----------

interface FormShellProps<TValues extends Record<string, unknown>> {
  defaultValues: TValues;
  fieldName: keyof TValues & string;
  children: React.ReactNode;
  /** Render function that pretty-prints the live form value for the story panel. */
  formatValue?: (value: unknown) => string;
}

function FormShell<TValues extends Record<string, unknown>>({
  defaultValues,
  fieldName,
  children,
  formatValue,
}: FormShellProps<TValues>) {
  const methods = useForm<TValues>({ defaultValues, mode: 'onChange' });

  return (
    <FormProvider {...methods}>
      <Stack
        component="form"
        spacing={2}
        onSubmit={methods.handleSubmit((values) => {
          // eslint-disable-next-line no-console
          console.log('submit', values);
        })}
      >
        {children}
        <LiveValue name={fieldName} formatValue={formatValue} />
        <Button type="submit" variant="contained" size="small" sx={{ alignSelf: 'flex-start' }}>
          Submit (see console)
        </Button>
      </Stack>
    </FormProvider>
  );
}

function LiveValue({
  name,
  formatValue,
}: {
  name: string;
  formatValue?: (value: unknown) => string;
}) {
  const value = useWatch({ name });
  const rendered =
    formatValue?.(value) ??
    (value === undefined || value === null
      ? String(value)
      : value instanceof Date
        ? value.toISOString()
        : JSON.stringify(value));

  return (
    <Box
      sx={{
        fontFamily: 'monospace',
        fontSize: 12,
        bgcolor: 'action.hover',
        p: 1,
        borderRadius: 1,
      }}
    >
      <Typography variant="caption" display="block" color="text.secondary">
        form state ({name}):
      </Typography>
      <code>{rendered}</code>
    </Box>
  );
}

// ---------- stories ----------

/**
 * Default `valueFormat='date'` — form state holds a `Date` object.
 * This is the legacy behavior; existing consumers see no change.
 */
export const Default: Story = {
  render: () => (
    <FormShell defaultValues={{ scheduledFor: new Date() }} fieldName="scheduledFor">
      <DateTimePickerInput name="scheduledFor" label="Scheduled for" />
    </FormShell>
  ),
};

/**
 * `valueFormat='iso-string'` — form state holds an ISO 8601 string.
 * Useful when the wire format for the backend is a string and you don't want
 * to juggle `Date` objects in your form values.
 */
export const IsoString: Story = {
  name: 'valueFormat="iso-string"',
  render: () => (
    <FormShell
      defaultValues={{ scheduledFor: '2025-01-01T12:00:00.000Z' }}
      fieldName="scheduledFor"
    >
      <DateTimePickerInput
        name="scheduledFor"
        label="Scheduled for (ISO)"
        valueFormat="iso-string"
      />
    </FormShell>
  ),
};

/**
 * `valueFormat='unix-ms'` — form state holds epoch milliseconds.
 */
export const UnixMs: Story = {
  name: 'valueFormat="unix-ms"',
  render: () => (
    <FormShell defaultValues={{ scheduledFor: Date.now() }} fieldName="scheduledFor">
      <DateTimePickerInput
        name="scheduledFor"
        label="Scheduled for (epoch ms)"
        valueFormat="unix-ms"
      />
    </FormShell>
  ),
};

/**
 * Custom `transform` — takes precedence over `valueFormat`.
 * Here the form state is a `{ iso: string }` object, demonstrating that the
 * transform can shape values into any structure a form needs.
 */
export const CustomTransform: Story = {
  name: 'custom transform',
  render: () => (
    <FormShell
      defaultValues={{ scheduledFor: { iso: '2025-06-15T09:30:00.000Z' } }}
      fieldName="scheduledFor"
      formatValue={(v) => JSON.stringify(v)}
    >
      <DateTimePickerInput
        name="scheduledFor"
        label="Scheduled for (wrapped object)"
        transform={{
          input: (v) => {
            if (v && typeof v === 'object' && 'iso' in v && typeof v.iso === 'string') {
              return new Date(v.iso);
            }
            return null;
          },
          output: (v) =>
            v instanceof Date && !Number.isNaN(v.getTime()) ? { iso: v.toISOString() } : null,
        }}
      />
    </FormShell>
  ),
};

/**
 * Empty default value — picker starts blank, `onChange` emits the configured
 * empty shape (`null` for `'iso-string'` / `'unix-ms'`, `null` for `'date'`).
 */
export const EmptyInitial: Story = {
  name: 'empty initial value (iso-string)',
  render: () => (
    <FormShell defaultValues={{ scheduledFor: null }} fieldName="scheduledFor">
      <DateTimePickerInput name="scheduledFor" label="Scheduled for" valueFormat="iso-string" />
    </FormShell>
  ),
};
