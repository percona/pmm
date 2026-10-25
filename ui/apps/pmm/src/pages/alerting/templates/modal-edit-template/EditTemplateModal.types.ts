import { Template } from 'types/alert-templates.types';

export interface EditTemplateModalProps {
  open: boolean;
  template: Template | null;
  onClose: () => void;
}
