import Typography from '@mui/material/Typography';
import { FC, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { RealtimePage } from '../components/rta-page';
import { useRealtimeQueries } from 'hooks/api/useRealtime';
import OverviewTable from './table/OverviewTable';
import { sampleQueries } from './table/OverviewTable.constants';
import { DetailsPane } from './details-pane';
import { QueryData } from 'types/rta.types';
import { Stack } from '@mui/material';

const RealtimeOverviewPage: FC = () => {
  const [searchParams] = useSearchParams();
  const serviceIds = searchParams.getAll('serviceIds');
  const {} = useRealtimeQueries(
    { serviceIds },
    {
      enabled: serviceIds.length > 0,
      // refetchInterval: 5000,
    }
  );
  const queries = sampleQueries;
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

  return (
    <RealtimePage>
      <Typography variant="body2">
        Service IDs: [{serviceIds.join(', ') || 'N/A'}]
      </Typography>
      <Stack
        sx={{
          position: 'relative',
          minHeight: 0,
        }}
      >
        <OverviewTable
          queries={queries || []}
          onQuerySelected={handleQueryChange}
        />
        <DetailsPane
          query={selectedQuery}
          onClose={handleCloseDetails}
          isFirstQuery={selectedQueryIndex === 0}
          isLastQuery={selectedQueryIndex === queries.length - 1}
          onNext={() => setSelectedQueryIndex((prev) => (prev || 0) + 1)}
          onPrevious={() => setSelectedQueryIndex((prev) => (prev || 0) - 1)}
        />
      </Stack>
    </RealtimePage>
  );
};

export default RealtimeOverviewPage;
