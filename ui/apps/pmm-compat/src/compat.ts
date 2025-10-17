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

export const initialize = () => {
    if (!isWithinIframe() && !window.location.pathname.startsWith(GRAFANA_LOGIN_PATH)) {
        // redirect user to the new UI
        window.location.replace(window.location.href.replace(GRAFANA_SUB_PATH, PMM_UI_PATH));
        return;
    }

    const messenger = new CrossFrameMessenger('GRAFANA').setTargetWindow(window.top!).register();

    messenger.addListener({
        type: 'CHANGE_THEME',
        onMessage: (msg: ChangeThemeMessage) => {
            if (!msg.payload) {
                return;
            }
            changeTheme(msg.payload.theme);
        },
    });

    messenger.sendMessage({ type: 'MESSENGER_READY' });

    // set docked state to false
    localStorage.setItem(GRAFANA_DOCKED_MENU_OPEN_LOCAL_STORAGE_KEY, 'false');

    applyCustomStyles();
    adjustToolbar();

    // Wire Grafana theme events to Percona scheme attribute and notify parent
    setupThemeWiring();

    messenger.sendMessage({ type: 'GRAFANA_READY' });

    messenger.addListener({
        type: 'LOCATION_CHANGE',
        onMessage: ({ payload: location }: LocationChangeMessage) => {
            if (!location) {
                return;
            }
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
            if (!msg.payload || !msg.payload.url) {
                return;
            }

            const url = getLinkWithVariables(msg.payload.url);

            messenger.sendMessage({
                id: msg.id,
                type: msg.type,
                payload: { url },
            });
        },
    });
};

/**
 * Wires Grafana theme changes (ThemeChangedEvent) to Percona CSS scheme.
 * Ensures <html> has the attribute our CSS reads and informs the parent frame.
 */
// comments in English only
function setupThemeWiring() {
    // Resolve parent origin robustly (dev vs prod)
    const getParentOrigin = (): string => {
        try {
            const u = new URL(document.referrer);
            return `${u.protocol}//${u.host}`;
        } catch {
            return '*';
        }
    };

    const isDev =
        location.hostname === 'localhost' ||
        location.hostname === '127.0.0.1' ||
        /^\d+\.\d+\.\d+\.\d+$/.test(location.hostname);

    const targetOrigin = isDev ? '*' : getParentOrigin();

    // Helper to apply our scheme attribute and notify parent
    const apply = (incoming: 'light' | 'dark' | string) => {
        // normalize to 'light' | 'dark'
        const mode: 'light' | 'dark' = String(incoming).toLowerCase() === 'dark' ? 'dark' : 'light';

        const html = document.documentElement;
        const scheme = mode === 'dark' ? 'percona-dark' : 'percona-light';

        // Set attributes our CSS reads
        html.setAttribute('data-md-color-scheme', scheme);
        html.setAttribute('data-theme', mode);
        (html.style as any).colorScheme = mode;

        // Notify outer PMM UI (new nav) to sync immediately
        try {
            const target = window.top && window.top !== window ? window.top : window.parent || window;
            target?.postMessage({ type: 'grafana.theme.changed', payload: { mode } }, targetOrigin);
        } catch (err) {
            // eslint-disable-next-line no-console
            console.warn('[pmm-compat] failed to post grafana.theme.changed:', err);
        }
    };

    // Initial apply from current Grafana theme config
    const initialMode = (config?.theme2?.colors?.mode === 'dark' ? 'dark' : 'light') as 'light' | 'dark';
    apply(initialMode);

    // React to Grafana theme changes (Preferences change/changeTheme())
    getAppEvents().subscribe(ThemeChangedEvent, (evt: any) => {
        const next = evt?.payload?.colors?.mode ?? (evt?.payload?.isDark ? 'dark' : 'light') ?? 'light';
        apply(next);
    });
}
