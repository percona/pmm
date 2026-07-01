import { MRT_ColumnDef, TextInput } from '@percona/percona-ui';
import { AlertThresholdRow } from './AlertThresholds.types';
import Typography from '@mui/material/Typography';

export const ALERT_THRESHOLDS_COLUMNS: MRT_ColumnDef<AlertThresholdRow>[] = [
  {
    accessorKey: 'alertRuleName',
    header: 'Alert rule',
  },
  {
    accessorKey: 'defaultThreshold',
    header: 'Default',
  },
  {
    accessorKey: 'overrideThreshold',
    header: 'Override',
    Cell: ({ row: { original } }) =>
      original.supportsOverride ? (
        <TextInput name={original.ruleUid} />
      ) : (
        <Typography variant="body2" color="text.secondary">
          Unsupported
        </Typography>
      ),
  },
  {
    accessorKey: 'unit',
    header: 'Unit',
  },
];
