import { VersionsFilter } from '../UpdateClients.types';

export interface ClientsFilterProps {
  value: VersionsFilter;
  onChange: (value: VersionsFilter) => void;
}
