import { AutocompleteRenderInputParams } from '@mui/material/Autocomplete';
import { RealtimeSession } from 'types/rta.types';
import { ManagedService } from 'types/services.types';

export type TagPresentation = 'label' | 'tags';

interface BaseProps {
  tagPresentation?: TagPresentation;
  disabled?: boolean;
  serviceIds: string[];
  onServiceIdsChange: (serviceIds: string[]) => void;
  inputProps?: Partial<AutocompleteRenderInputParams>;
  'data-testid'?: string;
}

type PropsWithSessions = BaseProps & {
  sessions: RealtimeSession[];
};

type PropsWithServices = BaseProps & {
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
