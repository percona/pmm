import type { FC } from 'react';
import type {
  AlertThresholdRow,
  AlertThresholdsFormValues,
} from '../AlertThresholds.types';
import IconButton from '@mui/material/IconButton';
import RestartAltIcon from '@mui/icons-material/RestartAlt';
import { useFormContext } from 'react-hook-form';
import { Messages } from '../AlertThresholds.messages';

interface Props {
  row: AlertThresholdRow;
}

const ResetValueCell: FC<Props> = ({ row }) => {
  const { setValue } = useFormContext<AlertThresholdsFormValues>();

  return (
    <IconButton
      aria-label={Messages.actions.reset}
      onClick={() => setValue(row.id, row.defaultValue)}
    >
      <RestartAltIcon />
    </IconButton>
  );
};

export default ResetValueCell;
