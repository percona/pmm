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
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward';
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward';
import { FC, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Page } from 'components/page';
import { useInvestigationsList, useCreateInvestigation } from 'hooks/api/useInvestigations';
import { CreateInvestigationModal } from './CreateInvestigationModal';
import { PMM_NEW_NAV_PATH } from 'lib/constants';
import type { CreateInvestigationBody, Investigation, InvestigationListItem } from 'api/investigations';
import { useSnackbar } from 'notistack';

type SortColumn = 'title' | 'status' | 'created_at' | 'updated_at';
type SortOrder = 'asc' | 'desc';

const InvestigationsListPage: FC = () => {
  const navigate = useNavigate();
  const { enqueueSnackbar } = useSnackbar();
  const [searchParams] = useSearchParams();
  const [modalOpen, setModalOpen] = useState(false);
  const [orderBy, setOrderBy] = useState<SortColumn>('updated_at');
  const [order, setOrder] = useState<SortOrder>('desc');
  const { data: list, isLoading, isError, error } = useInvestigationsList({
    orderBy,
    order,
  });

  const handleSort = (column: SortColumn) => {
    if (orderBy === column) {
      setOrder(order === 'asc' ? 'desc' : 'asc');
    } else {
      setOrderBy(column);
      setOrder(column === 'title' || column === 'status' ? 'asc' : 'desc');
    }
  };

  const SortIcon = ({ column }: { column: SortColumn }) =>
    orderBy === column ? (
      order === 'asc' ? (
        <ArrowUpwardIcon sx={{ fontSize: 16, verticalAlign: 'middle', ml: 0.25 }} />
      ) : (
        <ArrowDownwardIcon sx={{ fontSize: 16, verticalAlign: 'middle', ml: 0.25 }} />
      )
    ) : null;
  const createMutation = useCreateInvestigation();

  const initialFromParams = useMemo(() => {
    const sourceType = searchParams.get('source_type') ?? undefined;
    const sourceRef = searchParams.get('source_ref') ?? undefined;
    const timeFrom = searchParams.get('time_from') ?? undefined;
    const timeTo = searchParams.get('time_to') ?? undefined;
    const title =
      searchParams.get('title') ?? (sourceType ? `Investigation: ${sourceType}` : undefined);
    return { title, sourceType, sourceRef, timeFrom, timeTo };
  }, [searchParams]);

  const handleCreateClick = () => setModalOpen(true);

  const handleSubmit = (body: CreateInvestigationBody) => {
    createMutation.mutate(body, {
      onSuccess: (inv: Investigation) => {
        setModalOpen(false);
        navigate(`${PMM_NEW_NAV_PATH}/investigations/${inv.id}`);
      },
      onError: (err: Error) => {
        enqueueSnackbar(
          err?.message ?? 'Failed to create investigation',
          { variant: 'error' }
        );
      },
    });
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
            onClick={handleCreateClick}
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
                    <TableCell
                      onClick={() => handleSort('title')}
                      sx={{ cursor: 'pointer', userSelect: 'none' }}
                    >
                      Title <SortIcon column="title" />
                    </TableCell>
                    <TableCell
                      onClick={() => handleSort('status')}
                      sx={{ cursor: 'pointer', userSelect: 'none' }}
                    >
                      Status <SortIcon column="status" />
                    </TableCell>
                    <TableCell
                      onClick={() => handleSort('created_at')}
                      sx={{ cursor: 'pointer', userSelect: 'none' }}
                    >
                      Created <SortIcon column="created_at" />
                    </TableCell>
                    <TableCell
                      onClick={() => handleSort('updated_at')}
                      sx={{ cursor: 'pointer', userSelect: 'none' }}
                    >
                      Updated <SortIcon column="updated_at" />
                    </TableCell>
                    <TableCell align="right">Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {investigations.map((inv: InvestigationListItem) => (
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
                        {(inv.created_at ?? inv.createdAt)
                          ? new Date(inv.created_at ?? inv.createdAt).toLocaleString()
                          : '—'}
                      </TableCell>
                      <TableCell>
                        {(inv.updated_at ?? inv.updatedAt)
                          ? new Date(inv.updated_at ?? inv.updatedAt).toLocaleString()
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
      <CreateInvestigationModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onSubmit={handleSubmit}
        isPending={createMutation.isPending}
        initial={initialFromParams}
      />
    </Page>
  );
};

export default InvestigationsListPage;
