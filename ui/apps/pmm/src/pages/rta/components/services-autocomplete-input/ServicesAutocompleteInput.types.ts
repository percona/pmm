import { AutocompleteRenderInputParams } from '@mui/material/Autocomplete';
import { AvailableService, RealtimeSession } from 'types/rta.types';
import { ServiceType } from 'types/services.types';

export type TagPresentation = 'label' | 'tags';

interface BaseProps {
  tagPresentation?: TagPresentation;
  disabled?: boolean;
  serviceIds: string[];
  onServiceIdsChange: (serviceIds: string[]) => void;
  inputProps?: Partial<AutocompleteRenderInputParams>;
  // Set where the picker drives a single view of live queries.
  singleTechnology?: boolean;
  'data-testid'?: string;
}

type PropsWithSessions = BaseProps & {
  sessions: RealtimeSession[];
};

type PropsWithServices = BaseProps & {
  services: AvailableService[];
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
  // Unset for a cluster whose services disagree on technology.
  serviceType?: ServiceType;
  // Not derived from serviceType: a mixed cluster has none, yet is listed under
  // each of its technologies.
  technology: string;
}

export type ClusterSelectionState = 'all' | 'partial' | 'none';
