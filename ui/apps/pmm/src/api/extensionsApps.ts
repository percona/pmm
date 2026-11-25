import axios from 'axios';
import {
  EXTENSIONS_BASE_PATH,
  type EnabledApp,
  type MintedToken,
} from '@pmm-extensions/api';
import { getExtensionsToken } from 'extensions/extensionsTokenStore';

/**
 * A bare client for the side-car, deliberately not `@pmm-extensions/api`'s
 * `apiClient`. That client's interceptors mint through the token store, which
 * starts its renewal timer, and mark the session signed out on a 401. A list
 * fetched for the navigation must do neither: the renewal belongs to
 * `ExtensionsAuthGate`, and a side-car that is down must not leave every
 * Extensions page saying "signed out".
 */
const sidecar = axios.create({
  baseURL: `${EXTENSIONS_BASE_PATH}/api`,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
});

/**
 * Exchange the PMM session cookie for a side-car bearer used for this one
 * request. It is not recorded in the token store, so it schedules no renewal.
 */
const exchangeSession = async (signal?: AbortSignal): Promise<string> => {
  const { data } = await sidecar.post<MintedToken>(
    '/oauth/session/exchange',
    undefined,
    { signal }
  );
  return data.access_token;
};

/**
 * Every app the side-car serves, with its navigation metadata. Reuses the
 * bearer an open Extensions page holds, and exchanges the session only when
 * there is none. `signal` aborts whichever of the two requests is in flight.
 */
export const getExtensionsApps = async (
  signal?: AbortSignal
): Promise<EnabledApp[]> => {
  const token = getExtensionsToken() ?? (await exchangeSession(signal));
  const { data } = await sidecar.get<EnabledApp[]>('/apps/', {
    headers: { Authorization: `Bearer ${token}` },
    signal,
  });
  return data;
};
