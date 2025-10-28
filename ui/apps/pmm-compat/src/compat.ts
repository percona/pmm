import { locationService, getAppEvents, ThemeChangedEvent, config } from '@grafana/runtime';
import {
    ChangeThemeMessage,
    CrossFrameMessenger,
    DashboardVariablesMessage,
    HistoryAction,
    LocationChangeMessage,
} from '@pmm/shared';
import {
    GRAFANA_DOCKED_MENU_OPEN_LOCAL_STORAGE_KEY,
    GRAFANA_LOGIN_PATH,
    GRAFANA_SUB_PATH,
    PMM_UI_PATH,
} from 'lib/constants';
import { applyCustomStyles } from 'styles';
import { changeTheme } from 'theme';
import { adjustToolbar } from 'compat/toolbar';
import { isWithinIframe, getLinkWithVariables } from 'lib/utils';
import { documentTitleObserver } from 'lib/utils/document';

type ColorMode = 'light' | 'dark';

// Keep using the same string message types the file already uses elsewhere.
// If your shared package exports MessageType enum, swap the string with it:
// e.g. MessageType.GRAFANA_THEME_CHANGED / MessageType.CHANGE_THEME
const MSG_GRAFANA_THEME_CHANGED = 'GRAFANA_THEME_CHANGED' as const;

export const initialize = () => {
    if (!isWithinIframe() && !window.location.pathname.startsWith(GRAFANA_LOGIN_PATH)) {
        // redirect user to the new UI
        window.location.replace(window.location.href.replace(GRAFANA_SUB_PATH, PMM_UI_PATH));
        return;
    }

    // Register messenger towards PMM UI (top frame)
    const messenger = new CrossFrameMessenger('GRAFANA').setTargetWindow(window.top!).register();

    // React to PMM → Grafana theme changes
    messenger.addListener({
        type: 'CHANGE_THEME',
        onMessage: (msg: ChangeThemeMessage) => {
            if (!msg.payload) return;
            // Apply Grafana theme (handled on Grafana side)
            changeTheme(msg.payload.theme);
        },
    });

    messenger.sendMessage({ type: 'MESSENGER_READY' });

    // set docked state to false
    localStorage.setItem(GRAFANA_DOCKED_MENU_OPEN_LOCAL_STORAGE_KEY, 'false');

    applyCustomStyles();
    adjustToolbar();

    // -------- Theme relay: Grafana (right) → PMM UI (left) --------
    // Initial emit from Grafana current config
    const initial: ColorMode = config?.theme2?.colors?.mode === 'dark' ? 'dark' : 'light';
    messenger.sendMessage({
        type: MSG_GRAFANA_THEME_CHANGED,
        payload: { theme: initial },
    });

    // Forward future Grafana ThemeChangedEvent to PMM UI
    getAppEvents().subscribe(ThemeChangedEvent, (evt: unknown) => {
        // Grafana 11 emits ThemeChangedEvent with different shapes; normalize robustly.
        // Try known fields, then fall back to 'light'.
        const raw =
            // @ts-expect-error — best-effort probing of possible event shapes
            (evt?.payload?.colors?.mode as string | undefined) ??
            // @ts-expect-error — some places provide { theme: 'dark'|'light' }
            (evt?.theme as string | undefined) ??
            // @ts-expect-error — older shape: isDark boolean
            ((evt?.payload?.isDark as boolean | undefined) ? 'dark' : undefined);

        const next: ColorMode = raw?.toLowerCase() === 'dark' ? 'dark' : 'light';

        messenger.sendMessage({
            type: MSG_GRAFANA_THEME_CHANGED,
            payload: { theme: next },
        });
    });
    // --------------------------------------------------------------

    messenger.sendMessage({ type: 'GRAFANA_READY' });

    messenger.addListener({
        type: 'LOCATION_CHANGE',
        onMessage: ({ payload: location }: LocationChangeMessage) => {
            if (!location) return;
            locationService.replace(location);
        },
    });

    messenger.sendMessage({
        type: 'DOCUMENT_TITLE_CHANGE',
        payload: { title: document.title },
    });

    documentTitleObserver.listen((title) => {
        messenger.sendMessage({
            type: 'DOCUMENT_TITLE_CHANGE',
            payload: { title },
        });
    });

    let prevLocation: Location | undefined;
    locationService.getHistory().listen((location: Location, action: HistoryAction) => {
        // re-add custom toolbar buttons after closing kiosk mode
        if (prevLocation?.search.includes('kiosk') && !location.search.includes('kiosk')) {
            adjustToolbar();
        }

        messenger.sendMessage({
            type: 'LOCATION_CHANGE',
            payload: {
                action,
                ...location,
            },
        });

        prevLocation = location;
    });

    messenger.addListener({
        type: 'DASHBOARD_VARIABLES',
        onMessage: (msg: DashboardVariablesMessage) => {
            if (!msg.payload || !msg.payload.url) return;

            const url = getLinkWithVariables(msg.payload.url);

            messenger.sendMessage({
                id: msg.id,
                type: msg.type,
                payload: { url },
            });
        },
    });
};
