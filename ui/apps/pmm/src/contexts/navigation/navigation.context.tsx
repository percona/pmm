import { createContext } from 'react';
import type { NavigationContextProps } from './navigation.context.types';

export const NavigationContext = createContext<NavigationContextProps>({
  navTree: [],
  navOpen: false,
  setNavOpen: () => {},
});
