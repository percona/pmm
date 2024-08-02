import { ReactElement, ReactPortal } from 'react';
import {
  CircularProgress,
  Table as MuiTable,
  Paper,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import { TableProps } from './Table.types';
import { Messages } from './Table.messages';

export const Table = <T,>({
  columns,
  rowId,
  rows = [],
  emptyMessage = Messages.noData,
  isLoading,
}: TableProps<T>): ReactElement => (
  <TableContainer
    component={Paper}
    sx={() => ({
      borderWidth: 1,
      borderRadius: 1.5,
      borderStyle: 'solid',
      borderColor: 'rgba(44, 50, 62, 0.25)',
      boxShadow: 'none',
    })}
  >
    <MuiTable>
      <TableHead>
        <TableRow>
          {columns.map((col, idx) => (
            <TableCell key={idx}>
              <Typography variant="subHead2">{col.name}</Typography>
            </TableCell>
          ))}
        </TableRow>
      </TableHead>
      <TableBody>
        {isLoading ? (
          <TableRow>
            <TableCell colSpan={columns.length} align="center">
              <CircularProgress
                size={20}
                data-testid="table-loading-indicator"
              />
            </TableCell>
          </TableRow>
        ) : !rows.length ? (
          <TableRow>
            <TableCell colSpan={columns.length} align="center">
              {emptyMessage}
            </TableCell>
          </TableRow>
        ) : (
          rows.map((row) => (
            <TableRow key={row[rowId] as React.Key}>
              {columns.map((col) => (
                <TableCell key={col.field as React.Key}>
                  {col.cell ? (
                    col.cell(row)
                  ) : (
                    <Typography>{row[col.field] as ReactPortal}</Typography>
                  )}
                </TableCell>
              ))}
            </TableRow>
          ))
        )}
      </TableBody>
    </MuiTable>
  </TableContainer>
);
