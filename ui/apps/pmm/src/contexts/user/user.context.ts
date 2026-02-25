import { createContext } from 'react';
import { UserContextProps } from './user.context.types';

export const UserContext = createContext<UserContextProps>({
  isLoading: false,
});
