import { createContext } from 'react';
import { UpdatesContextProps } from './updates.context.types';
import { UpdateStatus } from 'types/updates.types';

export const UpdatesContext = createContext<UpdatesContextProps>({
  isLoading: false,
  inProgress: false,
  status: UpdateStatus.Pending,
  setStatus: () => {},
  recheck: () => {},
  areClientsUpToDate: false,
});
