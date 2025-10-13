import { createContext } from 'react';
import { NavigationContextProps } from './navigation.context.types';

export const NavigationContext = createContext<NavigationContextProps>({
  navTree: [],
});
