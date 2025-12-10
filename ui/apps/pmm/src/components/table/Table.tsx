import { paperClasses, Stack } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import { Table as PeakTable, TableProps } from '@percona/ui-lib';

const Table = <T extends Record<string, any>>(props: TableProps<T>) => {
  const theme = useTheme();
  const backgroundColor = theme.palette.background.default;

  return (
    <Stack
      id="test"
      sx={{
        [`& > .${paperClasses.root}`]: {
          overflow: 'hidden',
          borderWidth: 1,
          borderStyle: 'solid',
          borderColor: theme.palette.divider,
          borderRadius: theme.shape.borderRadius,
          backgroundColor: 'transparent',
        },
      }}
    >
      <PeakTable
        {...props}
        muiTableContainerProps={{
          sx: {
            backgroundColor,
          },
          ...props.muiTableContainerProps,
        }}
        muiTableHeadProps={{
          sx: {
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
      />
    </Stack>
  );
};

export default Table;
