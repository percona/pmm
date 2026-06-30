import Stack from '@mui/material/Stack';
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { FC } from 'react';
import { AlertRow } from '../../AlertsPage.types';
import Table from '@mui/material/Table';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import TableCell from '@mui/material/TableCell';
import Typography from '@mui/material/Typography';
import { Messages } from './RawDataTab.messages';
import TableBody from '@mui/material/TableBody';

interface Props {
  alert: AlertRow;
}

const RawDataTab: FC<Props> = ({ alert }) => {
  return (
    <Stack
      direction={{ xs: 'column', sm: 'row' }}
      justifyContent="space-evenly"
      spacing={2}
    >
      <Stack flex={1} spacing={2}>
        <Typography variant="h6">{Messages.labels.title}</Typography>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow
                sx={{
                  backgroundColor: 'background.default',
                  th: {
                    color: 'text.secondary',
                  },
                }}
              >
                <TableCell>{Messages.labels.label}</TableCell>
                <TableCell>{Messages.labels.value}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {Object.entries(alert.labels).map(([key, value]) => (
                <TableRow key={key}>
                  <TableCell sx={{ color: 'text.secondary' }}>{key}</TableCell>
                  <TableCell>{value}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Stack>
      <Stack flex={1} spacing={2}>
        <Typography variant="h6">{Messages.json.title}</Typography>
        <SyntaxHighlighter
          language="json"
          content={alert.rawJson}
          showCopyButton
          showLineNumbers
          maxHeight="80vh"
          data-testid="alert-raw-data"
        />
      </Stack>
    </Stack>
  );
};

export default RawDataTab;
