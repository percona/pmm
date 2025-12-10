import { FC, useState } from 'react';
import { RealTimeFilters } from './filters';
import { RealTimeTable } from './table';
import Stack from '@mui/material/Stack';

const RealTimeLivePage: FC = () => {
  const [showFilters, setShowFilters] = useState(false);

  return (
    <Stack>
      <RealTimeFilters
        showFilters={showFilters}
        setShowFilters={setShowFilters}
      />
      <RealTimeTable
        showFilters={showFilters}
        setShowFilters={setShowFilters}
      />
    </Stack>
  );
};

export default RealTimeLivePage;
