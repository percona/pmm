export interface ServiceOption {
  type: 'cluster' | 'service';
  id: string;
  label: string;
  serviceId?: string;
  cluster?: string;
}

export type ClusterSelectionState = 'all' | 'partial' | 'none';
