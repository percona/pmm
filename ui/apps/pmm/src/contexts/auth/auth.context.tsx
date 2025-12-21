import { createContext } from 'react';
import { AuthContextProps } from './auth.context.types';

export const AuthContext = createContext<AuthContextProps>({
  isLoading: false,
  isLoggedIn: false,
});
