import { useContext } from 'react';
import { UserContext } from './user.context';

export const useUser = () => useContext(UserContext);
