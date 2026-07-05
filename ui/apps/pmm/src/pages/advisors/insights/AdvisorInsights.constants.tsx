import { type MRT_ColumnDef } from 'material-react-table';
import Chip from '@mui/material/Chip';
import Link from '@mui/material/Link';
import { format } from 'date-fns';
import {
  AdvisorCheckResultStatus,
  CheckResultHistoryItem,
} from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import {
  ADVISOR_RESULT_STATUS,
  SEVERITY,
  TIME_FORMAT,
} from 'lib/constants';
import { capitalize } from 'utils/text.utils';
import { Messages } from './AdvisorInsights.messages';

const SEVERITY_OPTIONS = [
  Severity.emergency,
  Severity.alert,
  Severity.critical,
  Severity.error,
  Severity.warning,
  Severity.notice,
  Severity.info,
  Severity.debug,
].map((severity) => ({ label: SEVERITY[severity], value: severity }));

const STATUS_OPTIONS = [
  AdvisorCheckResultStatus.ok,
  AdvisorCheckResultStatus.failed,
  AdvisorCheckResultStatus.error,
].map((status) => ({ label: ADVISOR_RESULT_STATUS[status], value: status }));

const READ_OPTIONS = [
  { label: Messages.read, value: 'true' },
  { label: Messages.unread, value: 'false' },
];

export const getInsightsColumns = (
  categories: string[]
): MRT_ColumnDef<CheckResultHistoryItem>[] => [
  {
    header: Messages.columns.summary,
    accessorKey: 'summary',
    enableColumnFilter: false,
    size: 300,
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
    filterVariant: 'select',
    filterSelectOptions: categories.map((category) => ({
      label: capitalize(category),
      value: category,
    })),
    size: 140,
  },
  {
    id: 'severity',
    header: Messages.columns.severity,
    accessorFn: (row) => SEVERITY[row.severity],
    filterVariant: 'select',
    filterSelectOptions: SEVERITY_OPTIONS,
    size: 130,
  },
  {
    id: 'status',
    header: Messages.columns.status,
    accessorFn: (row) => ADVISOR_RESULT_STATUS[row.status],
    filterVariant: 'select',
    filterSelectOptions: STATUS_OPTIONS,
    size: 120,
  },
  {
    header: Messages.columns.checkedAt,
    accessorKey: 'checkedAt',
    enableColumnFilter: false,
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
    filterVariant: 'select',
    filterSelectOptions: READ_OPTIONS,
    size: 110,
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
