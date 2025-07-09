import { ModalProps as MuiModalProps } from '@mui/material';

export interface ModalProps extends MuiModalProps {
  title: string;
  subtitle?: string;
}
