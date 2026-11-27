import {
  Alert,
  Box,
  Button,
  Checkbox,
  CircularProgress,
  Link,
  MenuItem,
  Select,
  Stack,
  Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import { formatDistanceToNowStrict } from 'date-fns';
import { FC, ReactNode, useMemo, useState } from 'react';
import {
  Link as RouterLink,
  useNavigate,
  useSearchParams,
} from 'react-router-dom';
import { AdrePageShell } from 'components/adre/AdrePageShell';
import {
  useInvestigationsList,
  useCreateInvestigation,
  useDeleteInvestigation,
} from 'hooks/api/useInvestigations';
import { CreateInvestigationModal } from './CreateInvestigationModal';
import { PMM_NEW_NAV_PATH } from 'lib/constants';
import type {
  CreateInvestigationBody,
  Investigation,
  InvestigationListItem,
} from 'api/investigations';
import { useSnackbar } from 'notistack';
import { adreColors } from 'components/adre/adreAnswerSx';
import { StatusPill } from './components/StatusPill';
import {
  accentButtonSx,
  pillButtonSx,
  sourceLabel,
} from './investigationStyles';

type SortColumn = 'title' | 'status' | 'created_at' | 'updated_at';
type SortOrder = 'asc' | 'desc';
type TriggerFilter = 'all' | 'auto' | 'manual';

const SORTS: { value: string; label: string }[] = [
  { value: 'created_at:desc', label: 'Newest first' },
  { value: 'created_at:asc', label: 'Oldest first' },
  { value: 'updated_at:desc', label: 'Recently updated' },
  { value: 'title:asc', label: 'Title' },
  { value: 'status:asc', label: 'Status' },
];

const TRIGGERS: { value: TriggerFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'auto', label: 'Auto-investigated' },
  { value: 'manual', label: 'Manual' },
];

const SEVERITY_COLORS: Record<string, string> = {
  critical: '#FCA5A5',
  emergency: '#FCA5A5',
  alert: '#FCA5A5',
  error: '#FCA5A5',
  warning: '#FCD34D',
};

const investigationPath = (id: string) =>
  `${PMM_NEW_NAV_PATH}/investigations/${id}`;

const relativeTime = (iso?: string) => {
  const d = iso ? new Date(iso) : undefined;
  if (!d || Number.isNaN(d.getTime())) return '';
  return formatDistanceToNowStrict(d, { addSuffix: true });
};

const checkboxSx = {
  p: 0.5,
  color: adreColors.muted,
  '&.Mui-checked, &.MuiCheckbox-indeterminate': { color: adreColors.accent },
} as const;

const InvestigationRow: FC<{
  inv: InvestigationListItem;
  selected: boolean;
  onToggle: () => void;
}> = ({ inv, selected, onToggle }) => {
  const createdAt = inv.createdAt ?? inv.created_at;
  const updatedAt = inv.updatedAt ?? inv.updated_at;
  const meta = [
    sourceLabel(inv.sourceType ?? inv.source_type),
    inv.nodeName ?? inv.node_name,
    inv.serviceName ?? inv.service_name,
  ].filter(Boolean);
  const severity = inv.severity?.toLowerCase();

  return (
    <Box
      component="li"
      sx={{
        position: 'relative',
        display: 'grid',
        gridTemplateColumns: {
          xs: 'auto minmax(0, 1fr)',
          sm: 'auto minmax(0, 1fr) auto',
        },
        columnGap: 1.5,
        rowGap: 1,
        alignItems: 'start',
        px: 2,
        py: 1.75,
        borderTop: `1px solid ${adreColors.divider}`,
        '&:first-of-type': { borderTop: 0 },
        '&:hover': { bgcolor: 'rgba(255,255,255,0.03)' },
        '&:hover .inv-title': { color: adreColors.accentText },
      }}
    >
      <Checkbox
        checked={selected}
        onChange={onToggle}
        size="small"
        inputProps={{ 'aria-label': `Select ${inv.title || inv.id}` }}
        sx={{ ...checkboxSx, position: 'relative', zIndex: 1, mt: '-2px' }}
      />
      <Box sx={{ minWidth: 0 }}>
        {/* The title link covers the whole row, so the row opens the investigation. */}
        <Link
          className="inv-title"
          component={RouterLink}
          to={investigationPath(inv.id)}
          underline="none"
          sx={{
            display: 'block',
            fontSize: 15,
            fontWeight: 500,
            lineHeight: 1.4,
            color: adreColors.text,
            overflowWrap: 'anywhere',
            '&::after': { content: '""', position: 'absolute', inset: 0 },
          }}
        >
          {inv.title || inv.id}
        </Link>
        {inv.summary && (
          <Typography
            title={inv.summary}
            sx={{
              mt: 0.25,
              fontSize: 13,
              lineHeight: 1.5,
              color: adreColors.secondary,
              display: '-webkit-box',
              WebkitLineClamp: 1,
              WebkitBoxOrient: 'vertical',
              overflow: 'hidden',
            }}
          >
            {inv.summary}
          </Typography>
        )}
        <Stack
          direction="row"
          flexWrap="wrap"
          alignItems="center"
          columnGap={1}
          sx={{ mt: 0.5, fontSize: 12, color: adreColors.muted }}
        >
          {severity && (
            <Box
              component="span"
              sx={{ color: SEVERITY_COLORS[severity] ?? adreColors.secondary }}
            >
              {severity}
            </Box>
          )}
          {meta.map((m, i) => (
            <span key={m}>
              {(severity || i > 0) && '· '}
              {m}
            </span>
          ))}
        </Stack>
      </Box>
      <Stack
        alignItems={{ xs: 'flex-start', sm: 'flex-end' }}
        gap={0.75}
        sx={{ gridColumn: { xs: 2, sm: 3 } }}
      >
        <StatusPill status={inv.status} />
        <Typography
          title={[
            createdAt && `Created ${new Date(createdAt).toLocaleString()}`,
            updatedAt && `Updated ${new Date(updatedAt).toLocaleString()}`,
          ]
            .filter(Boolean)
            .join('\n')}
          sx={{ fontSize: 12, color: adreColors.muted, whiteSpace: 'nowrap' }}
        >
          {relativeTime(createdAt)}
        </Typography>
      </Stack>
    </Box>
  );
};

const InvestigationsListPage: FC = () => {
  const navigate = useNavigate();
  const { enqueueSnackbar } = useSnackbar();
  const [searchParams] = useSearchParams();
  const [modalOpen, setModalOpen] = useState(false);
  const [sort, setSort] = useState(SORTS[0].value);
  const [trigger, setTrigger] = useState<TriggerFilter>('all');
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [orderBy, order] = sort.split(':') as [SortColumn, SortOrder];
  const {
    data: list,
    isLoading,
    isError,
    error,
  } = useInvestigationsList({
    orderBy,
    order,
    trigger: trigger === 'all' ? undefined : trigger,
  });
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
      searchParams.get('title') ??
      (sourceType ? `Investigation: ${sourceType}` : undefined);
    return { title, sourceType, sourceRef, timeFrom, timeTo };
  }, [searchParams]);

  const handleSubmit = (body: CreateInvestigationBody) => {
    createMutation.mutate(body, {
      onSuccess: (inv: Investigation) => {
        setModalOpen(false);
        navigate(investigationPath(inv.id));
      },
      onError: (err: Error) => {
        enqueueSnackbar(err?.message ?? 'Failed to create investigation', {
          variant: 'error',
        });
      },
    });
  };

  const investigations = list ?? [];

  const handleSelectAll = () => {
    if (
      selectedIds.size === investigations.length &&
      investigations.length > 0
    ) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(
        new Set(investigations.map((inv: InvestigationListItem) => inv.id))
      );
    }
  };

  const handleDeleteSelected = async () => {
    if (selectedIds.size === 0) return;
    const ids = Array.from(selectedIds);
    const count = ids.length;
    try {
      await Promise.all(ids.map((id) => deleteMutation.mutateAsync(id)));
      setSelectedIds(new Set());
      enqueueSnackbar(
        `Deleted ${count} investigation${count === 1 ? '' : 's'}`,
        { variant: 'success' }
      );
    } catch (err) {
      enqueueSnackbar(
        err instanceof Error
          ? err.message
          : 'Failed to delete some investigations',
        { variant: 'error' }
      );
    }
  };

  let body: ReactNode;
  if (isLoading) {
    body = (
      <Box display="flex" justifyContent="center" p={4}>
        <CircularProgress size={28} sx={{ color: adreColors.accent }} />
      </Box>
    );
  } else if (isError) {
    body = (
      <Alert
        severity="error"
        variant="outlined"
        sx={{ color: adreColors.body }}
      >
        Failed to load investigations. {(error as Error)?.message}
      </Alert>
    );
  } else if (investigations.length === 0) {
    body = (
      <Box
        sx={{
          p: 4,
          textAlign: 'center',
          borderRadius: '16px',
          bgcolor: adreColors.panel,
          border: `1px solid ${adreColors.divider}`,
          color: adreColors.secondary,
          fontSize: 14,
        }}
      >
        {trigger === 'all'
          ? 'No investigations yet. Start one from an alert, a chat, or here.'
          : 'No investigations match this filter.'}
      </Box>
    );
  } else {
    body = (
      <Box
        sx={{
          borderRadius: '16px',
          bgcolor: adreColors.panel,
          border: `1px solid ${adreColors.divider}`,
          overflow: 'hidden',
        }}
      >
        <Stack
          direction="row"
          alignItems="center"
          gap={1.5}
          sx={{
            px: 2,
            py: 1,
            borderBottom: `1px solid ${adreColors.divider}`,
            fontSize: 12,
            color: adreColors.muted,
          }}
        >
          <Checkbox
            size="small"
            indeterminate={
              selectedIds.size > 0 && selectedIds.size < investigations.length
            }
            checked={selectedIds.size === investigations.length}
            onChange={handleSelectAll}
            inputProps={{ 'aria-label': 'Select all' }}
            sx={checkboxSx}
          />
          {selectedIds.size > 0
            ? `${selectedIds.size} selected`
            : `${investigations.length} investigation${investigations.length === 1 ? '' : 's'}`}
        </Stack>
        <Box
          component="ul"
          aria-label="Investigations"
          sx={{ m: 0, p: 0, listStyle: 'none' }}
        >
          {investigations.map((inv: InvestigationListItem) => (
            <InvestigationRow
              key={inv.id}
              inv={inv}
              selected={selectedIds.has(inv.id)}
              onToggle={() => handleToggleRow(inv.id)}
            />
          ))}
        </Box>
      </Box>
    );
  }

  return (
    <AdrePageShell
      title="Investigations"
      subtitle="What the Assistant looked into, with its root cause and next steps."
      actions={
        <>
          {selectedIds.size > 0 && (
            <Button
              variant="outlined"
              onClick={handleDeleteSelected}
              disabled={deleteMutation.isPending}
              sx={{
                ...pillButtonSx,
                color: '#FCA5A5',
                borderColor: 'rgba(248,113,113,0.4)',
              }}
            >
              Delete selected ({selectedIds.size})
            </Button>
          )}
          <Button
            variant="contained"
            disableElevation
            startIcon={<AddIcon />}
            onClick={() => setModalOpen(true)}
            disabled={createMutation.isPending}
            sx={accentButtonSx}
          >
            New investigation
          </Button>
        </>
      }
    >
      <Stack
        direction="row"
        flexWrap="wrap"
        alignItems="center"
        justifyContent="space-between"
        gap={1.5}
        sx={{ mb: 1.5 }}
      >
        <Stack
          direction="row"
          role="group"
          aria-label="Filter investigations by trigger"
          sx={{
            p: 0.5,
            gap: 0.5,
            borderRadius: 999,
            bgcolor: adreColors.panel,
            border: `1px solid ${adreColors.divider}`,
          }}
        >
          {TRIGGERS.map((t) => (
            <Button
              key={t.value}
              aria-pressed={trigger === t.value}
              onClick={() => {
                setTrigger(t.value);
                setSelectedIds(new Set());
              }}
              sx={{
                height: 30,
                px: 1.75,
                borderRadius: 999,
                textTransform: 'none',
                fontSize: 13,
                fontWeight: 500,
                color:
                  trigger === t.value ? adreColors.text : adreColors.secondary,
                bgcolor:
                  trigger === t.value
                    ? 'rgba(255,255,255,0.08)'
                    : 'transparent',
                '&:hover': { bgcolor: 'rgba(255,255,255,0.06)' },
              }}
            >
              {t.label}
            </Button>
          ))}
        </Stack>
        <Select
          size="small"
          value={sort}
          onChange={(e) => setSort(e.target.value)}
          inputProps={{ 'aria-label': 'Sort investigations' }}
          sx={{
            minWidth: 180,
            fontSize: 13,
            borderRadius: 999,
            color: adreColors.text,
            bgcolor: adreColors.panel,
            '& fieldset': { borderColor: adreColors.border },
            '& .MuiSelect-icon': { color: adreColors.secondary },
          }}
        >
          {SORTS.map((s) => (
            <MenuItem key={s.value} value={s.value}>
              {s.label}
            </MenuItem>
          ))}
        </Select>
      </Stack>

      {body}

      <CreateInvestigationModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onSubmit={handleSubmit}
        isPending={createMutation.isPending}
        initial={initialFromParams}
      />
    </AdrePageShell>
  );
};

export default InvestigationsListPage;
