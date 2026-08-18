import { Template } from 'types/alert-templates.types';

export interface ViewTemplateModalProps {
  open: boolean;
  template: Template | null;
  onClose: () => void;
}
