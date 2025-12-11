import { RealTimeQuery } from 'types/real-time.types';

export interface RealTimeTableProps {
  queries: RealTimeQuery[];
  showFilters: boolean;
  setShowFilters: (showFilters: boolean) => void;
  selectedQuery: RealTimeQuery | null;
  setQuery: (query: RealTimeQuery, index: number) => void;
}
