import Chip from '@mui/material/Chip';
import { FC, useMemo } from 'react';

export interface Props {
  state: string;
}

// TODO: WIP since the states are not defined yet
const StateCell: FC<Props> = ({ state }) => {
  const color = useMemo(() => {
    switch (state.toLowerCase()) {
      case 'blocked':
        return 'error';
      case 'running':
        return 'info';
      case 'processing':
        return 'warning';
      default:
        return 'default';
    }
  }, [state]);

  return <Chip color={color} label={state} variant="outlined" />;
};

export default StateCell;
