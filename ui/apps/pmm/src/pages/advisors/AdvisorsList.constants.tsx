import { type MRT_ColumnDef } from 'material-react-table';
import Select from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import Switch from '@mui/material/Switch';
import { AdvisorCheckRow, AdvisorInterval } from 'types/advisors.types';
import { ADVISOR_TECHNOLOGY, ADVISOR_INTERVAL } from 'lib/constants';
import { Messages } from './AdvisorsList.messages';

export const INTERVAL_OPTIONS = [
  AdvisorInterval.standard,
  AdvisorInterval.rare,
  AdvisorInterval.frequent,
];

interface AdvisorsColumnsProps {
  onToggleCheck: (check: AdvisorCheckRow) => void;
  onChangeInterval: (check: AdvisorCheckRow, interval: AdvisorInterval) => void;
}

export const getAdvisorsColumns = ({
  onToggleCheck,
  onChangeInterval,
}: AdvisorsColumnsProps): MRT_ColumnDef<AdvisorCheckRow>[] => [
  {
    header: Messages.columns.check,
    accessorKey: 'summary',
    size: 250,
    // the only growing column; all others stay at their fixed size
    grow: true,
  },
  {
    id: 'category',
    header: Messages.columns.category,
    accessorFn: (row) => row.category,
    size: 150,
    grow: false,
  },
  {
    id: 'technology',
    header: Messages.columns.technology,
    accessorFn: (row) => ADVISOR_TECHNOLOGY[row.technology],
    size: 140,
    grow: false,
  },
  {
    id: 'source',
    header: Messages.columns.source,
    accessorFn: (row) =>
      row.userDefined ? Messages.source.custom : Messages.source.builtin,
    size: 110,
    grow: false,
  },
  {
    id: 'interval',
    header: Messages.columns.interval,
    accessorFn: (row) => ADVISOR_INTERVAL[row.interval],
    size: 130,
    grow: false,
    Cell: ({ row }) => (
      <Select
        size="small"
        variant="standard"
        value={
          row.original.interval === AdvisorInterval.unspecified
            ? AdvisorInterval.standard
            : row.original.interval
        }
        onChange={(e) =>
          onChangeInterval(row.original, e.target.value as AdvisorInterval)
        }
        data-testid={`check-${row.original.checkName}-interval-select`}
      >
        {INTERVAL_OPTIONS.map((interval) => (
          <MenuItem key={interval} value={interval}>
            {ADVISOR_INTERVAL[interval]}
          </MenuItem>
        ))}
      </Select>
    ),
  },
  {
    id: 'status',
    header: Messages.columns.status,
    accessorFn: (row) =>
      row.enabled ? Messages.status.enabled : Messages.status.disabled,
    size: 110,
    grow: false,
    Cell: ({ row }) => (
      <Switch
        size="small"
        checked={row.original.enabled}
        onChange={() => onToggleCheck(row.original)}
        data-testid={`check-${row.original.checkName}-status-switch`}
      />
    ),
  },
];
