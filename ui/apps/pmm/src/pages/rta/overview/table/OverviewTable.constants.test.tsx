import { describe, it, expect } from 'vitest';
import { ServiceType } from 'types/services.types';
import { getOverviewTableColumns } from './OverviewTable.constants';
import { Messages } from './OverviewTable.messages';

describe('getOverviewTableColumns', () => {
  it.each([ServiceType.mysql, ServiceType.mongodb])(
    'keeps the pinned elapsed time header wide enough to read (%s)',
    (serviceType) => {
      const elapsed = getOverviewTableColumns(serviceType).find(
        (column) => column.header === Messages.columns.elapsedTime
      );

      // Narrower truncates the label beside its icons; minSize is the floor MRT enforces.
      expect(elapsed?.minSize).toBeGreaterThanOrEqual(160);
      expect(elapsed?.grow).toBe(false);
    }
  );
});
