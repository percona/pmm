import type { MRT_ColumnDef, MRT_RowData } from 'material-react-table';

export type DiscoveredInstance = MRT_RowData & { instanceId: string };

export interface DiscoveredInstancesProps<T extends DiscoveredInstance> {
  tableName: string;
  columns: MRT_ColumnDef<T>[];
  instances: T[];
  isLoading: boolean;
  onSelect: (instance: T) => void;
}
