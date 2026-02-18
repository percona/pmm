import { type MRT_ColumnDef } from 'material-react-table';

import { QueryData } from 'types/rta.types';
import { Messages } from './OverviewTable.messages';
import { QueryCell } from './query-cell';
import { formatDuration } from 'date-fns';
import UnavailableText from 'components/unavailable-text';

export const OVERVIEW_TABLE_COLUMNS: MRT_ColumnDef<QueryData>[] = [
  {
    minSize: 400,
    header: Messages.columns.queryText,
    accessorKey: 'queryText',
    Cell: ({ row }) => <QueryCell query={row.original.queryText} />,
  },
  {
    header: Messages.columns.host,
    accessorKey: 'serviceName',
  },
  {
    header: Messages.columns.operationId,
    accessorKey: 'queryId',
  },
  {
    size: 150,
    header: Messages.columns.elapsedTime,
    accessorKey: 'queryExecutionDurationMs',
    filterVariant: 'range',
    filterFn: 'timeRangeFilterFn',
    Cell: ({ cell }) => cell.getValue() ? `${formatDuration({
      seconds: cell.getValue<number>(),
    }, {
      format: ['seconds'],
    })}` : <UnavailableText />,
  },
];
