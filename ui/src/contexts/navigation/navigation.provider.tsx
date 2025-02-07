import { FC, PropsWithChildren, useEffect, useState } from 'react';
import { NavigationContext } from './navigation.context';
import { initialNavtree } from './navigation.contants';
import { useDashboardFolders } from 'hooks/api/useFolders';
import { addFolderLinks } from './navigation.utils';
import { useMessages } from 'contexts/messages/messages.hooks';

export const NavigationProvider: FC<PropsWithChildren> = ({ children }) => {
  const [navTree, setNavTree] = useState(initialNavtree);
  const { data: folders } = useDashboardFolders();
  const messages = useMessages('STARRED_DASHBOARDS');

  console.log('messages', messages);

  useEffect(() => {
    if (folders) {
      addFolderLinks(navTree, folders);
    }
  }, [folders]);

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
