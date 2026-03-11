import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import { FC } from 'react';
import { useNavigate } from 'react-router-dom';
import { Page } from 'components/page';
import { useInvestigationsList, useCreateInvestigation } from 'hooks/api/useInvestigations';
import { PMM_NEW_NAV_PATH } from 'lib/constants';

const InvestigationsListPage: FC = () => {
  const navigate = useNavigate();
  const { data: list, isLoading, isError, error } = useInvestigationsList();
  const createMutation = useCreateInvestigation();

  const handleCreate = () => {
    createMutation.mutate(
      { title: 'New investigation' },
      {
        onSuccess: (inv) => {
          navigate(`${PMM_NEW_NAV_PATH}/investigations/${inv.id}`);
        },
      }
    );
  };

  if (isLoading) {
    return (
      <Page title="Investigations">
        <Box display="flex" justifyContent="center" p={4}>
          <CircularProgress />
        </Box>
      </Page>
    );
  }

  if (isError) {
    return (
      <Page title="Investigations">
        <Card variant="outlined">
          <CardContent>
            <Alert severity="error">
              Failed to load investigations. {(error as Error)?.message}
            </Alert>
          </CardContent>
        </Card>
      </Page>
    );
  }

  const investigations = list ?? [];

  return (
    <Page
      title="Investigations"
      topBar={
        <Stack direction="row" justifyContent="flex-end" sx={{ mb: 1 }}>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={handleCreate}
            disabled={createMutation.isPending}
          >
            New investigation
          </Button>
        </Stack>
      }
    >
      <Card variant="outlined">
          <CardContent>
            {investigations.length === 0 ? (
              <Typography color="text.secondary">
                No investigations yet. Create one to get started.
              </Typography>
            ) : (
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Title</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>Updated</TableCell>
                    <TableCell align="right">Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {investigations.map((inv) => (
                    <TableRow key={inv.id} hover>
                      <TableCell>{inv.title || inv.id}</TableCell>
                      <TableCell>
                        <Chip
                          label={inv.status}
                          size="small"
                          variant="outlined"
                        />
                      </TableCell>
                      <TableCell>
                        {inv.updatedAt
                          ? new Date(inv.updatedAt).toLocaleString()
                          : '—'}
                      </TableCell>
                      <TableCell align="right">
                        <Button
                          size="small"
                          onClick={() =>
                            navigate(`${PMM_NEW_NAV_PATH}/investigations/${inv.id}`)
                          }
                        >
                          Open
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
    </Page>
  );
};

export default InvestigationsListPage;
