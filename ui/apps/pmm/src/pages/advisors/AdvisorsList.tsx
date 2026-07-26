import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import DeleteOutlineOutlinedIcon from '@mui/icons-material/DeleteOutlineOutlined';
import DnsOutlinedIcon from '@mui/icons-material/DnsOutlined';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import FilterAltOffOutlinedIcon from '@mui/icons-material/FilterAltOffOutlined';
import PlayArrowOutlinedIcon from '@mui/icons-material/PlayArrowOutlined';
import PlaylistAddCheckOutlinedIcon from '@mui/icons-material/PlaylistAddCheckOutlined';
import Badge from '@mui/material/Badge';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogTitle from '@mui/material/DialogTitle';
import IconButton from '@mui/material/IconButton';
import MenuItem from '@mui/material/MenuItem';
import { paperClasses } from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { Table } from '@percona/percona-ui';
import { Page } from 'components/page';
import {
  useAdvisors,
  useChangeAdvisorChecks,
  useDeleteAdvisorCheck,
  useStartAdvisorChecks,
} from 'hooks/api/useAdvisors';
import { ADVISOR_INTERVAL, ADVISOR_TECHNOLOGY } from 'lib/constants';
import {
  type MRT_PaginationState,
  type MRT_Updater,
} from 'material-react-table';
import { closeSnackbar, enqueueSnackbar } from 'notistack';
import { FC, useCallback, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { AdvisorCheckRow, AdvisorInterval } from 'types/advisors.types';
import { OrgRole } from 'types/user.types';
import { flattenAdvisorChecks } from 'utils/advisors.utils';
import { getAdvisorsColumns, INTERVAL_OPTIONS } from './AdvisorsList.constants';
import { Messages } from './AdvisorsList.messages';
import { AdvisorCheckForm, AdvisorCheckFormMode } from './check-form';
import { AdvisorCheckDetailsPane } from './details-pane';
import { DisableServicesDrawer } from './disable-services';

interface CheckFilters {
  category: string;
  subcategory: string;
  technology: string;
  interval: string;
  status: string;
}

const DEFAULT_PAGE_SIZE = 50;

interface FilterOption {
  label: string;
  value: string;
}

interface FilterSelectProps {
  id: string;
  label: string;
  options: FilterOption[];
  value: string;
  onChange: (value: string) => void;
}

const FilterSelect: FC<FilterSelectProps> = ({
  id,
  label,
  options,
  value,
  onChange,
}) => (
  <TextField
    select
    size="small"
    // deterministic id: React's useId default breaks jsdom selector matching
    id={`${id}-filter-select`}
    label={label}
    value={value || Messages.filters.all}
    onChange={(e) =>
      onChange(e.target.value === Messages.filters.all ? '' : e.target.value)
    }
    sx={{ minWidth: 160 }}
    data-testid={`${id}-filter`}
  >
    <MenuItem value={Messages.filters.all}>{Messages.filters.all}</MenuItem>
    {options.map((option) => (
      <MenuItem key={option.value} value={option.value}>
        {option.label}
      </MenuItem>
    ))}
  </TextField>
);

const AdvisorsList: FC = () => {
  const navigate = useNavigate();
  const { data: advisors = [], isLoading } = useAdvisors();
  const { mutate: startChecks, isPending: isStarting } =
    useStartAdvisorChecks();
  const { mutate: changeChecks } = useChangeAdvisorChecks();
  const { mutate: deleteCheck, isPending: isDeleting } =
    useDeleteAdvisorCheck();

  const [detailsMaximized, setDetailsMaximized] = useState(false);
  const [checkForm, setCheckForm] = useState<{
    mode: AdvisorCheckFormMode;
    checkName?: string;
  } | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<AdvisorCheckRow | null>(
    null
  );
  const [servicesTargetName, setServicesTargetName] = useState<string | null>(
    null
  );

  const [searchParams, setSearchParams] = useSearchParams();

  // the URL query string is the single source of truth for filters +
  // pagination, so a shared link reproduces the exact same view
  const search = searchParams.get('search') || '';
  const filters = useMemo<CheckFilters>(
    () => ({
      category: searchParams.get('category') || '',
      subcategory: searchParams.get('subcategory') || '',
      technology: searchParams.get('technology') || '',
      interval: searchParams.get('interval') || '',
      status: searchParams.get('status') || '',
    }),
    [searchParams]
  );
  const pagination: MRT_PaginationState = {
    pageIndex: Math.max(0, (Number(searchParams.get('page')) || 1) - 1),
    pageSize: Number(searchParams.get('pageSize')) || DEFAULT_PAGE_SIZE,
  };

  // apply query-string changes; filter/search changes reset to the first page
  const patchParams = (
    mutate: (params: URLSearchParams) => void,
    { resetPage = true }: { resetPage?: boolean } = {}
  ) => {
    const next = new URLSearchParams(searchParams);
    mutate(next);
    if (resetPage) {
      next.delete('page');
    }
    setSearchParams(next, { replace: true });
  };

  const openDetails = (check: AdvisorCheckRow, maximized = false) => {
    setDetailsMaximized(maximized);
    patchParams((p) => p.set('details', check.checkName), { resetPage: false });
  };

  const setSearch = (value: string) =>
    patchParams((p) => {
      if (value) {
        p.set('search', value);
      } else {
        p.delete('search');
      }
    });

  const updateFilter = (name: keyof CheckFilters, value: string) =>
    patchParams((p) => {
      if (value) {
        p.set(name, value);
      } else {
        p.delete(name);
      }
    });

  const handleClearFilters = () => {
    // keep the chosen page size; drop every filter, search and the page
    const next = new URLSearchParams();
    const pageSize = searchParams.get('pageSize');
    if (pageSize) {
      next.set('pageSize', pageSize);
    }
    setSearchParams(next, { replace: true });
  };

  const handlePaginationChange = (
    updater: MRT_Updater<MRT_PaginationState>
  ) => {
    const nextPagination =
      typeof updater === 'function' ? updater(pagination) : updater;
    patchParams(
      (p) => {
        p.set('page', String(nextPagination.pageIndex + 1));
        p.set('pageSize', String(nextPagination.pageSize));
      },
      { resetPage: false }
    );
  };

  const rows = useMemo(() => flattenAdvisorChecks(advisors), [advisors]);

  // the open details overlay is driven by the ?details=<checkName> URL param,
  // so the overlay is deep-linkable and reproduced by a shared URL
  const detailsCheck = useMemo(
    () =>
      rows.find(
        (row) => row.checkName === (searchParams.get('details') || '')
      ) ?? null,
    [rows, searchParams]
  );

  // resolved from rows on every render so the open drawer always shows the
  // fresh disabled-services list after a mutation refetch
  const servicesTarget = useMemo(
    () => rows.find((row) => row.checkName === servicesTargetName) ?? null,
    [rows, servicesTargetName]
  );

  const categoryOptions = useMemo<FilterOption[]>(
    () =>
      [...new Set(rows.map((row) => row.category))]
        .sort()
        .map((category) => ({ label: category, value: category })),
    [rows]
  );

  const subcategoryOptions = useMemo<FilterOption[]>(
    () =>
      [...new Set(rows.map((row) => row.subcategory))]
        .sort()
        .map((subcategory) => ({ label: subcategory, value: subcategory })),
    [rows]
  );

  const technologyOptions = useMemo<FilterOption[]>(
    () =>
      [...new Set(rows.map((row) => ADVISOR_TECHNOLOGY[row.technology]))]
        .sort()
        .map((technology) => ({ label: technology, value: technology })),
    [rows]
  );

  const intervalOptions = useMemo<FilterOption[]>(
    () =>
      INTERVAL_OPTIONS.map((interval) => ({
        label: ADVISOR_INTERVAL[interval],
        value: ADVISOR_INTERVAL[interval],
      })),
    []
  );

  const statusOptions: FilterOption[] = [
    { label: Messages.status.enabled, value: Messages.status.enabled },
    { label: Messages.status.disabled, value: Messages.status.disabled },
  ];

  const filteredRows = useMemo(() => {
    const term = search.trim().toLowerCase();
    return rows.filter((row) => {
      if (filters.category && row.category !== filters.category) {
        return false;
      }
      if (filters.subcategory && row.subcategory !== filters.subcategory) {
        return false;
      }
      if (
        filters.technology &&
        ADVISOR_TECHNOLOGY[row.technology] !== filters.technology
      ) {
        return false;
      }
      if (
        filters.interval &&
        ADVISOR_INTERVAL[row.interval] !== filters.interval
      ) {
        return false;
      }
      if (filters.status) {
        const label = row.enabled
          ? Messages.status.enabled
          : Messages.status.disabled;
        if (label !== filters.status) {
          return false;
        }
      }
      if (term) {
        const haystack =
          `${row.summary} ${row.description} ${row.category} ${row.subcategory} ${ADVISOR_TECHNOLOGY[row.technology]}`.toLowerCase();
        if (!haystack.includes(term)) {
          return false;
        }
      }
      return true;
    });
  }, [rows, search, filters]);

  const hasActiveFilters =
    !!search.trim() || Object.values(filters).some(Boolean);

  // enabled checks currently visible after search + filters; "Run selected"
  // means "all", so it is only offered once a filter narrows the list
  const selectedNames = hasActiveFilters
    ? filteredRows.filter((row) => row.enabled).map((row) => row.checkName)
    : [];

  const runChecks = useCallback(
    (names: string[], message: string) =>
      startChecks(names, {
        onSuccess: (batchId) => {
          void navigator.clipboard.writeText(batchId);
          enqueueSnackbar(message, {
            variant: 'success',
            action: (key) => (
              <Button
                color="inherit"
                size="small"
                onClick={() => {
                  closeSnackbar(key);
                  navigate(`/advisors/insights?batchId=${batchId}`);
                }}
                data-testid="view-run-results"
              >
                {Messages.viewResults}
              </Button>
            ),
          });
        },
      }),
    [startChecks, navigate]
  );

  const handleToggleCheck = useCallback(
    (check: AdvisorCheckRow) =>
      changeChecks([{ name: check.checkName, enable: !check.enabled }], {
        onSuccess: () =>
          enqueueSnackbar(
            check.enabled
              ? Messages.success.checkDisabled(check.summary)
              : Messages.success.checkEnabled(check.summary),
            { variant: 'success' }
          ),
      }),
    [changeChecks]
  );

  const handleChangeInterval = useCallback(
    (check: AdvisorCheckRow, interval: AdvisorInterval) =>
      changeChecks([{ name: check.checkName, interval }], {
        onSuccess: () =>
          enqueueSnackbar(
            Messages.success.intervalChanged(
              check.summary,
              ADVISOR_INTERVAL[interval]
            ),
            { variant: 'success' }
          ),
      }),
    [changeChecks]
  );

  const handleDeleteConfirm = useCallback(() => {
    if (!deleteTarget) {
      return;
    }
    const target = deleteTarget;
    deleteCheck(target.checkName, {
      onSuccess: () => {
        enqueueSnackbar(Messages.success.checkDeleted(target.summary), {
          variant: 'success',
        });
        setDeleteTarget(null);
      },
    });
  }, [deleteCheck, deleteTarget]);

  const columns = useMemo(
    () =>
      getAdvisorsColumns({
        onToggleCheck: handleToggleCheck,
        onChangeInterval: handleChangeInterval,
      }),
    [handleToggleCheck, handleChangeInterval]
  );

  return (
    <Page
      title={Messages.title}
      fullWidth
      wide
      fillViewport
      footer={null}
      roles={[OrgRole.Editor, OrgRole.Admin]}
    >
      <Stack
        gap={2}
        sx={{
          flex: 1,
          minHeight: 0,
          // let the table fill the remaining height and scroll internally
          // (mirrors RealtimeTableWrapper), so the page itself does not scroll
          [`& > .${paperClasses.root}`]: {
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            minHeight: 0,
            overflow: 'hidden',
          },
        }}
      >
        <Typography variant="body2">{Messages.description}</Typography>
        <Stack direction="row" flexWrap="wrap" gap={2} alignItems="center">
          <TextField
            size="small"
            placeholder={Messages.searchPlaceholder}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            sx={{ minWidth: 240 }}
            data-testid="advisors-search"
          />
          <FilterSelect
            id="category"
            label={Messages.filters.category}
            options={categoryOptions}
            value={filters.category}
            onChange={(value) => updateFilter('category', value)}
          />
          <FilterSelect
            id="subcategory"
            label={Messages.filters.subcategory}
            options={subcategoryOptions}
            value={filters.subcategory}
            onChange={(value) => updateFilter('subcategory', value)}
          />
          <FilterSelect
            id="technology"
            label={Messages.filters.technology}
            options={technologyOptions}
            value={filters.technology}
            onChange={(value) => updateFilter('technology', value)}
          />
          <FilterSelect
            id="interval"
            label={Messages.filters.interval}
            options={intervalOptions}
            value={filters.interval}
            onChange={(value) => updateFilter('interval', value)}
          />
          <FilterSelect
            id="status"
            label={Messages.filters.status}
            options={statusOptions}
            value={filters.status}
            onChange={(value) => updateFilter('status', value)}
          />
          <Tooltip title={Messages.filters.clear} arrow>
            <Box component="span">
              <IconButton
                disabled={!hasActiveFilters}
                onClick={handleClearFilters}
                aria-label={Messages.filters.clear}
                data-testid="clear-filters"
              >
                <FilterAltOffOutlinedIcon />
              </IconButton>
            </Box>
          </Tooltip>
          <Tooltip title={Messages.runAll} arrow>
            <Box component="span">
              <IconButton
                disabled={isStarting}
                onClick={() => runChecks([], Messages.success.checksStarted)}
                aria-label={Messages.runAll}
                data-testid="run-all-checks"
              >
                <PlayArrowOutlinedIcon />
              </IconButton>
            </Box>
          </Tooltip>
          <Tooltip title={Messages.runSelected} arrow>
            <Box component="span">
              <IconButton
                disabled={isStarting || !selectedNames.length}
                onClick={() =>
                  runChecks(selectedNames, Messages.success.checksStarted)
                }
                aria-label={Messages.runSelected}
                data-testid="run-selected-checks"
              >
                <PlaylistAddCheckOutlinedIcon />
              </IconButton>
            </Box>
          </Tooltip>
          <Tooltip title={Messages.addAdvisor} arrow>
            <IconButton
              aria-label={Messages.addAdvisor}
              onClick={() => setCheckForm({ mode: 'create' })}
              data-testid="add-advisor"
            >
              <AddOutlinedIcon />
            </IconButton>
          </Tooltip>
        </Stack>
        <Table
          tableName="advisors-list-table"
          columns={columns}
          data={filteredRows}
          noDataMessage={Messages.noData}
          state={{ isLoading, pagination }}
          onPaginationChange={handlePaginationChange}
          enableStickyHeader
          enablePagination
          enableTopToolbar={false}
          enableGlobalFilter={false}
          enableColumnFilters={false}
          enableColumnActions={false}
          enableHiding={false}
          enableRowActions
          positionActionsColumn="last"
          // size the actions column so four icons fit centered, show the
          // "Actions" header (percona-ui hides it by default), and lock the width
          // (grow: false) so the grid layout doesn't stretch it to fill the row
          displayColumnDefOptions={{
            'mrt-row-actions': {
              header: Messages.columns.actions,
              // undefined lets MRT render the `header` string with its normal head-cell
              // layout (percona-ui defaults it to a visually-hidden span)
              Header: undefined,
              size: 150,
              grow: false,
              muiTableBodyCellProps: { align: 'center', sx: { px: 1 } },
              // MRT zeroes the bottom padding on display-column head cells;
              // match the 0.6rem data headers get so the label aligns with them
              muiTableHeadCellProps: {
                align: 'center',
                sx: {
                  px: 1,
                  pb: '0.6rem',
                  // align='center' only sets text-align. MRT's head-cell
                  // content row pushes its (here empty) actions box right with
                  // margin-left:auto, which eats the free space before
                  // justify-content applies — so stretch the label box to fill
                  // the row and centre the label inside it instead.
                  '& .Mui-TableHeadCell-Content-Labels': {
                    flex: 1,
                    justifyContent: 'center',
                  },
                },
              },
            },
          }}
          renderRowActions={({ row }) => (
            <Stack
              direction="row"
              gap={0.25}
              alignItems="center"
              justifyContent="center"
              // percona-ui's theme pins MuiIconButton to a fixed 40x40, so four
              // icons overflow the column however small their padding gets.
              // Unpin the box to collapse each button onto its icon.
              sx={{
                width: '100%',
                '& .MuiIconButton-root': {
                  width: 'auto',
                  height: 'auto',
                  p: 0.5,
                },
              }}
            >
              <Tooltip title={Messages.run} arrow>
                <Box component="span">
                  <IconButton
                    color="inherit"
                    disabled={!row.original.enabled || isStarting}
                    aria-label={Messages.run}
                    onClick={() =>
                      runChecks(
                        [row.original.checkName],
                        Messages.success.checkStarted(row.original.summary)
                      )
                    }
                    data-testid={`check-${row.original.checkName}-run`}
                  >
                    <PlayArrowOutlinedIcon />
                  </IconButton>
                </Box>
              </Tooltip>
              <Tooltip
                title={
                  row.original.disabledServiceIds.length
                    ? Messages.disabledForServices(
                        row.original.disabledServiceIds.length
                      )
                    : Messages.disableForServices
                }
                arrow
              >
                <IconButton
                  aria-label={Messages.disableForServices}
                  onClick={() => setServicesTargetName(row.original.checkName)}
                  data-testid={`check-${row.original.checkName}-services`}
                >
                  <Badge
                    // the count doubles as the "has disablements" signal, so the
                    // state is not conveyed by colour alone
                    badgeContent={row.original.disabledServiceIds.length}
                    color="warning"
                    overlap="circular"
                    data-testid={`check-${row.original.checkName}-services-badge`}
                    sx={{
                      '& .MuiBadge-badge': {
                        minWidth: 14,
                        height: 14,
                        p: '0 3px',
                        fontSize: 10,
                      },
                    }}
                  >
                    <DnsOutlinedIcon />
                  </Badge>
                </IconButton>
              </Tooltip>
              {row.original.userDefined && (
                <>
                  <Tooltip title={Messages.edit} arrow>
                    <IconButton
                      aria-label={Messages.edit}
                      onClick={() =>
                        setCheckForm({
                          mode: 'edit',
                          checkName: row.original.checkName,
                        })
                      }
                      data-testid={`check-${row.original.checkName}-edit`}
                    >
                      <EditOutlinedIcon />
                    </IconButton>
                  </Tooltip>
                  <Tooltip title={Messages.delete} arrow>
                    <IconButton
                      aria-label={Messages.delete}
                      onClick={() => setDeleteTarget(row.original)}
                      data-testid={`check-${row.original.checkName}-delete`}
                    >
                      <DeleteOutlineOutlinedIcon />
                    </IconButton>
                  </Tooltip>
                </>
              )}
            </Stack>
          )}
          muiTableBodyRowProps={({ row }) => ({
            'data-testid': `advisor-row-${row.original.checkName}`,
            // double-click opens the check details overlay maximized
            onDoubleClick: () => openDetails(row.original, true),
          })}
          muiTableContainerProps={{
            sx: {
              flex: 1,
              borderRadius: 2,
              border: '1px solid',
              borderColor: 'divider',
            },
          }}
        />
        <AdvisorCheckDetailsPane
          check={detailsCheck}
          initialMaximized={detailsMaximized}
          onClose={() =>
            patchParams((p) => p.delete('details'), { resetPage: false })
          }
          onClone={() => {
            if (detailsCheck) {
              setCheckForm({
                mode: 'clone',
                checkName: detailsCheck.checkName,
              });
              patchParams((p) => p.delete('details'), { resetPage: false });
            }
          }}
          onEdit={() => {
            if (detailsCheck) {
              setCheckForm({
                mode: 'edit',
                checkName: detailsCheck.checkName,
              });
              patchParams((p) => p.delete('details'), { resetPage: false });
            }
          }}
        />
        <AdvisorCheckForm
          open={!!checkForm}
          mode={checkForm?.mode ?? 'create'}
          checkName={checkForm?.checkName}
          onClose={() => setCheckForm(null)}
        />
        <DisableServicesDrawer
          check={servicesTarget}
          onClose={() => setServicesTargetName(null)}
        />
        <Dialog
          open={!!deleteTarget}
          onClose={() => setDeleteTarget(null)}
          data-testid="delete-advisor-dialog"
        >
          <DialogTitle>{Messages.deleteDialog.title}</DialogTitle>
          <DialogContent>
            <DialogContentText>
              {deleteTarget
                ? Messages.deleteDialog.body(
                    deleteTarget.summary || deleteTarget.checkName
                  )
                : ''}
            </DialogContentText>
          </DialogContent>
          <DialogActions>
            <Button
              color="inherit"
              onClick={() => setDeleteTarget(null)}
              data-testid="delete-advisor-cancel"
            >
              {Messages.deleteDialog.cancel}
            </Button>
            <Button
              color="error"
              variant="contained"
              disabled={isDeleting}
              onClick={handleDeleteConfirm}
              data-testid="delete-advisor-confirm"
            >
              {Messages.deleteDialog.confirm}
            </Button>
          </DialogActions>
        </Dialog>
      </Stack>
    </Page>
  );
};

export default AdvisorsList;
