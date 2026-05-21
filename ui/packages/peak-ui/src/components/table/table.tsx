import {
  FilterList as FilterListIcon,
  KeyboardDoubleArrowDown as KeyboardDoubleArrowDownIcon,
  MoreVert as MoreVertIcon,
  Search as SearchIcon,
  ViewColumn as ViewColumnIcon,
} from '@mui/icons-material';
import { Alert, AlertProps, Box, SxProps, Theme } from '@mui/material';
import visuallyHidden from '@mui/utils/visuallyHidden';
import { MaterialReactTable, MRT_VisibilityState } from 'material-react-table';
import { useEffect } from 'react';
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

const mergeSx = (
  defaultSx: SxProps<Theme> | undefined,
  consumerSx: SxProps<Theme> | undefined
): SxProps<Theme> | undefined => {
  if (!consumerSx) return defaultSx;
  if (!defaultSx) return consumerSx;
  return [
    ...(Array.isArray(defaultSx) ? defaultSx : [defaultSx]),
    ...(Array.isArray(consumerSx) ? consumerSx : [consumerSx]),
  ] as SxProps<Theme>;
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function Table<T extends Record<string, any>>(props: TableProps<T>) {
  const {
    data,
    columns,
    muiTablePaperProps,
    muiTopToolbarProps,
    muiSearchTextFieldProps,
    muiTableHeadCellFilterTextFieldProps,
    displayColumnDefOptions,
    noDataMessage = 'No data',
    noDataAlertProps,
    emptyFilterResultsMessage = 'No data found',
    hideExpandAllIcon,
    tableName,
    state,
    initialState,
    emptyState,
    enableRowHoverAction = false,
    rowHoverAction = () => {},
    muiTableBodyRowProps,
    muiTableProps,
    muiTableHeadProps,
    muiTableBodyProps,
    muiTableDetailPanelProps,
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

  // @ts-expect-error MRT muiTopToolbarProps type doesn't expose sx directly
  const { sx: muiTopToolbarPropsSx = {}, ...muiTopToolbarRestProps } = muiTopToolbarProps || {};

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

  return (
    <MaterialReactTable
      renderEmptyRowsFallback={({ table: { getPreFilteredRowModel } }) => (
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
      )}
      layoutMode="grid"
      enablePagination={data.length > 10}
      enableBottomToolbar={rest.enablePagination ?? data.length > 10}
      enableDensityToggle={false}
      enableFullScreenToggle={false}
      enableSorting={!!data.length}
      autoResetAll={false}
      onColumnVisibilityChange={setColumnVisibility}
      icons={{
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
      }}
      muiExpandAllButtonProps={{ sx: { height: 40, width: 40 } }}
      muiExpandButtonProps={{ sx: { height: 40, width: 40 } }}
      muiSearchTextFieldProps={({ table }) => ({
        variant: 'outlined',
        size: 'small',
        ...(typeof muiSearchTextFieldProps === 'function'
          ? muiSearchTextFieldProps({ table })
          : muiSearchTextFieldProps),
      })}
      muiTableHeadCellFilterTextFieldProps={({ column, table, rangeFilterIndex }) => {
        const consumer =
          typeof muiTableHeadCellFilterTextFieldProps === 'function'
            ? muiTableHeadCellFilterTextFieldProps({ column, table, rangeFilterIndex })
            : muiTableHeadCellFilterTextFieldProps;
        const isRangeInput = rangeFilterIndex !== undefined;
        return {
          variant: 'outlined',
          size: 'small',
          ...consumer,
          sx: mergeSx(isRangeInput ? { minWidth: 0, width: '100%', mx: 0 } : {}, consumer?.sx),
        };
      }}
      muiToolbarAlertBannerProps={{
        color: 'neutral',
        icon: false,
        sx: (theme: Theme) => ({
          borderRadius: 1.25,
          fontSize: theme.typography.body2.fontSize,
          padding: '4px 8px',
          '& .MuiAlert-message': { padding: 0 },
          '& .MuiAlert-message > .MuiBox-root': { padding: 0 },
        }),
      }}
      positionExpandColumn="last"
      positionActionsColumn="last"
      muiTablePaperProps={({ table }) => {
        const consumer =
          typeof muiTablePaperProps === 'function'
            ? muiTablePaperProps({ table })
            : muiTablePaperProps;
        return {
          elevation: 0,
          ...consumer,
          sx: mergeSx({ backgroundColor: 'transparent' }, consumer?.sx),
        };
      }}
      muiTableContainerProps={({ table }) => {
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
      }}
      muiTopToolbarProps={{
        sx: {
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
          ...muiTopToolbarPropsSx,
        },
        ...muiTopToolbarRestProps,
      }}
      displayColumnDefOptions={{
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
              sx: mergeSx(
                hideExpandAllIcon ? { '& button': { display: 'none' } } : {},
                resolved?.sx
              ),
            };
          },
        },
      }}
      muiTableHeadCellColumnActionsButtonProps={{
        size: 'medium',
        sx: {
          height: 40,
          width: 40,
          m: 0,
          transform: 'none',
        },
      }}
      muiTableProps={({ table }) => {
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
              '& .MuiTableCell-head .Mui-TableHeadCell-Content-Labels .MuiTableSortLabel-root': {
                transform: 'none',
              },
              '& .MuiTableCell-head:has(.Mui-TableHeadCell-Content-Actions .MuiIconButton-root:hover)':
                {
                  backgroundColor: theme.palette.action.hover,
                },
            }),
            consumer?.sx
          ),
        };
      }}
      muiTableHeadProps={({ table }) => {
        const consumer =
          typeof muiTableHeadProps === 'function'
            ? muiTableHeadProps({ table })
            : muiTableHeadProps;
        return {
          ...consumer,
          sx: mergeSx(
            { '& tr': { backgroundColor: 'background.paper', boxShadow: 'none' } },
            consumer?.sx
          ),
        };
      }}
      muiBottomToolbarProps={{
        sx: { backgroundColor: 'transparent', boxShadow: 'none' },
      }}
      muiTableBodyProps={({ table }) => {
        const consumer =
          typeof muiTableBodyProps === 'function'
            ? muiTableBodyProps({ table })
            : muiTableBodyProps;
        return {
          ...consumer,
          sx: mergeSx(
            { '& tr': { backgroundColor: 'background.paper' }, minHeight: 'unset' },
            consumer?.sx
          ),
        };
      }}
      muiTableDetailPanelProps={({ table, row }) => {
        const consumer =
          typeof muiTableDetailPanelProps === 'function'
            ? muiTableDetailPanelProps({ table, row })
            : muiTableDetailPanelProps;
        return {
          ...consumer,
          sx: mergeSx({ width: '100%' }, consumer?.sx),
        };
      }}
      {...rest}
      columns={customColumns}
      data={data}
      state={{
        columnVisibility: { ...columnVisibility, ...columnVisibilityState },
        ...restOfState,
      }}
      initialState={{
        columnVisibility,
        ...initialState,
      }}
      muiTableBodyRowProps={(args) => {
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
      }}
    />
  );
}

export default Table;
