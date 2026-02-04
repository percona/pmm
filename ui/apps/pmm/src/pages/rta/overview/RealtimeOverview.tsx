import { FC, useState } from 'react';
import { Link as RouterLink, useSearchParams } from 'react-router-dom';
import { RealtimePage } from '../components/rta-page';
import { useRealtimeQueries, useRealtimeSessions } from 'hooks/api/useRealtime';
import OverviewTable from './table/OverviewTable';
import { DetailsPane } from './details-pane';
import { QueryData } from 'types/rta.types';
import { Icon } from 'components/icon';
import { Messages } from './RealtimeOverview.messages';
import { createRealtimeSessionsUrl } from 'utils/link.utils';
import Stack from '@mui/material/Stack';
import Button from '@mui/material/Button';
import { ServicesAutocompleteInput } from '../components/services-autocomplete-input';
import { FetchingIndicator } from './fetching-indicator';

const RealtimeOverviewPage: FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const serviceIds = searchParams.getAll('serviceIds');
  const [fetching, setFetching] = useState(serviceIds.length > 0);
  const { data: queries = [] } = useRealtimeQueries(
    { serviceIds },
    {
      enabled: fetching,
      refetchInterval: 5000,
    }
  );
  const [selectedQueryIndex, setSelectedQueryIndex] = useState<number>();
  const [selectedQuery, setSelectedQuery] = useState<QueryData>();
  const { data: sessions = [] } = useRealtimeSessions();

  const handleQueryChange = (query: QueryData, index: number) => {
    setSelectedQuery(query);
    setSelectedQueryIndex(index);
    setFetching(false);
  };

  const handleCloseDetails = () => {
    setSelectedQuery(undefined);
    setSelectedQueryIndex(undefined);
    setFetching(true);
  };

  const handleNextQuery = () => {
    const idx = (selectedQueryIndex || 0) + 1;
    if (idx >= queries.length) {
      return;
    }
    handleQueryChange(queries[idx], idx);
  };

  const handlePreviousQuery = () => {
    const idx = (selectedQueryIndex || 0) - 1;
    if (idx < 0) {
      return;
    }
    handleQueryChange(queries[idx], idx);
  };

  const handleServiceIdsChange = (newServiceIds: string[]) => {
    // start fetching if previous state was empty
    if (serviceIds.length === 0 && newServiceIds.length > 0) {
      setFetching(true);
    } else {
      setFetching((fetching) => {
        // if not fetching, don't start fetching
        if (!fetching) {
          return false;
        }

        return newServiceIds.length !== 0;
      });
    }

    setSearchParams({ serviceIds: newServiceIds });
  };

  return (
    <RealtimePage>
      <OverviewTable
        queries={queries || []}
        onQuerySelected={handleQueryChange}
        actions={() => (
          <Stack
            direction="row"
            alignItems="center"
            justifyContent="space-between"
            sx={{
              pl: 2,
              flex: 1,
            }}
          >
            <Stack gap={2} direction="row" alignItems="center">
              <Stack sx={{ minWidth: 360 }}>
                <ServicesAutocompleteInput
                  data-testid="overview-table-services-autocomplete-input"
                  sessions={sessions}
                  serviceIds={serviceIds}
                  onServiceIdsChange={handleServiceIdsChange}
                  inputProps={{
                    size: 'small',
                  }}
                />
              </Stack>
              <FetchingIndicator isFetching={fetching} />
              <Button
                data-testid={
                  fetching
                    ? 'overview-table-pause-button'
                    : 'overview-table-resume-button'
                }
                size="small"
                startIcon={
                  fetching ? <Icon name="pause" /> : <Icon name="play-arrow" />
                }
                disabled={serviceIds.length === 0}
                color={fetching ? 'inherit' : undefined}
                variant={fetching ? 'text' : 'contained'}
                onClick={() => setFetching(!fetching)}
                disableElevation
                sx={{
                  width: 100,
                  height: 32,
                }}
              >
                {fetching ? Messages.pause : Messages.resume}
              </Button>
            </Stack>
            <Button
              color="inherit"
              data-testid="overview-table-all-sessions-button"
              startIcon={<Icon name="dynamic-feed" />}
              component={RouterLink}
              to={createRealtimeSessionsUrl(serviceIds)}
            >
              {Messages.allSessions}
            </Button>
          </Stack>
        )}
      />
      <DetailsPane
        query={selectedQuery}
        onClose={handleCloseDetails}
        isFirstQuery={selectedQueryIndex === 0}
        isLastQuery={selectedQueryIndex === queries.length - 1}
        onNext={handleNextQuery}
        onPrevious={handlePreviousQuery}
      />
    </RealtimePage>
  );
};

export default RealtimeOverviewPage;
