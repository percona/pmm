import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import { Table } from '@percona/peak-ui';
import type { MRT_ColumnDef } from 'material-react-table';
import { useMemo } from 'react';
import { WRAPPING_HEADER_SX } from './DiscoveredInstances.constants';
import { Messages } from './DiscoveredInstances.messages';
import type {
  DiscoveredInstance,
  DiscoveredInstancesProps,
} from './DiscoveredInstances.types';

export const DiscoveredInstances = <T extends DiscoveredInstance>({
  tableName,
  columns,
  instances,
  isLoading,
  onSelect,
}: DiscoveredInstancesProps<T>) => {
  const columnsWithAction = useMemo<MRT_ColumnDef<T>[]>(
    () => [
      ...columns.map((column) => ({
        ...column,
        muiTableHeadCellProps: { sx: WRAPPING_HEADER_SX },
      })),
      {
        id: 'action',
        header: Messages.action,
        // Fits the button and takes none of the spare width.
        size: 170,
        grow: false,
        Cell: ({ row }) => (
          <Button
            size="small"
            variant="outlined"
            data-testid={`start-monitoring-${row.original.instanceId}`}
            onClick={() => onSelect(row.original)}
          >
            {Messages.startMonitoring}
          </Button>
        ),
      },
    ],
    [columns, onSelect]
  );

  return (
    <Box data-testid="discovered-instances">
      <Table
        tableName={tableName}
        noDataMessage={Messages.noData}
        columns={columnsWithAction}
        data={instances}
        state={{ isLoading }}
        enableFilters={false}
        enableColumnActions={false}
        enableColumnOrdering={false}
        enableTopToolbar={false}
        enableColumnDragging={false}
      />
    </Box>
  );
};
