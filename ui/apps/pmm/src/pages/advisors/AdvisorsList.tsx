import { FC, useCallback, useMemo } from 'react';
import Button from '@mui/material/Button';
import { boxClasses } from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import PlayArrowOutlinedIcon from '@mui/icons-material/PlayArrowOutlined';
import { Table } from '@percona/percona-ui';
import { enqueueSnackbar } from 'notistack';
import { Page } from 'components/page';
import {
  useAdvisors,
  useChangeAdvisorChecks,
  useStartAdvisorChecks,
} from 'hooks/api/useAdvisors';
import { AdvisorCheckRow, AdvisorInterval } from 'types/advisors.types';
import { OrgRole } from 'types/user.types';
import { flattenAdvisorChecks } from 'utils/advisors.utils';
import { capitalize } from 'utils/text.utils';
import { ADVISOR_FAMILY, ADVISOR_INTERVAL } from 'lib/constants';
import { Messages } from './AdvisorsList.messages';
import { getAdvisorsColumns } from './AdvisorsList.constants';

const AdvisorsList: FC = () => {
  const { data: advisors = [], isLoading } = useAdvisors();
  const { mutate: startChecks, isPending: isStarting } =
    useStartAdvisorChecks();
  const { mutate: changeChecks } = useChangeAdvisorChecks();

  const rows = useMemo(() => flattenAdvisorChecks(advisors), [advisors]);

  const categories = useMemo(
    () =>
      [...new Set(rows.map((row) => capitalize(row.category)))].sort(),
    [rows]
  );

  const vendors = useMemo(
    () => [...new Set(rows.map((row) => ADVISOR_FAMILY[row.family]))].sort(),
    [rows]
  );

  const runChecks = useCallback(
    (names: string[], message: string) =>
      startChecks(names, {
        onSuccess: () => enqueueSnackbar(message, { variant: 'success' }),
      }),
    [startChecks]
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

  const columns = useMemo(
    () =>
      getAdvisorsColumns({
        categories,
        vendors,
        onToggleCheck: handleToggleCheck,
        onChangeInterval: handleChangeInterval,
      }),
    [categories, vendors, handleToggleCheck, handleChangeInterval]
  );

  return (
    <Page
      title={Messages.title}
      fullWidth
      wide
      roles={[OrgRole.Editor, OrgRole.Admin]}
    >
      <Stack gap={2} sx={{ flex: 1 }}>
        <Typography variant="body2">{Messages.description}</Typography>
        <Table
          tableName="advisors-list-table"
          columns={columns}
          data={rows}
          noDataMessage={Messages.noData}
          state={{ isLoading }}
          initialState={{
            pagination: {
              pageSize: 25,
              pageIndex: 0,
            },
            showGlobalFilter: true,
          }}
          enableStickyHeader
          enableHiding={false}
          enableRowActions
          positionActionsColumn="last"
          renderRowActions={({ row }) => (
            <Button
              color="inherit"
              size="small"
              disabled={!row.original.enabled || isStarting}
              onClick={() =>
                runChecks(
                  [row.original.checkName],
                  Messages.success.checkStarted(row.original.summary)
                )
              }
              data-testid={`check-${row.original.checkName}-run`}
            >
              {Messages.run}
            </Button>
          )}
          renderTopToolbarCustomActions={({ table }) => {
            const { globalFilter, columnFilters } = table.getState();
            // no active search/filter means no selection ("all")
            const hasSelection =
              !!globalFilter ||
              columnFilters.some(
                (filter) => filter.value !== undefined && filter.value !== ''
              );
            // enabled checks currently visible after global search + filters
            const selectedNames = hasSelection
              ? table
                  .getFilteredRowModel()
                  .rows.map((row) => row.original)
                  .filter((row) => row.enabled)
                  .map((row) => row.checkName)
              : [];

            return (
              <Stack direction="row" alignItems="center" gap={2}>
                <Button
                  startIcon={<PlayArrowOutlinedIcon />}
                  disabled={isStarting}
                  onClick={() => runChecks([], Messages.success.checksStarted)}
                  data-testid="run-all-checks"
                >
                  {Messages.runAll}
                </Button>
                <Button
                  startIcon={<PlayArrowOutlinedIcon />}
                  disabled={isStarting || !selectedNames.length}
                  onClick={() =>
                    runChecks(selectedNames, Messages.success.checksStarted)
                  }
                  data-testid="run-selected-checks"
                >
                  {Messages.runSelected}
                </Button>
                <Button
                  startIcon={<AddOutlinedIcon />}
                  data-testid="add-advisor"
                >
                  {Messages.addAdvisor}
                </Button>
              </Stack>
            );
          }}
          muiSearchTextFieldProps={{
            // inline style: MRT's toolbar overwrites any `sx` passed here
            style: { width: 400 },
          }}
          muiTopToolbarProps={{
            sx: {
              // search on the left, action buttons on the right,
              // flush with the table edges
              [`& > .${boxClasses.root}`]: {
                alignItems: 'center',
                flexDirection: 'row-reverse',
                px: 0,
              },
            },
          }}
          muiTableContainerProps={{
            sx: {
              flex: 1,
              borderRadius: 2,
              border: '1px solid',
              borderColor: 'divider',
            },
          }}
        />
      </Stack>
    </Page>
  );
};

export default AdvisorsList;
