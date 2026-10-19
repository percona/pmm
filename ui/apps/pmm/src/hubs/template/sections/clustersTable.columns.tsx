import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { Tooltip } from '@percona/peak-ui';
import type { MRT_ColumnDef, MRT_VisibilityState } from 'material-react-table';
import { Messages } from '../hub.messages';
import type { HubTreeColumn, HubTreeNode } from '../hub.types';
import { DoorLink } from './DoorLink';
import { HealthChip } from './HealthChip';
import { MeasureValue } from './MeasureValue';
import { focusableText } from './hub.sx';

/** Columns that fold into the name cell's caption line below md. */
export const NARROW_HIDDEN: MRT_VisibilityState = {
  type: false,
  environment: false,
  members: false,
};

export const WIDE_VISIBLE: MRT_VisibilityState = {
  type: true,
  environment: true,
  members: true,
};

const membersText = (members: NonNullable<HubTreeNode['members']>) =>
  `${members.up}/${members.total}`;

export const clusterColumns = (
  treeColumns: HubTreeColumn[]
): MRT_ColumnDef<HubTreeNode>[] => [
  {
    id: 'name',
    header: Messages.clusters.columns.name,
    accessorKey: 'name',
    size: 185,
    Cell: ({ row, table }) => {
      const node = row.original;
      const folded = table.getState().columnVisibility.type === false;
      const caption = [
        node.type,
        row.depth === 0 ? node.environment : undefined,
        node.members ? membersText(node.members) : undefined,
      ]
        .filter(Boolean)
        .join(' · ');

      return (
        <Box sx={{ pl: row.depth * 2, minWidth: 0 }}>
          <DoorLink door={node.doors.name} color="inherit">
            {node.name}
          </DoorLink>
          {folded && caption && (
            <Typography
              variant="caption"
              color="text.secondary"
              component="div"
            >
              {caption}
            </Typography>
          )}
        </Box>
      );
    },
  },
  {
    id: 'type',
    header: Messages.clusters.columns.type,
    accessorKey: 'type',
    size: 130,
  },
  {
    id: 'environment',
    header: Messages.clusters.columns.environment,
    accessorFn: (node) => node.environment ?? '',
    size: 115,
    Cell: ({ row }) => (row.depth === 0 ? row.original.environment : null),
  },
  {
    id: 'health',
    header: Messages.clusters.columns.health,
    accessorKey: 'health',
    size: 134,
    Cell: ({ row }) => {
      const node = row.original;

      return node.showHealth ? (
        <HealthChip
          health={node.health}
          since={node.since}
          detail={node.healthDetail}
        />
      ) : null;
    },
  },
  {
    id: 'members',
    header: Messages.clusters.columns.members,
    accessorFn: (node) => node.members?.up ?? null,
    size: 122,
    Cell: ({ row }) => {
      const { members, doors } = row.original;

      if (!members) {
        return null;
      }

      return (
        <span>
          {doors.members ? (
            <DoorLink door={doors.members} color="inherit">
              {membersText(members)}
            </DoorLink>
          ) : (
            membersText(members)
          )}
          {members.noData > 0 &&
            Messages.clusters.membersNoData(members.noData)}
        </span>
      );
    },
  },
  ...treeColumns.map<MRT_ColumnDef<HubTreeNode>>((column) => ({
    id: column.id,
    header: column.header,
    accessorFn: (node) => node.measures[column.id]?.kind ?? null,
    size: 130,
    Header: () => (
      <Tooltip title={column.description} describeChild>
        <Box component="span" tabIndex={0} sx={focusableText}>
          {column.header}
        </Box>
      </Tooltip>
    ),
    Cell: ({ row }) => (
      <MeasureValue measure={row.original.measures[column.id]} />
    ),
  })),
];
