import { createContext } from 'react';
import type { AuthContextProps } from './auth.context.types';

export const AuthContext = createContext<AuthContextProps>({
  isLoading: false,
  isLoggedIn: false,
});
