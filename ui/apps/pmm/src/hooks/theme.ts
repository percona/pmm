import { ColorModeContext } from '@percona/design';
import { useContext } from 'react';
import { useUpdatePreferences } from './api/useUser';
import messenger from 'lib/messenger';
import type { MessageType } from '@pmm/shared';

export const useColorMode = () => {
    const { colorMode, toggleColorMode } = useContext(ColorModeContext);
    const { mutate } = useUpdatePreferences();

    const onToggle = () => {
        const next = colorMode === 'light' ? 'dark' : 'light';

        // 1) local apply (left UI)
        toggleColorMode();

        // 2) tell Grafana iframe to switch immediately
        messenger.sendMessage({
            type: 'CHANGE_THEME' as MessageType,
            payload: { theme: next },
        });

        // 3) persist in Grafana Preferences
        mutate({ theme: next });
    };

    return { colorMode, toggleColorMode: onToggle };
};