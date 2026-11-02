import { useSyncExternalStore } from 'react';
import { safeLocalStorage } from 'utils/storage.utils';

export const SEARCH_EXPERIMENT_KEY = 'pmm-ui.experiments.search';

export const SEARCH_EXPERIMENT_PARAM = 'search';

export type SearchExperimentValue = 'on' | 'off';

// Vitest runs with search off so existing suites keep checking today's UI.
export const SEARCH_EXPERIMENT_FALLBACK: SearchExperimentValue =
  import.meta.env.MODE === 'test' ? 'off' : 'on';

const isExperimentValue = (
  value: string | null
): value is SearchExperimentValue => value === 'on' || value === 'off';

export const resolveSearchExperiment = ({
  param,
  stored,
  fallback,
}: {
  param: string | null;
  stored: string | null;
  fallback: SearchExperimentValue;
}): {
  value: SearchExperimentValue;
  persist: SearchExperimentValue | null;
  stripParam: boolean;
} => {
  if (isExperimentValue(param)) {
    return { value: param, persist: param, stripParam: true };
  }

  return {
    value: isExperimentValue(stored) ? stored : fallback,
    persist: null,
    stripParam: false,
  };
};

const decodeParamKey = (segment: string) => {
  const key = segment.split('=')[0].replace(/\+/g, ' ');

  try {
    return decodeURIComponent(key);
  } catch {
    return key;
  }
};

// Drops the switch's own segments and leaves every other one byte for byte.
const stripSearchParam = (search: string, value: SearchExperimentValue) => {
  const kept = search
    .replace(/^\?/, '')
    .split('&')
    .filter(
      (segment) =>
        segment !== '' &&
        !(
          decodeParamKey(segment) === SEARCH_EXPERIMENT_PARAM &&
          segment.slice(segment.indexOf('=') + 1) === value
        )
    );

  return kept.length ? `?${kept.join('&')}` : '';
};

export const applySearchExperimentParam = (
  loc: Location | undefined = typeof window === 'undefined'
    ? undefined
    : window.location,
  hist: History | undefined = typeof window === 'undefined'
    ? undefined
    : window.history
): SearchExperimentValue => {
  const param = loc
    ? new URLSearchParams(loc.search).get(SEARCH_EXPERIMENT_PARAM)
    : null;
  const { value, persist, stripParam } = resolveSearchExperiment({
    param,
    stored: safeLocalStorage.getItem(SEARCH_EXPERIMENT_KEY),
    fallback: SEARCH_EXPERIMENT_FALLBACK,
  });

  if (persist) {
    safeLocalStorage.setItem(SEARCH_EXPERIMENT_KEY, persist);
  }

  if (stripParam && loc && hist) {
    try {
      hist.replaceState(
        hist.state,
        '',
        loc.pathname + stripSearchParam(loc.search, value) + loc.hash
      );
    } catch {
      // The address keeps the param; the value still holds for this session.
    }
  }

  return value;
};

let current: SearchExperimentValue = applySearchExperimentParam();

const listeners = new Set<() => void>();

export const isSearchExperimentEnabled = () => current === 'on';

export const setSearchExperiment = (enabled: boolean) => {
  current = enabled ? 'on' : 'off';
  safeLocalStorage.setItem(SEARCH_EXPERIMENT_KEY, current);
  listeners.forEach((listener) => listener());
};

export const subscribeSearchExperiment = (listener: () => void) => {
  listeners.add(listener);

  return () => {
    listeners.delete(listener);
  };
};

export const useSearchExperiment = () =>
  useSyncExternalStore(
    subscribeSearchExperiment,
    isSearchExperimentEnabled,
    isSearchExperimentEnabled
  );
