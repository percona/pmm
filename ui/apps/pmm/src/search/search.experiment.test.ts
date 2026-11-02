import { act, renderHook } from '@testing-library/react';
import {
  SEARCH_EXPERIMENT_FALLBACK,
  SEARCH_EXPERIMENT_KEY,
  applySearchExperimentParam,
  isSearchExperimentEnabled,
  resolveSearchExperiment,
  setSearchExperiment,
  subscribeSearchExperiment,
  useSearchExperiment,
} from './search.experiment';

const fakeLocation = (search: string, hash = '') =>
  ({ pathname: '/pmm-ui/graph/alerting/list', search, hash }) as Location;

const fakeHistory = () =>
  ({ state: { idx: 3 }, replaceState: vi.fn() }) as unknown as History & {
    replaceState: ReturnType<typeof vi.fn>;
  };

describe('search.experiment', () => {
  beforeEach(() => {
    localStorage.removeItem(SEARCH_EXPERIMENT_KEY);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    setSearchExperiment(false);
  });

  it('is off under Vitest so existing suites keep today’s UI', () => {
    expect(SEARCH_EXPERIMENT_FALLBACK).toBe('off');
    expect(isSearchExperimentEnabled()).toBe(false);
  });

  describe('resolveSearchExperiment', () => {
    it.each([
      ['on', null, 'off', { value: 'on', persist: 'on', stripParam: true }],
      ['off', 'on', 'on', { value: 'off', persist: 'off', stripParam: true }],
      [null, 'on', 'off', { value: 'on', persist: null, stripParam: false }],
      [null, 'off', 'on', { value: 'off', persist: null, stripParam: false }],
      [null, null, 'on', { value: 'on', persist: null, stripParam: false }],
      [null, 'junk', 'off', { value: 'off', persist: null, stripParam: false }],
      [
        'state:firing',
        'on',
        'off',
        { value: 'on', persist: null, stripParam: false },
      ],
      ['ON', null, 'off', { value: 'off', persist: null, stripParam: false }],
    ] as const)(
      'param %s, stored %s, fallback %s',
      (param, stored, fallback, expected) => {
        expect(resolveSearchExperiment({ param, stored, fallback })).toEqual(
          expected
        );
      }
    );
  });

  describe('applySearchExperimentParam', () => {
    it('persists the param and drops only it, keeping other params, the hash and the history state', () => {
      const history = fakeHistory();

      expect(
        applySearchExperimentParam(
          fakeLocation('?a=1&search=on&var-x=a:b', '#panel'),
          history
        )
      ).toBe('on');
      expect(localStorage.getItem(SEARCH_EXPERIMENT_KEY)).toBe('on');
      expect(history.replaceState).toHaveBeenCalledWith(
        { idx: 3 },
        '',
        '/pmm-ui/graph/alerting/list?a=1&var-x=a:b#panel'
      );
    });

    it('leaves no question mark when the switch was the only param', () => {
      const history = fakeHistory();

      expect(
        applySearchExperimentParam(fakeLocation('?search=off'), history)
      ).toBe('off');
      expect(localStorage.getItem(SEARCH_EXPERIMENT_KEY)).toBe('off');
      expect(history.replaceState).toHaveBeenCalledWith(
        { idx: 3 },
        '',
        '/pmm-ui/graph/alerting/list'
      );
    });

    it('leaves Grafana’s own search param alone', () => {
      const history = fakeHistory();
      localStorage.setItem(SEARCH_EXPERIMENT_KEY, 'on');

      expect(
        applySearchExperimentParam(
          fakeLocation('?search=state:firing'),
          history
        )
      ).toBe('on');
      expect(history.replaceState).not.toHaveBeenCalled();
      expect(localStorage.getItem(SEARCH_EXPERIMENT_KEY)).toBe('on');
    });

    it('falls back to the stored value, then the default', () => {
      const history = fakeHistory();

      expect(applySearchExperimentParam(fakeLocation(''), history)).toBe('off');

      localStorage.setItem(SEARCH_EXPERIMENT_KEY, 'on');

      expect(applySearchExperimentParam(fakeLocation(''), history)).toBe('on');
      expect(history.replaceState).not.toHaveBeenCalled();
    });

    it('still returns the param value when storage throws', () => {
      vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
        throw new Error('blocked');
      });
      vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
        throw new Error('blocked');
      });
      const history = fakeHistory();

      expect(
        applySearchExperimentParam(fakeLocation('?search=on'), history)
      ).toBe('on');
      expect(history.replaceState).toHaveBeenCalled();
    });
  });

  it('persists a change and notifies subscribers', () => {
    const listener = vi.fn();
    const unsubscribe = subscribeSearchExperiment(listener);

    setSearchExperiment(true);

    expect(isSearchExperimentEnabled()).toBe(true);
    expect(localStorage.getItem(SEARCH_EXPERIMENT_KEY)).toBe('on');
    expect(listener).toHaveBeenCalledTimes(1);

    unsubscribe();
    setSearchExperiment(false);

    expect(listener).toHaveBeenCalledTimes(1);
  });

  it('re-renders hook users when it changes', () => {
    const { result } = renderHook(() => useSearchExperiment());

    expect(result.current).toBe(false);

    act(() => setSearchExperiment(true));

    expect(result.current).toBe(true);
  });
});
