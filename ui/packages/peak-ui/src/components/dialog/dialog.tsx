import { LinearProgress, Dialog as MatDialog } from '@mui/material';
import { DialogProps } from './dialog.types';

const Dialog = ({ loading, children, ...props }: DialogProps) => {
  return (
    <MatDialog {...props}>
      {loading && <LinearProgress></LinearProgress>}
      {children}
    </MatDialog>
  );
};

export default Dialog;
