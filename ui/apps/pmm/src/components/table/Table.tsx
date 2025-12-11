import { useTheme } from '@mui/material/styles';
import { Table as PeakTable, TableProps } from '@percona/ui-lib';

const Table = <T extends Record<string, any>>(props: TableProps<T>) => {
  const theme = useTheme();
  const backgroundColor = theme.palette.background.default;

  return (
    <PeakTable
      muiTablePaperProps={{
        sx: {
          flex: '1 1 0',
          display: 'flex',
          flexFlow: 'column',
          borderWidth: 1,
          borderStyle: 'solid',
          borderColor: theme.palette.divider,
          borderRadius: theme.shape.borderRadius,
          overflow: 'hidden',
        },
      }}
      muiTableContainerProps={{
        sx: {
          flex: '1 1 0',
          backgroundColor,
        },
        ...props.muiTableContainerProps,
      }}
      muiTableHeadProps={{
        sx: {
          tr: {
            boxShadow: 'none',
          },

          th: {
            backgroundColor,
          },
        },
      }}
      muiTableHeadCellProps={{
        sx: {
          p: 2,
          py: 1.75,
          backgroundColor,
        },
      }}
      muiTableHeadRowProps={{
        sx: {
          backgroundColor,
        },
      }}
      muiTableBodyRowProps={{
        sx: {
          backgroundColor,
        },
      }}
      muiTableFooterProps={{
        sx: {
          backgroundColor,
        },
      }}
      muiBottomToolbarProps={{
        sx: {
          backgroundColor,
        },
      }}
      muiTableHeadCellFilterTextFieldProps={{
        variant: 'outlined',
        size: 'small',
        sx: {
          marginTop: 1,
        },
      }}
      {...props}
    />
  );
};

export default Table;
