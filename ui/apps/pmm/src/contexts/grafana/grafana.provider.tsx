import {
  FC,
  PropsWithChildren,
  useEffect,
  useRef,
  useState,
  useRef as useRef2,
} from 'react';
import { useLocation, useNavigate, useNavigationType } from 'react-router';
import { GrafanaContext } from './grafana.context';
import {
  GRAFANA_SUB_PATH,
  PMM_NEW_NAV_GRAFANA_PATH,
  PMM_NEW_NAV_PATH,
} from 'lib/constants';
import { DocumentTitleUpdateMessage, LocationChangeMessage } from '@pmm/shared';
import messenger from 'lib/messenger';
import { getLocationUrl } from './grafana.utils';
import { updateDocumentTitle } from 'lib/utils/document.utils';
import { useKioskMode } from 'hooks/utils/useKioskMode';
import { useColorMode } from 'hooks/theme';
import { useSetTheme } from 'themes/setTheme';

type Mode = 'light' | 'dark';

/** Reads canonical mode from <html> attributes set by our theme hook. */
const readHtmlMode = (): Mode =>
  document.documentElement
    .getAttribute('data-md-color-scheme')
    ?.includes('dark')
    ? 'dark'
    : 'light';

/** Normalizes any incoming value to 'light' | 'dark'. */
const normalizeMode = (v: unknown): Mode =>
  typeof v === 'string' && v.toLowerCase() === 'dark'
    ? 'dark'
    : v === true
      ? 'dark'
      : 'light';

export const GrafanaProvider: FC<PropsWithChildren> = ({ children }) => {
  const navigationType = useNavigationType();
  const location = useLocation();
  const src = location.pathname.replace(PMM_NEW_NAV_PATH, '');
  const isGrafanaPage = src.startsWith(GRAFANA_SUB_PATH);

  const [isLoaded, setIsLoaded] = useState(false);
  const frameRef = useRef<HTMLIFrameElement>(null);
  const navigate = useNavigate();
  const kioskMode = useKioskMode();

  // Ensure our theme context is mounted (also mounts the global theme sync hook elsewhere)
  useColorMode();

  const { setFromGrafana } = useSetTheme();

  // Remember last theme we sent to avoid resending the same value.
  const lastSentThemeRef = useRef2<Mode>(readHtmlMode());

  // Mark iframe area as loaded when we hit /graph/*
  useEffect(() => {
    if (isGrafanaPage) setIsLoaded(true);
  }, [isGrafanaPage]);

  // Propagate location changes to Grafana (except POP from Grafana itself)
  useEffect(() => {
    if (
      !location.pathname.includes('/graph') ||
      (location.state?.fromGrafana && navigationType !== 'POP')
    ) {
      return;
    }

    messenger.sendMessage({
      type: 'LOCATION_CHANGE',
      payload: {
        ...location,
        pathname: location.pathname.replace(PMM_NEW_NAV_GRAFANA_PATH, ''),
        action: navigationType,
      },
    });
  }, [location, navigationType]);

  // Set up messenger and standard listeners once iframe area is ready
  useEffect(() => {
    if (!isLoaded) return;

    messenger
      .setTargetWindow(frameRef.current?.contentWindow!, '#grafana-iframe')
      .register();

    // Send the current canonical theme to Grafana once messenger is ready.
    messenger.waitForMessage('MESSENGER_READY').then(() => {
      const mode = readHtmlMode(); // take from <html>, already synced with Grafana Prefs
      if (lastSentThemeRef.current !== mode) {
        lastSentThemeRef.current = mode;
        messenger.sendMessage({
          type: 'CHANGE_THEME',
          payload: { theme: mode },
        });
      }
    });

    // Mirror Grafana → PMM route changes (except POP)
    messenger.addListener({
      type: 'LOCATION_CHANGE',
      onMessage: ({ payload: loc }: LocationChangeMessage) => {
        if (!loc || loc.action === 'POP') return;
        navigate(getLocationUrl(loc), {
          state: { fromGrafana: true },
          replace: true,
        });
      },
    });

    // Mirror Grafana document title
    messenger.addListener({
      type: 'DOCUMENT_TITLE_CHANGE',
      onMessage: ({ payload }: DocumentTitleUpdateMessage) => {
        if (!payload) return;
        updateDocumentTitle(payload.title);
      },
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoaded]);

  // If outer theme changes (our hook updates <html>), reflect it to Grafana quickly
  useEffect(() => {
    if (!isLoaded) return;
    const mode = readHtmlMode();
    if (lastSentThemeRef.current !== mode) {
      lastSentThemeRef.current = mode;
      messenger.sendMessage({ type: 'CHANGE_THEME', payload: { theme: mode } });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoaded, location]); // re-evaluate on navigation; inexpensive and safe

  // Hard guarantee: listen for grafana.theme.changed on /graph/* pages and apply locally (no persist/broadcast).
  useEffect(() => {
    if (!isLoaded) return;

    const onMsg = (e: MessageEvent) => {
      if (!e?.data || e.data.type !== 'grafana.theme.changed') return;
      const p = e.data?.payload ?? {};
      const raw = p.mode ?? p.payloadMode ?? (p.isDark ? 'dark' : 'light');
      const desired = normalizeMode(raw);

      // Apply locally only to avoid ping-pong; persistence is handled by left action.
      setFromGrafana(desired).catch((err) =>
        console.warn('[GrafanaProvider] setFromGrafana failed:', err)
      );
    };

    window.addEventListener('message', onMsg);
    return () => window.removeEventListener('message', onMsg);
  }, [isLoaded, setFromGrafana]);

  return (
    <GrafanaContext.Provider
      value={{
        frameRef,
        isFrameLoaded: isLoaded,
        isOnGrafanaPage: isGrafanaPage,
        isFullScreen: kioskMode.active,
      }}
    >
      {children}
    </GrafanaContext.Provider>
  );
};
