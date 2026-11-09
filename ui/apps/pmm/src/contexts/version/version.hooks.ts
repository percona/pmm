import { useContext } from 'react';
import { VersionContext } from './version.context';

export const useVersion = () => useContext(VersionContext);
