import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  Chip,
  CircularProgress,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward';
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward';
import { FC, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Page } from 'components/page';
import { useInvestigationsList, useCreateInvestigation, useDeleteInvestigation } from 'hooks/api/useInvestigations';
import { CreateInvestigationModal } from './CreateInvestigationModal';
import { PMM_NEW_NAV_PATH } from 'lib/constants';
import type { CreateInvestigationBody, Investigation, InvestigationListItem } from 'api/investigations';
import { useSnackbar } from 'notistack';

type SortColumn = 'title' | 'status' | 'created_at' | 'updated_at';
type SortOrder = 'asc' | 'desc';
type TriggerFilter = 'all' | 'auto' | 'manual';

const InvestigationsListPage: FC = () => {
  const navigate = useNavigate();
  const { enqueueSnackbar } = useSnackbar();
  const [searchParams] = useSearchParams();
  const [modalOpen, setModalOpen] = useState(false);
  const [orderBy, setOrderBy] = useState<SortColumn>('created_at');
  const [order, setOrder] = useState<SortOrder>('desc');
  const [trigger, setTrigger] = useState<TriggerFilter>('all');
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const { data: list, isLoading, isError, error } = useInvestigationsList({
    orderBy,
    order,
    trigger: trigger === 'all' ? undefined : trigger,
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
  const deleteMutation = useDeleteInvestigation();

  const handleToggleRow = (id: string) => {
    setSelectedIds((prev: Set<string>) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

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

  const handleSelectAll = () => {
    if (selectedIds.size === investigations.length && investigations.length > 0) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(investigations.map((inv: InvestigationListItem) => inv.id)));
    }
  };

  const handleDeleteSelected = async () => {
    if (selectedIds.size === 0) return;
    const ids = Array.from(selectedIds);
    const count = ids.length;
    try {
      await Promise.all(ids.map((id) => deleteMutation.mutateAsync(id)));
      setSelectedIds(new Set());
      enqueueSnackbar(`Deleted ${count} investigation${count === 1 ? '' : 's'}`, { variant: 'success' });
    } catch (err) {
      enqueueSnackbar(err instanceof Error ? err.message : 'Failed to delete some investigations', { variant: 'error' });
    }
  };

  return (
    <Page
      title="Investigations"
      topBar={
        <Stack direction="row" justifyContent="space-between" alignItems="center" gap={1} sx={{ mb: 1 }}>
          <ToggleButtonGroup
            size="small"
            exclusive
            value={trigger}
            onChange={(_, value: TriggerFilter | null) => value && setTrigger(value)}
            aria-label="Filter investigations by trigger"
          >
            <ToggleButton value="all">All</ToggleButton>
            <ToggleButton value="auto">Auto</ToggleButton>
            <ToggleButton value="manual">Manual</ToggleButton>
          </ToggleButtonGroup>
          <Stack direction="row" alignItems="center" gap={1}>
            {selectedIds.size > 0 && (
              <Button
                variant="outlined"
                color="error"
                onClick={handleDeleteSelected}
                disabled={deleteMutation.isPending}
              >
                Delete selected ({selectedIds.size})
              </Button>
            )}
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={handleCreateClick}
              disabled={createMutation.isPending}
            >
              New investigation
            </Button>
          </Stack>
        </Stack>
      }
    >
      {/* flexShrink:0 keeps the Page's flex column from squishing the card.
          MUI Card defaults to overflow:hidden, so a shrunk card clips its
          content instead of letting the page scroll. */}
      <Box sx={{ flexShrink: 0 }}>
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
                    <TableCell padding="checkbox">
                      <Checkbox
                        indeterminate={selectedIds.size > 0 && selectedIds.size < investigations.length}
                        checked={investigations.length > 0 && selectedIds.size === investigations.length}
                        onChange={handleSelectAll}
                        aria-label="Select all"
                      />
                    </TableCell>
                    <TableCell
                      onClick={() => handleSort('title')}
                      sx={{ cursor: 'pointer', userSelect: 'none' }}
                    >
                      Title <SortIcon column="title" />
                    </TableCell>
                    <TableCell>Source</TableCell>
                    <TableCell>Node</TableCell>
                    <TableCell>Service</TableCell>
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
                      <TableCell padding="checkbox">
                        <Checkbox
                          checked={selectedIds.has(inv.id)}
                          onChange={() => handleToggleRow(inv.id)}
                          aria-label={`Select ${inv.title || inv.id}`}
                        />
                      </TableCell>
                      <TableCell>{inv.title || inv.id}</TableCell>
                      <TableCell>
                        {(inv.sourceType ?? inv.source_type) === 'alert'
                          ? 'Alert'
                          : 'User request'}
                      </TableCell>
                      <TableCell>
                        {inv.nodeName ?? inv.node_name ?? '—'}
                      </TableCell>
                      <TableCell>
                        {inv.serviceName ?? inv.service_name ?? '—'}
                      </TableCell>
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
      </Box>
    </Page>
  );
};

export default InvestigationsListPage;
