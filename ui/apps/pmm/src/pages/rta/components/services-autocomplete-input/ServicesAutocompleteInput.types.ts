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
  // Restrict the selection to one database technology. Set where the picker
  // drives a single view of live queries; starting sessions has no such limit.
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
  // For a cluster option this is the technology shared by its services, and is
  // left unset if they somehow disagree.
  serviceType?: ServiceType;
  // The group this option is listed under. It cannot be derived from
  // serviceType: a cluster spanning technologies is listed once under each of
  // them while carrying no serviceType of its own, so deriving the group would
  // put those headers in a nameless group of their own and split the run of
  // options the list is sorted into.
  technology: string;
}

export type ClusterSelectionState = 'all' | 'partial' | 'none';
