import { type MRT_ColumnDef } from 'material-react-table';
import Switch from '@mui/material/Switch';
import type { AdvisorCheckRow } from 'types/advisors.types';
import { ADVISOR_TECHNOLOGY } from 'lib/constants';
import { Messages } from './AdvisorsList.messages';

// query-string params remembered across visits
export const CHECK_FILTER_PARAMS = [
  'search',
  'category',
  'technology',
  'status',
  'pageSize',
];

interface AdvisorsColumnsProps {
  onToggleCheck: (check: AdvisorCheckRow) => void;
}

export const getAdvisorsColumns = ({
  onToggleCheck,
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
