import Chip from '@mui/material/Chip';
import { FC } from 'react';
import { getStyles } from './StateCell.styles';
import { useTheme } from '@mui/material/styles';

export interface Props {
  state: string;
}

const StateCell: FC<Props> = ({ state }) => {
  const theme = useTheme();
  const styles = getStyles(theme, state);

  return <Chip color="default" label={state} variant="outlined" sx={styles} />;
};

export default StateCell;
