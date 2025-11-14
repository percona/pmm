import { LinkProps } from '@mui/material/Link';

export interface ClientsModalProps {
  homeLinkProps: LinkProps;
  isOpen: boolean;
  onClose: () => void;
}
