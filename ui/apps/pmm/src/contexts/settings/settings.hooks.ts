import { useContext } from 'react';
import { SettingsContext } from './settings.context';

export const useSettings = () => useContext(SettingsContext);
