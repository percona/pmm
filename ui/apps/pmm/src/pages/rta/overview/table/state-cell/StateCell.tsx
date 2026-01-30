import Chip from '@mui/material/Chip';
import { FC, useMemo } from 'react';

export interface Props {
  state: string;
}

// TODO: WIP since the states are not defined yet
const StateCell: FC<Props> = ({ state }) => {
  const color = useMemo(() => {
    const normalizedState = state.toLowerCase();

    if (normalizedState === 'blocked') {
      return 'error';
    }

    if (normalizedState === 'running') {
      return 'info';
    }

    if (normalizedState === 'processing') {
      return 'warning';
    }

    return 'default';
  }, [state]);

  return <Chip color={color} label={state} variant="outlined" />;
};

export default StateCell;
