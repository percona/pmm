import { useContext } from 'react';
import { UpdatesContext } from './updates.context';

export const useUpdates = () => useContext(UpdatesContext);
