import { ColorModeContext } from '@percona/design';
import { useContext } from 'react';
import { useUpdatePreferences } from './api/useUser';

export const useColorMode = () => {
  const { colorMode, toggleColorMode } = useContext(ColorModeContext);
  const { mutate } = useUpdatePreferences();

  const toggleMode = () => {
    toggleColorMode();
    mutate({
      theme: colorMode === 'light' ? 'dark' : 'light',
    });
  };

  return { colorMode, toggleColorMode: toggleMode };
};
