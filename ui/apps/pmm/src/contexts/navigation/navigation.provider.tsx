import { FC, PropsWithChildren, useState } from 'react';
import { NavigationContext } from './navigation.context';
import { INITIAL_ITEMS } from './navigation.constants';
import { NavItem } from 'lib/types';

export const NavigationProvider: FC<PropsWithChildren> = ({ children }) => {
  const [items] = useState<NavItem[]>(INITIAL_ITEMS);

  return (
    <NavigationContext.Provider
      value={{
        navTree: items,
      }}
    >
      {children}
    </NavigationContext.Provider>
  );
};
