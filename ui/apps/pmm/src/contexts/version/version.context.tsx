import { createContext } from 'react';
import { VersionContextProps } from './version.context.types';

export const VersionContext = createContext<VersionContextProps>({
  isOutdated: false,
  serverVersion: '',
  reload: () => {},
});
