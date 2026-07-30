import FileOpenOutlinedIcon from '@mui/icons-material/FileOpenOutlined';
import Box from '@mui/material/Box';
import Tooltip from '@mui/material/Tooltip';
import { format } from 'date-fns';
import { ADVISOR_RESULT_STATUS, SEVERITY, TIME_FORMAT } from 'lib/constants';
import { type MRT_ColumnDef } from 'material-react-table';
import { Insight } from 'types/advisors.types';
import { Severity } from 'types/severity.types';
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

export const getInsightsColumns = (): MRT_ColumnDef<Insight>[] => [
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
            {/* indicator only — the link itself lives in the details pane, so
                the icon must not compete with the row's own click targets */}
            <Tooltip title={Messages.readMoreAvailable} arrow>
              <FileOpenOutlinedIcon
                fontSize="inherit"
                color="action"
                sx={{ verticalAlign: 'text-bottom' }}
                data-testid="insight-read-more-indicator"
              />
            </Tooltip>
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
    grow: true,
  },
  {
    id: 'category',
    header: Messages.columns.category,
    accessorFn: (row) => row.category,
    size: 140,
    grow: false,
  },
  {
    id: 'severity',
    header: Messages.columns.severity,
    accessorFn: (row) => SEVERITY[row.severity],
    sortingFn: (rowA, rowB) =>
      SEVERITY_ORDER[rowA.original.severity] -
      SEVERITY_ORDER[rowB.original.severity],
    size: 130,
    grow: false,
  },
  {
    id: 'status',
    header: Messages.columns.status,
    accessorFn: (row) => ADVISOR_RESULT_STATUS[row.status],
    size: 120,
    grow: false,
  },
  {
    header: Messages.columns.checkedAt,
    accessorKey: 'checkedAt',
    size: 160,
    grow: false,
    Cell: ({ row }) =>
      row.original.checkedAt ? (
        <Box component="span" sx={{ fontSize: '0.85rem' }}>
          {format(new Date(row.original.checkedAt), TIME_FORMAT)}
        </Box>
      ) : null,
  },
];
