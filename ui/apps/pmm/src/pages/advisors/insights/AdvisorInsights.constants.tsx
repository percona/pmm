import { type MRT_ColumnDef } from 'material-react-table';
import Chip from '@mui/material/Chip';
import Link from '@mui/material/Link';
import { format } from 'date-fns';
import { CheckResultHistoryItem } from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import {
  ADVISOR_RESULT_STATUS,
  SEVERITY,
  TIME_FORMAT,
} from 'lib/constants';
import { capitalize } from 'utils/text.utils';
import { Messages } from './AdvisorInsights.messages';

const SEVERITY_ORDER: Record<Severity, number> = {
  [Severity.emergency]: 1,
  [Severity.alert]: 2,
  [Severity.critical]: 3,
  [Severity.error]: 4,
  [Severity.warning]: 5,
  [Severity.notice]: 6,
  [Severity.info]: 7,
  [Severity.debug]: 8,
  [Severity.unspecified]: 9,
};

export const getInsightsColumns = (): MRT_ColumnDef<CheckResultHistoryItem>[] => [
  {
    header: Messages.columns.summary,
    accessorKey: 'summary',
    enableSorting: false,
    size: 300,
    grow: true,
    Cell: ({ row }) => (
      <span>
        {row.original.summary}
        {row.original.readMoreUrl && (
          <>
            {' '}
            <Link
              href={row.original.readMoreUrl}
              target="_blank"
              rel="noopener noreferrer"
            >
              {Messages.readMore}
            </Link>
          </>
        )}
      </span>
    ),
  },
  {
    id: 'serviceName',
    header: Messages.columns.service,
    accessorKey: 'serviceName',
    size: 160,
  },
  {
    id: 'category',
    header: Messages.columns.category,
    accessorFn: (row) => capitalize(row.category),
    size: 140,
  },
  {
    id: 'severity',
    header: Messages.columns.severity,
    accessorFn: (row) => SEVERITY[row.severity],
    sortingFn: (rowA, rowB) =>
      SEVERITY_ORDER[rowA.original.severity] -
      SEVERITY_ORDER[rowB.original.severity],
    size: 130,
  },
  {
    id: 'status',
    header: Messages.columns.status,
    accessorFn: (row) => ADVISOR_RESULT_STATUS[row.status],
    size: 120,
  },
  {
    header: Messages.columns.checkedAt,
    accessorKey: 'checkedAt',
    size: 160,
    Cell: ({ row }) =>
      row.original.checkedAt
        ? format(new Date(row.original.checkedAt), TIME_FORMAT)
        : null,
  },
  {
    id: 'isRead',
    header: Messages.columns.read,
    accessorFn: (row) => (row.isRead ? Messages.read : Messages.unread),
    size: 90,
    grow: false,
    Cell: ({ row }) => (
      <Chip
        size="small"
        color={row.original.isRead ? 'default' : 'info'}
        label={row.original.isRead ? Messages.read : Messages.unread}
        data-testid={`insight-${row.original.id}-read-state`}
      />
    ),
  },
];
