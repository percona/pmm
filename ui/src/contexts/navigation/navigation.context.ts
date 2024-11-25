import { createContext } from 'react';
import { NavigationContextProps } from './navigation.context.types';
import { initialNavtree } from './navigation.contants';

export const NavigationContext = createContext<NavigationContextProps>({
  navTree: initialNavtree,
  setNavTree: () => {},
});
