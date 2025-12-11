import Autocomplete from '@mui/material/Autocomplete';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import { FC, useState } from 'react';
import { Messages } from './RealTimeFilters.messages';
import IconButton from '@mui/material/IconButton';
import FilterListOutlinedIcon from '@mui/icons-material/FilterListOutlined';
import FilterListOffOutlinedIcon from '@mui/icons-material/FilterListOffOutlined';
import ElectricBoltOutlinedIcon from '@mui/icons-material/ElectricBoltOutlined';
import Button from '@mui/material/Button';
import PauseOutlinedIcon from '@mui/icons-material/PauseOutlined';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import StopIcon from '@mui/icons-material/Stop';
import { RealTimeFiltersProps } from './RealTimeFilters.types';

const RealTimeFilters: FC<RealTimeFiltersProps> = ({
  showFilters,
  setShowFilters,
}) => {
  const [fetching, setFetching] = useState(true);

  return (
    <Stack gap={2} direction="row" alignItems="center" mb={2}>
      <IconButton onClick={() => setShowFilters(!showFilters)}>
        {showFilters ? (
          <FilterListOffOutlinedIcon />
        ) : (
          <FilterListOutlinedIcon />
        )}
      </IconButton>
      <Autocomplete
        renderInput={(params) => (
          <TextField {...params} label={Messages.selectCluster} />
        )}
        options={[]}
        sx={{ width: 240 }}
      />
      <Autocomplete
        renderInput={(params) => (
          <TextField {...params} label={Messages.selectService} />
        )}
        options={[]}
        sx={{ width: 240 }}
      />
      <Button
        disableRipple
        component="div"
        color={fetching ? 'primary' : 'inherit'}
        startIcon={<ElectricBoltOutlinedIcon />}
        sx={{
          cursor: 'default',

          '&:hover': {
            backgroundColor: 'transparent',
          },
        }}
      >
        {fetching ? Messages.fetching : Messages.paused}
      </Button>
      <Button
        color={fetching ? 'inherit' : 'primary'}
        startIcon={fetching ? <PauseOutlinedIcon /> : <PlayArrowIcon />}
        onClick={() => setFetching(!fetching)}
      >
        {fetching ? Messages.pause : Messages.resume}
      </Button>
      <Button startIcon={<StopIcon />} color="inherit">
        {Messages.stopAgent}
      </Button>
    </Stack>
  );
};

export default RealTimeFilters;
