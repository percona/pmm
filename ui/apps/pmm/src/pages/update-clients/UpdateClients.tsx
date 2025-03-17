import {
  Alert,
  Button,
  Card,
  CardContent,
  Chip,
  Link,
  Stack,
  Typography,
} from '@mui/material';
import { FetchingIcon } from 'components/fetching-icon';
import OpenInNew from '@mui/icons-material/OpenInNew';
import { Page } from 'components/page';
import { useUpdates } from 'contexts/updates';
import { FC, useMemo, useState } from 'react';
import { useAgentVersions } from 'hooks/api/useAgents';
import { SeverityChip } from './severity-chip';
import { VersionsFilter } from './UpdateClients.types';
import { filterClients } from './UpdateClients.utils';
import { GetAgentVersionItem } from 'types/agent.types';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import { UpdateStatus } from 'types/updates.types';
import { PMM_DOCS_UPDATE_CLIENT_URL } from 'lib/constants';
import { Messages } from './UpdateClients.messages';
import { TextSelect } from 'components/text-select';
import { FILTER_OPTIONS } from './UpdateClients.constants';
import { Table } from '@percona/ui-lib';
import { type MRT_ColumnDef } from 'material-react-table';
import { HomeLink } from 'components/home-link';

export const UpdateClients: FC = () => {
  const {
    versionInfo,
    clients: data,
    status,
    areClientsUpToDate: isUpToDate,
  } = useUpdates();
  const { isRefetching, refetch } = useAgentVersions();
  const [filter, setFilter] = useState<VersionsFilter>(VersionsFilter.All);
  const clients = useMemo(
    () => filterClients(data || [], filter),
    [data, filter]
  );

  const columns: MRT_ColumnDef<GetAgentVersionItem>[] = useMemo(
    () => [
      {
        accessorKey: 'nodeName',
        header: Messages.table.node,
      },
      {
        accessorKey: 'agentId',
        header: Messages.table.client,
      },
      {
        accessorKey: 'version',
        header: Messages.table.version,
      },
      {
        accessorKey: 'severity',
        header: Messages.table.severity,
        Cell: ({ row }) => <SeverityChip severity={row.original.severity} />,
      },
    ],
    []
  );

  return (
    <Page title={Messages.pageTitle}>
      <Card>
        <CardContent>
          <Stack spacing={2}>
            {isUpToDate && (
              <Alert
                icon={<CheckCircleOutlineIcon />}
                data-testid="pmm-server-up-to-date-alert"
              >
                {Messages.title}
                {!!versionInfo?.latestNewsUrl && (
                  <>
                    {Messages.see}
                    <Link
                      color="inherit"
                      target="_blank"
                      rel="noopener noreferrer"
                      href={versionInfo?.latestNewsUrl}
                    >
                      {Messages.releaseNotes}
                    </Link>
                    {Messages.dot}
                  </>
                )}
                {Messages.notify}
              </Alert>
            )}
            <Stack direction="row" alignItems="center" spacing={1}>
              <Typography variant="h4">
                {Messages.pmmUpdate(versionInfo?.latest?.version)}
              </Typography>
              {status === UpdateStatus.UpdateClients && (
                <Chip label={Messages.inProgress} color="warning" />
              )}
            </Stack>
            <Typography variant="h5">{Messages.step}</Typography>
            <Typography>{Messages.stepDescription}</Typography>
            <Stack
              direction="row"
              justifyContent="space-between"
              alignItems="center"
            >
              <Stack direction="row" spacing={1} alignItems="center">
                {!isUpToDate && (
                  <Link
                    target="_blank"
                    rel="noopener noreferrer"
                    href={PMM_DOCS_UPDATE_CLIENT_URL}
                    data-testid="how-to-update-clients-link"
                  >
                    <Button variant="contained" endIcon={<OpenInNew />}>
                      {Messages.howToUpdate}
                    </Button>
                  </Link>
                )}
                <Button
                  startIcon={<FetchingIcon isFetching={isRefetching} />}
                  variant="outlined"
                  onClick={() => refetch()}
                  data-testid="refresh-list-button"
                >
                  {isRefetching ? Messages.refreshing : Messages.refreshList}
                </Button>

                {isUpToDate && (
                  <HomeLink data-testid="pmm-home-link">
                    <Button variant="contained">{Messages.home}</Button>
                  </HomeLink>
                )}
              </Stack>
              <TextSelect
                value={filter}
                label={Messages.filter.label}
                options={FILTER_OPTIONS}
                onChange={setFilter}
              />
            </Stack>
            <Table
              tableName="pmm-clients"
              noDataMessage={Messages.table.empty}
              columns={columns}
              data={clients || []}
              enableFilters={false}
              enableColumnActions={false}
              enableColumnOrdering={false}
              enableTopToolbar={false}
              enableColumnDragging={false}
            />
          </Stack>
        </CardContent>
      </Card>
    </Page>
  );
};
