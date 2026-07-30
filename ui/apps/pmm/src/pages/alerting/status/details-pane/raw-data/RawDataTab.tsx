import Stack from '@mui/material/Stack';
import { FC } from 'react';
import Table from '@mui/material/Table';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import TableCell from '@mui/material/TableCell';
import Typography from '@mui/material/Typography';
import { Messages } from './RawDataTab.messages';
import TableBody from '@mui/material/TableBody';
import { AlertDetailsPane } from '../AlertDetailsPane.types';
import { CodeBlock } from '@percona/percona-ui';

interface Props {
  details: AlertDetailsPane;
}

const RawDataTab: FC<Props> = ({ details }) => (
  <Stack
    direction={{ xs: 'column', md: 'row' }}
    spacing={2}
    sx={{
      '& > *': {
        width: { xs: '100%', md: '50%' },
      },
    }}
  >
    <Stack spacing={2}>
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
            {Object.entries(details.rawData.labels).map(([name, value]) => (
              <TableRow key={name}>
                <TableCell sx={{ color: 'text.secondary' }}>{name}</TableCell>
                <TableCell>{value}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Stack>
    <Stack spacing={2}>
      <Typography variant="h6">{Messages.json.title}</Typography>
      <CodeBlock
        language="json"
        content={details.rawData.json}
        copyable
        // TODO: extend CodeBlock with line numbers support
        // showLineNumbers
        maxHeight="80vh"
        data-testid="alert-raw-data"
      />
    </Stack>
  </Stack>
);

export default RawDataTab;
