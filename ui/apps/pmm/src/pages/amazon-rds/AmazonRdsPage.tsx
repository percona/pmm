import { useMemo, type FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Skeleton from '@mui/material/Skeleton';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import NotificationsOutlined from '@mui/icons-material/NotificationsOutlined';
import { NetworkIntelligenceIcon } from '@percona/peak-ui';
import { Page } from 'components/page';
import { useSettings } from 'contexts/settings';
import { useUser } from 'contexts/user';
import { useRdsInstances } from 'hooks/api/useRds';
import { ADVISORS_URL, ALERTING_URL } from './AmazonRds.constants';
import { Messages } from './AmazonRds.messages';
import type {
  FilteredGroups,
  GroupSelection,
  InstanceGroup,
} from './AmazonRds.types';
import {
  filterGroups,
  getAccountOptions,
  getEngineOptions,
  getGroupOptions,
  getRegionOptions,
  getTypeOptions,
  groupInstances,
  narrowSelection,
} from './AmazonRds.utils';
import { GroupsTable } from './table/GroupsTable';
import { RdsToolbar, type SelectFilter } from './toolbar/RdsToolbar';
import { useRdsFilters } from './useRdsFilters';

type RdsGroupsProps = ReturnType<typeof useRdsFilters> & {
  groups: InstanceGroup[];
  filtered: FilteredGroups;
};

const RdsGroups: FC<RdsGroupsProps> = ({
  groups,
  filtered,
  search,
  setSearch,
  filters,
  update,
  clear,
}) => {
  const selects = useMemo(
    (): SelectFilter[] => [
      {
        id: 'regions',
        label: Messages.columns.region,
        placeholder: Messages.allRegions,
        options: getRegionOptions(groups),
        value: filters.regions,
        width: 110,
      },
      {
        id: 'accounts',
        label: Messages.columns.awsAccount,
        placeholder: Messages.allAccounts,
        options: getAccountOptions(groups, filters.regions),
        value: filters.accounts,
        width: 150,
      },
      {
        id: 'engines',
        label: Messages.columns.engine,
        placeholder: Messages.allEngines,
        options: getEngineOptions(groups),
        value: filters.engines,
        width: 110,
      },
      {
        id: 'types',
        label: Messages.columns.type,
        placeholder: Messages.allTypes,
        options: getTypeOptions(groups),
        value: filters.types,
        width: 100,
      },
      {
        id: 'groups',
        label: Messages.columns.identifier,
        placeholder: Messages.allGroups,
        options: getGroupOptions(groups, filters),
        value: filters.groups,
        width: 150,
      },
    ],
    [groups, filters]
  );

  const handleSelectChange = (id: keyof GroupSelection, value: string[]) =>
    update(narrowSelection(groups, { ...filters, [id]: value }));

  return (
    <Stack gap={2}>
      <RdsToolbar
        search={search}
        onSearchChange={setSearch}
        selects={selects}
        onSelectChange={handleSelectChange}
      />
      {filtered.groups.length ? (
        <GroupsTable
          groups={filtered.groups}
          autoExpanded={filtered.autoExpanded}
          expansionKey={filters.query}
        />
      ) : (
        <Alert
          severity="info"
          action={
            <Button color="inherit" size="small" onClick={clear}>
              {Messages.clearFilters}
            </Button>
          }
        >
          {Messages.noMatch}
        </Alert>
      )}
    </Stack>
  );
};

export const AmazonRdsPage: FC = () => {
  const { user } = useUser();
  const { settings } = useSettings();
  const { data: rows, isError } = useRdsInstances();
  const groups = useMemo(() => groupInstances(rows ?? []), [rows]);
  const rdsFilters = useRdsFilters();
  const filtered = useMemo(
    () => filterGroups(groups, rdsFilters.filters),
    [groups, rdsFilters.filters]
  );
  // Same rule as the Advisors menu.
  const showAdvisors = !!user?.isEditor && !!settings?.advisorEnabled;

  const renderContent = () => {
    if (rows) {
      return groups.length ? (
        <RdsGroups groups={groups} filtered={filtered} {...rdsFilters} />
      ) : (
        <Alert severity="info">{Messages.noData}</Alert>
      );
    }

    if (isError) {
      return <Alert severity="error">{Messages.fetchError}</Alert>;
    }

    return (
      <Skeleton aria-label={Messages.loading} variant="rounded" height="50vh" />
    );
  };

  return (
    <Page title={Messages.title} hideTitle maxWidth="full">
      <Stack gap={2}>
        <Stack
          direction="row"
          flexWrap="wrap"
          alignItems="center"
          justifyContent="space-between"
          gap={1}
        >
          <Stack direction="row" alignItems="baseline" gap={2}>
            <Typography variant="h2">{Messages.title}</Typography>
            {!!groups.length && (
              <Typography variant="body2" color="text.secondary">
                {Messages.shownGroups(filtered.groups.length, groups.length)}
              </Typography>
            )}
          </Stack>
          <Stack direction="row" gap={1}>
            <Button
              component={RouterLink}
              to={ALERTING_URL}
              variant="outlined"
              startIcon={<NotificationsOutlined />}
            >
              {Messages.alerting}
            </Button>
            {showAdvisors && (
              <Button
                component={RouterLink}
                to={ADVISORS_URL}
                variant="outlined"
                startIcon={<NetworkIntelligenceIcon />}
              >
                {Messages.advisors}
              </Button>
            )}
          </Stack>
        </Stack>
        {renderContent()}
      </Stack>
    </Page>
  );
};
