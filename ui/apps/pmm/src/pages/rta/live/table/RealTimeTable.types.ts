import { RealTimeQuery } from 'types/real-time.types';

export interface RealTimeTableProps {
  showFilters: boolean;
  setShowFilters: (showFilters: boolean) => void;
  selectedQuery: RealTimeQuery | null;
  setQuery: (query: RealTimeQuery) => void;
}
