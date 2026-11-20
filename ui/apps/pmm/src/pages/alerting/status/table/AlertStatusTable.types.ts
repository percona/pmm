import { AlertRow, AlertsTableRow } from '../AlertsPage.types';

export interface AlertStatusTableProps {
  rows: AlertRow[];
  onNavigableRowsChange: (rows: AlertsTableRow[]) => void;
  onOpenDetail: (row: AlertRow) => void;
}
