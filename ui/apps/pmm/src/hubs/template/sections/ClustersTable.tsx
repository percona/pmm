import Box from '@mui/material/Box';
import ListItemText from '@mui/material/ListItemText';
import MenuItem from '@mui/material/MenuItem';
import { useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import { Table } from '@percona/peak-ui';
import { type FC, type MouseEvent, useMemo } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { PreviewChip } from 'components/preview-chip';
import { Messages } from '../hub.messages';
import type { HubDoor, HubTreeColumn, HubTreeNode } from '../hub.types';
import {
  clusterColumns,
  NARROW_HIDDEN,
  WIDE_VISIBLE,
} from './clustersTable.columns';
import { visuallyHidden } from './hub.sx';
import { useInlineSize } from './useInlineSize';

const EXPAND_BUTTON = 40;
const EXPAND_COLUMN = EXPAND_BUTTON + 16;
// Peak Table's row actions column, and the outlined frame's two borders.
const ACTIONS_COLUMN = 56;
const FRAME_BORDERS = 2;

// A link or button in a cell does its own thing; only a click on the row's plain area toggles it.
const keepClickInCell = (event: MouseEvent<HTMLElement>) => {
  if ((event.target as HTMLElement).closest('a, button')) {
    event.stopPropagation();
  }
};

const rowMenu = (doors: HubDoor[], closeMenu: () => void) =>
  doors.map((door) => (
    <MenuItem
      key={`${door.label}:${door.to}`}
      component={RouterLink}
      to={door.to}
      onClick={(event) => {
        event.stopPropagation();
        closeMenu();
      }}
    >
      <ListItemText>
        {door.label}
        {door.context && (
          <Box component="span" sx={visuallyHidden}>
            {` ${door.context}`}
          </Box>
        )}
      </ListItemText>
      {door.preview && <PreviewChip level={door.preview} />}
    </MenuItem>
  ));

export interface ClustersTableProps {
  /** Peak Table name, such as "hub-mongodb-clusters". */
  tableName: string;
  clusters: HubTreeNode[];
  treeColumns: HubTreeColumn[];
  expanded: Record<string, true>;
  /** The table's accessible name, such as "MongoDB clusters". */
  label: string;
}

/** The cluster tree: clusters, their shards or sets, and members, worst first. */
export const ClustersTable: FC<ClustersTableProps> = ({
  tableName,
  clusters,
  treeColumns,
  expanded,
  label,
}) => {
  const theme = useTheme();
  const narrowViewport = useMediaQuery(theme.breakpoints.down('md'));
  const columns = useMemo(() => clusterColumns(treeColumns), [treeColumns]);
  // The hub's own width decides, not the viewport's: the sidebar takes 257 px from md up.
  const fullWidth = useMemo(
    () =>
      columns.reduce((sum, column) => sum + (column.size ?? 0), 0) +
      EXPAND_COLUMN +
      ACTIONS_COLUMN +
      FRAME_BORDERS,
    [columns]
  );
  const [frameRef, width] = useInlineSize<HTMLDivElement>();
  const narrow = width === undefined ? narrowViewport : width < fullWidth;

  return (
    <Box ref={frameRef}>
      <Table<HubTreeNode>
        tableName={tableName}
        data={clusters}
        columns={columns}
        getRowId={(node) => node.id}
        getSubRows={(node) => node.children}
        enableExpanding
        // Every way in is a link or button; focusable cells would only add stops, and an expand-all on Enter.
        enableExpandAll={false}
        enableKeyboardShortcuts={false}
        enablePagination={false}
        // Rows arrive worst first; sorting would break the tree.
        enableSorting={false}
        enableColumnActions={false}
        enableColumnFilters={false}
        enableGlobalFilter={false}
        enableHiding={false}
        enableTopToolbar={false}
        enableBottomToolbar={false}
        enableRowActions
        positionActionsColumn="last"
        renderRowActionMenuItems={({ row, closeMenu }) =>
          rowMenu(row.original.doors.menu, closeMenu)
        }
        enableRowHoverAction
        rowHoverAction={(row) => {
          if (row.getCanExpand()) {
            row.toggleExpanded();
          }
        }}
        initialState={{ density: 'compact', expanded }}
        state={{ columnVisibility: narrow ? NARROW_HIDDEN : WIDE_VISIBLE }}
        displayColumnDefOptions={{
          'mrt-row-expand': {
            size: EXPAND_COLUMN,
            grow: false,
            Header: () => (
              <Box component="span" sx={visuallyHidden}>
                {Messages.clusters.expandColumn}
              </Box>
            ),
          },
        }}
        muiTableProps={{ 'aria-label': label }}
        muiTablePaperProps={{
          variant: 'outlined',
          sx: {
            borderRadius: `${theme.shape.borderRadiusMd}px`,
            overflow: 'hidden',
          },
        }}
        muiTableContainerProps={{ sx: { borderBottom: 'none' } }}
        muiTableBodyCellProps={{ onClick: keepClickInCell }}
        muiExpandButtonProps={({ row }) => ({
          'aria-label': (row.getIsExpanded()
            ? Messages.clusters.collapse
            : Messages.clusters.expand)(row.original.name),
          'aria-expanded': row.getCanExpand() ? row.getIsExpanded() : undefined,
          // The name cell indents each level, so the buttons stay in one column; a member has none.
          sx: {
            height: EXPAND_BUTTON,
            width: EXPAND_BUTTON,
            ml: 0,
            visibility: row.getCanExpand() ? 'visible' : 'hidden',
          },
        })}
      />
    </Box>
  );
};
