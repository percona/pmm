import { AutocompleteRenderInputParams } from '@mui/material/Autocomplete';
import { ManagedService } from 'types/services.types';

export interface ServicesAutocompleteInputProps {
  disabled?: boolean;
  serviceIds: string[];
  onServiceIdsChange: (serviceIds: string[]) => void;
  services: ManagedService[];

  inputProps?: Partial<AutocompleteRenderInputParams>;
}

export interface ServiceOption {
  type: 'cluster' | 'service';
  id: string;
  label: string;
  serviceId?: string;
  cluster?: string;
}

export type ClusterSelectionState = 'all' | 'partial' | 'none';
