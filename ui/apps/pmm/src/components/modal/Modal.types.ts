import type { ModalProps as MuiModalProps } from '@mui/material/Modal/Modal';

export interface ModalProps extends MuiModalProps {
  title: string;
  subtitle?: string;
}
