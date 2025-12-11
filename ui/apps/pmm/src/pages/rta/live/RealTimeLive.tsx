import { FC, useState } from 'react';
import { RealTimeFilters } from './filters';
import { RealTimeTable } from './table';
import Stack from '@mui/material/Stack';
import { DetailsPane } from './details-pane';
import { RealTimeQuery } from 'types/real-time.types';
import { REAL_TIME_TABLE_MOCK_DATA } from './table/RealTimeTable.constants';
import { useTheme } from '@mui/material/styles';

const RealTimeLivePage: FC = () => {
  const [showFilters, setShowFilters] = useState(false);
  const [showFullDetails, setShowFullDetails] = useState(false);
  const queries = REAL_TIME_TABLE_MOCK_DATA;
  const [selectedQueryIndex, setSelectedQueryIndex] = useState<number | null>(
    0
  );
  const selectedQuery =
    selectedQueryIndex !== null ? queries[selectedQueryIndex] : null;
  const theme = useTheme();
  const transition = theme.transitions.create(
    ['flex-grow', 'flex-basis', 'min-height', 'opacity'],
    {
      duration: theme.transitions.duration.complex,
      easing: theme.transitions.easing.easeInOut,
    }
  );

  const handleQueryChange = (query: RealTimeQuery | null, index: number) => {
    if (selectedQuery === query) {
      setSelectedQueryIndex(null);
      setShowFullDetails(false);
    } else {
      setSelectedQueryIndex(index);
    }
  };

  const handleCloseDetails = () => {
    setSelectedQueryIndex(null);
    setShowFullDetails(false);
  };

  return (
    <Stack
      sx={{
        px: 2,
        pt: 2,
        flex: 1,
        overflowY: 'hidden',
      }}
    >
      <RealTimeFilters
        showFilters={showFilters}
        setShowFilters={setShowFilters}
      />
      <Stack
        sx={[
          {
            mb: 2,
            flexGrow: 1,
            transition,
            overflow: 'hidden',
          },
          showFullDetails && {
            mb: 0,
            flexGrow: 0,
            minHeight: 0,
            opacity: 0,
            pointerEvents: 'none',
          },
        ]}
      >
        <RealTimeTable
          queries={queries}
          showFilters={showFilters}
          setShowFilters={setShowFilters}
          selectedQuery={selectedQuery}
          setQuery={handleQueryChange}
        />
      </Stack>
      <Stack
        sx={[
          {
            transition,
          },
          selectedQuery
            ? {
                minHeight: 350,
                flexGrow: showFullDetails ? 3 : 0,
                flexBasis: 350,
              }
            : {
                opacity: 0,
                minHeight: 0,
                flexGrow: 0,
                flexBasis: 0,
              },
        ]}
      >
        <DetailsPane
          query={selectedQuery}
          expanded={showFullDetails}
          onClose={handleCloseDetails}
          onExpand={() => setShowFullDetails(true)}
          onCollapse={() => setShowFullDetails(false)}
          onNext={() => setSelectedQueryIndex((prev) => (prev || 0) + 1)}
          onPrevious={() => setSelectedQueryIndex((prev) => (prev || 0) - 1)}
        />
      </Stack>
    </Stack>
  );
};

export default RealTimeLivePage;
