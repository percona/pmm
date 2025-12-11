import { RealTimeQuery } from 'types/real-time.types';

export interface DetailsPaneProps {
  query: RealTimeQuery | null;
  expanded: boolean;

  onClose: () => void;
  onExpand: () => void;
  onCollapse: () => void;
  onNext: () => void;
  onPrevious: () => void;
}
