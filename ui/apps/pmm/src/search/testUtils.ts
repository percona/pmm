import * as Peak from '@percona/peak-ui';
import { setSearchExperiment } from './search.experiment';
import { resetSearchStore } from './search.store';

// The published Peak UI 1.0.26 has no search components; the peak-local Vitest config resolves the source that does.
export const hasPeakSearch =
  typeof (Peak as unknown as Record<string, unknown>).SearchDialog ===
  'function';

// Call at describe level: turns search on for each test and back off after it.
export const enableSearchForTest = () => {
  beforeEach(() => {
    setSearchExperiment(true);
  });

  afterEach(() => {
    setSearchExperiment(false);
    resetSearchStore();
  });
};
