import { createContext } from 'react';
import type { SettingsContextProps } from './settings.context.types';

export const SettingsContext = createContext<SettingsContextProps>({
  isLoading: false,
  settings: null,
});
