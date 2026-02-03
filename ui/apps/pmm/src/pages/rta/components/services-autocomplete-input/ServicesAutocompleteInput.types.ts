import { AutocompleteRenderInputParams } from '@mui/material/Autocomplete';
import { RealtimeSession } from 'types/rta.types';
import { ManagedService } from 'types/services.types';

interface BaseProps {
  disabled?: boolean;
  serviceIds: string[];
  onServiceIdsChange: (serviceIds: string[]) => void;
  inputProps?: Partial<AutocompleteRenderInputParams>;
}

export type PropsWithSessions = BaseProps & {
  sessions: RealtimeSession[];
};

export type PropsWithServices = BaseProps & {
  services: ManagedService[];
};

export type ServicesAutocompleteInputProps =
  | PropsWithSessions
  | PropsWithServices;

export interface ServiceOption {
  type: 'cluster' | 'service';
  id: string;
  label: string;
  serviceId?: string;
  cluster?: string;
}

export type ClusterSelectionState = 'all' | 'partial' | 'none';
