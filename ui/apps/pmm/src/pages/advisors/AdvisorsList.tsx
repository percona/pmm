import { FC, useCallback, useMemo } from 'react';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
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
import { RunMenuButton, RunMenuOption } from './run-menu-button';

const AdvisorsList: FC = () => {
  const { data: advisors = [], isLoading } = useAdvisors();
  const { mutate: startChecks, isPending: isStarting } =
    useStartAdvisorChecks();
  const { mutate: changeChecks } = useChangeAdvisorChecks();

  const rows = useMemo(() => flattenAdvisorChecks(advisors), [advisors]);

  const categoryOptions = useMemo<RunMenuOption[]>(
    () =>
      [...new Set(rows.map((row) => row.category))].sort().map((category) => ({
        label: capitalize(category),
        names: rows
          .filter((row) => row.category === category && row.enabled)
          .map((row) => row.checkName),
      })),
    [rows]
  );

  const technologyOptions = useMemo<RunMenuOption[]>(
    () =>
      [...new Set(rows.map((row) => row.family))].sort().map((family) => ({
        label: ADVISOR_FAMILY[family],
        names: rows
          .filter((row) => row.family === family && row.enabled)
          .map((row) => row.checkName),
      })),
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
        categories: categoryOptions.map((option) => option.label),
        technologies: technologyOptions.map((option) => option.label),
        onToggleCheck: handleToggleCheck,
        onChangeInterval: handleChangeInterval,
      }),
    [categoryOptions, technologyOptions, handleToggleCheck, handleChangeInterval]
  );

  return (
    <Page title={Messages.title} fullWidth roles={[OrgRole.Editor, OrgRole.Admin]}>
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
          renderTopToolbarCustomActions={() => (
            <Stack direction="row" alignItems="center" gap={2}>
              <Button
                startIcon={<PlayArrowOutlinedIcon />}
                disabled={isStarting}
                onClick={() => runChecks([], Messages.success.checksStarted)}
                data-testid="run-all-checks"
              >
                {Messages.runAll}
              </Button>
              <RunMenuButton
                label={Messages.runCategory}
                options={categoryOptions}
                disabled={isStarting}
                onRun={(option) =>
                  runChecks(option.names, Messages.success.checksStarted)
                }
              />
              <RunMenuButton
                label={Messages.runTechnology}
                options={technologyOptions}
                disabled={isStarting}
                onRun={(option) =>
                  runChecks(option.names, Messages.success.checksStarted)
                }
              />
            </Stack>
          )}
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
