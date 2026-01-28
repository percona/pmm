import { LinkProps } from '@mui/material';

export interface ClientsModalProps {
  homeLinkProps?: LinkProps;
  isOpen: boolean;
  onClose: () => void;
}
