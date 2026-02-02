import { FC, useState } from 'react';
import { Link as RouterLink, useSearchParams } from 'react-router-dom';
import { RealtimePage } from '../components/rta-page';
import { useRealtimeQueries } from 'hooks/api/useRealtime';
import OverviewTable from './table/OverviewTable';
import { DetailsPane } from './details-pane';
import { QueryData } from 'types/rta.types';
import { Icon } from 'components/icon';
import { Messages } from './RealtimeOverview.messages';
import { createRealtimeSessionsUrl } from 'utils/link.utils';
import Stack from '@mui/material/Stack';
import Button from '@mui/material/Button';

const RealtimeOverviewPage: FC = () => {
  const [searchParams] = useSearchParams();
  const serviceIds = searchParams.getAll('serviceIds');
  const { data: queries = [] } = useRealtimeQueries(
    { serviceIds },
    {
      enabled: serviceIds.length > 0,
      refetchInterval: 5000,
    }
  );
  const [selectedQueryIndex, setSelectedQueryIndex] = useState<number>();
  const [selectedQuery, setSelectedQuery] = useState<QueryData>();

  const handleQueryChange = (query: QueryData, index: number) => {
    setSelectedQuery(query);
    setSelectedQueryIndex(index);
  };

  const handleCloseDetails = () => {
    setSelectedQuery(undefined);
    setSelectedQueryIndex(undefined);
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
              flex: 1,
            }}
          >
            <Stack>{/* Leaving space for the filters/pause/etc. */}</Stack>
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
