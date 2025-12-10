import Chip from '@mui/material/Chip';
import { FC } from 'react';
import { getStyles } from './StateCell.styles';
import { useTheme } from '@mui/material/styles';
import { StateCellProps } from './StateCell.types';

const StateCell: FC<StateCellProps> = ({ state }) => {
  const theme = useTheme();
  const styles = getStyles(theme, state);

  return <Chip color="default" label={state} variant="outlined" sx={styles} />;
};

export default StateCell;
