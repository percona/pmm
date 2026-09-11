import { TextInput } from '@percona/peak-ui';
import type { MRT_ColumnDef } from '@percona/peak-ui';
import type { AlertThresholdRow } from './AlertThresholds.types';
import ResetValueCell from './reset-value-cell';
import { Messages } from './AlertThresholds.messages';
import { formatUnit } from './AlertThresholds.utils';

// Maps the backend ParamUnit enum to a display symbol.
export const UNIT_SYMBOLS: Record<string, string> = {
  PARAM_UNIT_PERCENTAGE: '%',
  PARAM_UNIT_SECONDS: 's',
};

export const ALERT_THRESHOLDS_COLUMNS: MRT_ColumnDef<AlertThresholdRow>[] = [
  {
    accessorKey: 'ruleTitle',
    header: Messages.table.columns.ruleTitle,
  },
  {
    accessorKey: 'summary',
    header: Messages.table.columns.parameter,
    Cell: ({ row: { original } }) => original.paramName,
  },
  {
    accessorKey: 'defaultValue',
    header: Messages.table.columns.default,
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
    header: Messages.table.columns.override,
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
    header: Messages.table.columns.unit,
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
