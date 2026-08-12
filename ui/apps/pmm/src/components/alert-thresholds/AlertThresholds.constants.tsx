import { MRT_ColumnDef, TextInput } from '@percona/peak-ui';
import { AlertThresholdRow } from './AlertThresholds.types';
import ResetValueCell from './reset-value-cell';

// Maps the backend ParamUnit enum to a display symbol.
export const UNIT_SYMBOLS: Record<string, string> = {
  PARAM_UNIT_PERCENTAGE: '%',
  PARAM_UNIT_SECONDS: 's',
};

export const formatUnit = (unit?: string): string =>
  (unit && UNIT_SYMBOLS[unit]) || '';

export const ALERT_THRESHOLDS_COLUMNS: MRT_ColumnDef<AlertThresholdRow>[] = [
  {
    accessorKey: 'ruleTitle',
    header: 'Alert rule',
  },
  {
    accessorKey: 'summary',
    header: 'Parameter',
    Cell: ({ row: { original } }) => original.paramName,
  },
  {
    accessorKey: 'defaultValue',
    header: 'Default',
    enableColumnActions: false,
    enableColumnFilter: false,
    enableSorting: false,
    muiTableHeadCellProps: {
      sx: {
        '.Mui-TableHeadCell-Content': {
          height: 40,
        },
      },
    },
  },
  {
    accessorKey: 'effectiveValue',
    header: 'Override',
    enableColumnActions: false,
    enableColumnFilter: false,
    enableSorting: false,
    muiTableHeadCellProps: {
      sx: {
        '.Mui-TableHeadCell-Content': {
          height: 40,
        },
      },
    },
    // Every row returned by the endpoint is overridable; the field is
    // pre-filled with the effective value via react-hook-form defaults.
    Cell: ({ row: { original } }) => (
      <TextInput
        name={original.id}
        textFieldProps={{
          type: 'number',
          sx: { m: 0 },
        }}
      />
    ),
  },
  {
    id: 'unit',
    size: 80,
    grow: false,
    header: 'Unit',
    enableColumnActions: false,
    muiTableHeadCellProps: {
      sx: {
        '.Mui-TableHeadCell-Content': {
          height: 40,
        },
      },
    },
    Cell: ({ row: { original } }) => formatUnit(original.unit),
  },
  {
    id: 'reset',
    size: 80,
    header: '',
    enableColumnActions: false,
    Cell: ({ row: { original } }) => <ResetValueCell row={original} />,
  },
];
