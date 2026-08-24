import { createContext } from 'react';
import type { UserContextProps } from './user.context.types';

export const UserContext = createContext<UserContextProps>({
  isLoading: false,
});
