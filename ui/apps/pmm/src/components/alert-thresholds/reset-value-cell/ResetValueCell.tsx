import { FC } from 'react';
import {
  AlertThresholdRow,
  AlertThresholdsFormValues,
} from '../AlertThresholds.types';
import IconButton from '@mui/material/IconButton';
import RestartAltIcon from '@mui/icons-material/RestartAlt';
import { useFormContext } from 'react-hook-form';

interface Props {
  row: AlertThresholdRow;
}

const ResetValueCell: FC<Props> = ({ row }) => {
  const { setValue } = useFormContext<AlertThresholdsFormValues>();

  return (
    <IconButton onClick={() => setValue(row.id, row.defaultValue)}>
      <RestartAltIcon />
    </IconButton>
  );
};

export default ResetValueCell;
