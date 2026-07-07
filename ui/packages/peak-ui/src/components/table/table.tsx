import {
  Close as CloseIcon,
  FilterList as FilterListIcon,
  KeyboardDoubleArrowDown as KeyboardDoubleArrowDownIcon,
  MoreVert as MoreVertIcon,
  Search as SearchIcon,
  ViewColumn as ViewColumnIcon,
} from '@mui/icons-material';
import { Alert, AlertProps, Box, IconButton, InputAdornment, Theme, Tooltip } from '@mui/material';
import visuallyHidden from '@mui/utils/visuallyHidden';
import {
  MaterialReactTable,
  MRT_RowData,
  MRT_VisibilityState,
  useMaterialReactTable,
} from 'material-react-table';
import { useEffect } from 'react';
import { mergeSx } from '@/utils';
import { ICONS_OPACITY } from './table.constants';
import { TableProps } from './table.types';
import usePersistentColumnVisibility from './usePersistentColumnVisibility';

const NoDataAlertMessage = ({ message, ...rest }: { message: string } & AlertProps) => {
  const { sx, ...alertProps } = rest;
  return (
    <Alert
      severity="info"
      sx={{
        alignItems: 'center',
        margin: 1,
        ...sx,
      }}
      {...alertProps}
    >
      {message}
    </Alert>
  );
};

function Table<T extends MRT_RowData>(props: TableProps<T>) {
  const {
    data,
    columns,
    muiTablePaperProps,
    muiTopToolbarProps,
    muiSearchTextFieldProps,
    muiFilterTextFieldProps,
    displayColumnDefOptions,
    noDataMessage = 'No data',
    noDataAlertProps,
    emptyFilterResultsMessage = 'No data found',
    hideExpandAllIcon,
    tableName,
    tableInstanceRef,
    state,
    initialState,
    emptyState,
    enableRowHoverAction = false,
    rowHoverAction = () => {},
    muiTableBodyRowProps,
    muiTableProps,
    muiTableHeadProps,
    muiTableBodyProps,
    muiDetailPanelProps,
    muiTableContainerProps,
    ...rest
  } = props;
  const [columnVisibility, setColumnVisibility] = usePersistentColumnVisibility(tableName);

  let columnVisibilityState: MRT_VisibilityState | undefined = {};
  let restOfState = {};

  if (state) {
    const { columnVisibility: cv, ...rest } = state;
    columnVisibilityState = cv;
    restOfState = rest;
  }

  const stopPropagation = (e: Event) => {
    e.stopPropagation();
  };

  useEffect(() => {
    const hideColumnsIcon = document.querySelector('[aria-label="Show/Hide columns"]');
    const showFiltersIcon = document.querySelector('[aria-label="Show/Hide filters"]');
    const globalFilterIcon = document.querySelector('[aria-label="Show/Hide search"]');
    const elementsWithExpandLabel = document.querySelectorAll('[aria-label="Column Actions"]');

    if (!data.length) {
      hideColumnsIcon?.addEventListener('click', stopPropagation);
      showFiltersIcon?.addEventListener('click', stopPropagation);
      globalFilterIcon?.addEventListener('click', stopPropagation);
      elementsWithExpandLabel.forEach((element) => {
        element.addEventListener('click', stopPropagation);
      });
    }

    return () => {
      globalFilterIcon?.removeEventListener('click', stopPropagation);
      showFiltersIcon?.removeEventListener('click', stopPropagation);
      hideColumnsIcon?.removeEventListener('click', stopPropagation);
      elementsWithExpandLabel.forEach((element) => {
        element.removeEventListener('click', stopPropagation);
      });
    };
  }, [data]);

  // disable hiding for first 2 columns
  const customColumns = columns.map((col, index) => {
    if (index < 2) {
      return { ...col, enableHiding: false };
    }
    return col;
  });

  const rowActionsConsumerOpts = displayColumnDefOptions?.['mrt-row-actions'];
  const rowExpandConsumerOpts = displayColumnDefOptions?.['mrt-row-expand'];

  const table = useMaterialReactTable({
    renderEmptyRowsFallback: ({ table: { getPreFilteredRowModel } }) => (
      <>
        {/* This means there was data before filtering, so we show the message of empty filtering result */}
        {getPreFilteredRowModel().rows.length > 0 ? (
          <NoDataAlertMessage message={emptyFilterResultsMessage} {...noDataAlertProps} />
        ) : emptyState ? (
          <Box>{emptyState}</Box>
        ) : (
          <NoDataAlertMessage message={noDataMessage} {...noDataAlertProps} />
        )}
      </>
    ),
    layoutMode: 'grid',
    enablePagination: data.length > 10,
    enableBottomToolbar: rest.enablePagination ?? data.length > 10,
    enableDensityToggle: false,
    enableFullScreenToggle: false,
    enableSorting: !!data.length,
    autoResetAll: false,
    onColumnVisibilityChange: setColumnVisibility,
    icons: {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      KeyboardDoubleArrowDownIcon: (propsIcon: any) =>
        data.length ? (
          <KeyboardDoubleArrowDownIcon {...propsIcon} />
        ) : (
          <KeyboardDoubleArrowDownIcon sx={{ opacity: ICONS_OPACITY }} />
        ),
      SearchIcon: () => <SearchIcon sx={{ opacity: !data.length ? ICONS_OPACITY : 1 }} />,
      FilterListIcon: () => <FilterListIcon sx={{ opacity: !data.length ? ICONS_OPACITY : 1 }} />,
      ViewColumnIcon: () => <ViewColumnIcon sx={{ opacity: !data.length ? ICONS_OPACITY : 1 }} />,
      MoreVertIcon: () => <MoreVertIcon sx={{ opacity: !data.length ? ICONS_OPACITY : 1 }} />,
    },
    muiExpandAllButtonProps: { sx: { height: 40, width: 40 } },
    muiExpandButtonProps: { sx: { height: 40, width: 40 } },
    muiSearchTextFieldProps: ({ table }) => ({
      variant: 'outlined',
      size: 'small',
      ...(typeof muiSearchTextFieldProps === 'function'
        ? muiSearchTextFieldProps({ table })
        : muiSearchTextFieldProps),
    }),
    muiFilterTextFieldProps: ({ column, table, rangeFilterIndex }) => {
      const consumer =
        typeof muiFilterTextFieldProps === 'function'
          ? muiFilterTextFieldProps({ column, table, rangeFilterIndex })
          : muiFilterTextFieldProps;
      const isRangeInput = rangeFilterIndex !== undefined;
      const { filterVariant } = column.columnDef;
      const usesPicker =
        !!filterVariant &&
        (filterVariant.startsWith('date') ||
          filterVariant.startsWith('time') ||
          filterVariant === 'autocomplete');
      const filterFn = table.getState().columnFilterFns?.[column.id];
      // Own the end adornment so consumer adornments, the select caret and the
      // clear button coexist (MRT drops its clear button when one is passed).
      const managesEndAdornment = !usesPicker && filterFn !== 'empty' && filterFn !== 'notEmpty';

      const filterValue = column.getFilterValue();
      const hasFilterValue = Array.isArray(filterValue)
        ? filterValue.some((value) => value !== undefined && value !== null && value !== '')
        : filterValue !== undefined && filterValue !== null && filterValue !== '';

      const { slotProps: consumerSlotProps, ...consumerRest } = consumer ?? {};
      const consumerInputProps =
        typeof consumerSlotProps?.input === 'function' ? undefined : consumerSlotProps?.input;

      const clearLabel = table.options.localization.clearFilter;
      const clearAdornment =
        managesEndAdornment && !isRangeInput && hasFilterValue ? (
          <InputAdornment position="end">
            <Tooltip title={clearLabel}>
              <IconButton
                aria-label={clearLabel}
                size="small"
                onClick={(event) => {
                  event.stopPropagation();
                  column.setFilterValue(filterVariant === 'multi-select' ? [] : undefined);
                  table.refs.filterInputRefs.current?.[`${column.id}-0`]?.focus?.();
                }}
              >
                <CloseIcon />
              </IconButton>
            </Tooltip>
          </InputAdornment>
        ) : null;
      const endAdornment =
        consumerInputProps?.endAdornment || clearAdornment ? (
          <>
            {consumerInputProps?.endAdornment}
            {clearAdornment}
          </>
        ) : undefined;

      return {
        variant: 'outlined',
        size: 'small',
        ...consumerRest,
        slotProps: {
          ...consumerSlotProps,
          input: managesEndAdornment ? { ...consumerInputProps, endAdornment } : consumerInputProps,
        },
        sx: mergeSx({ minWidth: 0, width: '100%', mx: 0 }, consumer?.sx),
      };
    },
    muiToolbarAlertBannerProps: {
      color: 'neutral',
      icon: false,
      sx: (theme: Theme) => ({
        borderRadius: 1.25,
        fontSize: theme.typography.body2.fontSize,
        padding: '4px 8px',
        '& .MuiAlert-message': { padding: 0 },
        '& .MuiAlert-message > .MuiBox-root': { padding: 0 },
      }),
    },
    positionActionsColumn: 'last',
    muiTablePaperProps: ({ table }) => {
      const consumer =
        typeof muiTablePaperProps === 'function'
          ? muiTablePaperProps({ table })
          : muiTablePaperProps;
      return {
        elevation: 0,
        ...consumer,
        sx: mergeSx({ backgroundColor: 'transparent' }, consumer?.sx),
      };
    },
    muiTableContainerProps: ({ table }) => {
      const consumer =
        typeof muiTableContainerProps === 'function'
          ? muiTableContainerProps({ table })
          : muiTableContainerProps;
      return {
        ...consumer,
        sx: mergeSx(
          (theme: Theme) => ({
            backgroundColor: theme.palette.background.paper,
            borderBottom: `1px solid ${theme.palette.dividers?.divider}`,
          }),
          consumer?.sx
        ),
      };
    },
    muiTopToolbarProps: (args) => {
      const resolvedTopToolbarProps =
        typeof muiTopToolbarProps === 'function' ? muiTopToolbarProps(args) : muiTopToolbarProps;
      const { sx: muiTopToolbarPropsSx = {}, ...muiTopToolbarRestProps } =
        resolvedTopToolbarProps || {};
      return {
        ...muiTopToolbarRestProps,
        sx: mergeSx(
          {
            backgroundColor: 'transparent',
            '& > .MuiBox-root': {
              alignItems: 'center',
              flexDirection: 'row-reverse',
              flexWrap: 'wrap',
              '& > .MuiBox-root:has(.MuiCollapse-root)': {
                gap: 1,
              },
              '& > .MuiBox-root:has(.percona-table-internal-actions)': {
                marginRight: 'auto',
              },
            },
          },
          muiTopToolbarPropsSx
        ),
      };
    },
    displayColumnDefOptions: {
      ...displayColumnDefOptions,
      'mrt-row-actions': {
        header: 'Actions',
        Header: () => (
          <Box component="span" sx={visuallyHidden}>
            Actions
          </Box>
        ),
        size: 56,
        muiTableBodyCellProps: { align: 'right', sx: { px: 1 } },
        muiTableHeadCellProps: { align: 'right', sx: { px: 1 } },
        ...rowActionsConsumerOpts,
      },
      'mrt-row-expand': {
        ...rowExpandConsumerOpts,
        muiTableHeadCellProps: (args) => {
          const consumer = rowExpandConsumerOpts?.muiTableHeadCellProps;
          const resolved = typeof consumer === 'function' ? consumer(args) : consumer;
          return {
            ...resolved,
            sx: mergeSx(hideExpandAllIcon ? { '& button': { display: 'none' } } : {}, resolved?.sx),
          };
        },
      },
    },
    muiColumnActionsButtonProps: {
      size: 'medium',
      // MRT has no open-state hook for its menu, so flag the anchor while open.
      // DOM-only on purpose: a state update here re-renders mid-click and kills the menu.
      onClickCapture: (event) => {
        if (data.length) {
          event.currentTarget.classList.add('percona-column-actions-open');
        }
      },
      // the closing menu restores focus to the anchor
      onFocus: (event) => event.currentTarget.classList.remove('percona-column-actions-open'),
      sx: {
        height: 40,
        width: 40,
        m: 0,
        transform: 'none',
      },
    },
    muiTableProps: ({ table }) => {
      const consumer =
        typeof muiTableProps === 'function' ? muiTableProps({ table }) : muiTableProps;
      return {
        'data-testid': tableName,
        ...consumer,
        sx: mergeSx(
          (theme: Theme) => ({
            '& .MuiTableCell-root': {
              borderBottomColor: theme.palette.dividers?.divider,
            },
            '& .MuiTableCell-head': {
              typography: 'subHead2',
              transition: theme.transitions.create('background-color', {
                duration: theme.transitions.duration.shortest,
              }),
            },
            '& .MuiTableCell-head:has(.MuiCheckbox-root), & .MuiTableCell-head:has([aria-label="Expand all"])':
              {
                paddingTop: '18px',
              },
            '& .MuiTableCell-body': {
              typography: 'body1',
            },
            '& .MuiTableBody-root tr:last-of-type td': {
              borderBottom: 'none',
            },
            '& .MuiTableCell-head .MuiCollapse-wrapperInner > .MuiBox-root': {
              gap: 0.5,
            },
            '& .MuiTableCell-head .Mui-TableHeadCell-Content-Actions': {
              marginLeft: 'auto',
            },
            '& .MuiTableCell-head .Mui-TableHeadCell-Content-Labels': {
              gap: 0.5,
            },
            // "Filtered by" indicator: undo MRT's 0.75 downscale so its icon renders at 20px,
            // pairing with the 20px column-actions icon
            '& .MuiTableCell-head .Mui-TableHeadCell-Content-Labels > span .MuiIconButton-root': {
              width: 32,
              height: 32,
              margin: 0,
              padding: '6px',
              transform: 'none',
              '& > .MuiSvgIcon-root': { width: 20, height: 20, fontSize: 20 },
            },
            '& .MuiTableCell-head .Mui-TableHeadCell-Content-Labels .MuiTableSortLabel-root': {
              transform: 'none',
              width: 20,
              margin: 0,
              opacity: 0,
              transition: theme.transitions.create('opacity', {
                duration: theme.transitions.duration.shortest,
              }),
              '@media (hover: none)': { opacity: 0.5 },
              '& .MuiTableSortLabel-icon': { width: 20, height: 20, fontSize: 20, margin: 0 },
            },
            '& .MuiTableCell-head .Mui-TableHeadCell-Content-Actions .MuiIconButton-root': {
              opacity: 0,
              transition: theme.transitions.create('opacity', {
                duration: theme.transitions.duration.shortest,
              }),
              '@media (hover: none)': { opacity: 0.5 },
            },
            // :has(:focus-visible), not :focus-within: reveal for keyboard focus only,
            // so mouse-clicking sort doesn't pin the controls via lingering focus
            '& .MuiTableCell-head:is(:hover, :focus-visible, :has(:focus-visible)) .MuiTableSortLabel-root, & .MuiTableCell-head:is(:hover, :focus-visible, :has(:focus-visible)) .Mui-TableHeadCell-Content-Actions .MuiIconButton-root':
              {
                opacity: 0.5,
              },
            '& .MuiTableCell-head .MuiTableSortLabel-root:hover, & .MuiTableCell-head .MuiTableSortLabel-root:focus-visible, & .MuiTableCell-head .Mui-TableHeadCell-Content-Actions .MuiIconButton-root:hover, & .MuiTableCell-head .Mui-TableHeadCell-Content-Actions .MuiIconButton-root:focus-visible':
              {
                opacity: 1,
              },
            '& .MuiTableCell-head[data-sort] .MuiTableSortLabel-root': {
              opacity: 1,
            },
            '& .MuiTableCell-head .Mui-TableHeadCell-Content-Actions .MuiIconButton-root.percona-column-actions-open':
              {
                opacity: 1,
                backgroundColor: theme.palette.action.selected,
              },
            '& .MuiTableCell-head:has(.Mui-TableHeadCell-Content-Actions .MuiIconButton-root:hover), & .MuiTableCell-head:has(.percona-column-actions-open)':
              {
                backgroundColor: theme.palette.action.hover,
              },
          }),
          consumer?.sx
        ),
      };
    },
    muiTableHeadProps: ({ table }) => {
      const consumer =
        typeof muiTableHeadProps === 'function' ? muiTableHeadProps({ table }) : muiTableHeadProps;
      return {
        ...consumer,
        sx: mergeSx(
          { '& tr': { backgroundColor: 'background.paper', boxShadow: 'none' } },
          consumer?.sx
        ),
      };
    },
    muiBottomToolbarProps: {
      sx: {
        backgroundColor: 'transparent',
        boxShadow: 'none',
        '& label': {
          transform: 'none !important',
        },
      },
    },
    muiTableBodyProps: ({ table }) => {
      const consumer =
        typeof muiTableBodyProps === 'function' ? muiTableBodyProps({ table }) : muiTableBodyProps;
      return {
        ...consumer,
        sx: mergeSx(
          (theme: Theme) => ({
            '& tr': {
              backgroundColor: `${theme.palette.background.paper} !important`,
              minHeight: 'unset',
            },
          }),
          consumer?.sx
        ),
      };
    },
    muiDetailPanelProps: ({ table, row }) => {
      const consumer =
        typeof muiDetailPanelProps === 'function'
          ? muiDetailPanelProps({ table, row })
          : muiDetailPanelProps;
      return {
        ...consumer,
        sx: mergeSx({ width: '100%' }, consumer?.sx),
      };
    },
    ...rest,
    columns: customColumns,
    data,
    state: {
      columnVisibility: { ...columnVisibility, ...(columnVisibilityState ?? {}) },
      ...restOfState,
    },
    initialState: {
      columnVisibility,
      ...initialState,
    },
    muiTableBodyRowProps: (args) => {
      const { row, isDetailPanel } = args;
      const ownProps =
        (typeof muiTableBodyRowProps === 'function'
          ? muiTableBodyRowProps(args)
          : muiTableBodyRowProps) || {};
      const { sx, onClick, ...restOfProps } = ownProps;
      return {
        onClick: (e) => {
          if (
            !isDetailPanel &&
            enableRowHoverAction &&
            e.currentTarget.contains(e.target as Node)
          ) {
            rowHoverAction(row);
            onClick?.(e);
          }
        },
        sx: mergeSx(
          (theme: Theme) => ({
            ...(!isDetailPanel &&
              enableRowHoverAction && {
                cursor: 'pointer',
              }),
            '&:hover td': {
              backgroundColor: theme.palette.primary.hover,
            },
            '&.Mui-selected td': {
              backgroundColor: theme.palette.primary.selected,
            },
            '&.Mui-selected:hover td': {
              backgroundColor: theme.palette.primary.focus,
            },
          }),
          sx
        ),
        ...restOfProps,
      };
    },
  });

  useEffect(() => {
    if (!tableInstanceRef) {
      return;
    }
    tableInstanceRef.current = table;
    return () => {
      tableInstanceRef.current = null;
    };
  }, [table, tableInstanceRef]);

  return <MaterialReactTable table={table} />;
}

export default Table;
