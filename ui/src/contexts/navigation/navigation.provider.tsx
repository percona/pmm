import { FC, PropsWithChildren, useState } from 'react';
import { NavigationContext } from './navigation.context';
import { initialNavtree } from './navigation.contants';

export const NavigationProvider: FC<PropsWithChildren> = ({ children }) => {
  const [navTree, setNavTree] = useState(initialNavtree);

  return (
    <NavigationContext.Provider
      value={{
        navTree,
        setNavTree,
      }}
    >
      {children}
    </NavigationContext.Provider>
  );
};
