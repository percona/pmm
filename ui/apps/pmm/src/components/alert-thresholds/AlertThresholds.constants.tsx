import { MRT_ColumnDef, TextInput } from '@percona/percona-ui';
import { AlertThresholdRow } from './AlertThresholds.types';

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
    Cell: ({ row: { original } }) => original.summary || original.paramName,
  },
  {
    accessorKey: 'defaultValue',
    header: 'Default',
  },
  {
    accessorKey: 'effectiveValue',
    header: 'Override',
    // Every row returned by the endpoint is overridable; the field is
    // pre-filled with the effective value via react-hook-form defaults.
    Cell: ({ row: { original } }) => <TextInput name={original.id} />,
  },
  {
    id: 'unit',
    header: 'Unit',
    Cell: ({ row: { original } }) => formatUnit(original.unit),
  },
];
