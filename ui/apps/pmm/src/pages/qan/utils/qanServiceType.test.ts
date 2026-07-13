import { describe, expect, it } from 'vitest';
import { serviceTypeToQanDatabase } from './qanServiceType';

describe('serviceTypeToQanDatabase', () => {
  it('maps postgresql service type', () => {
    expect(serviceTypeToQanDatabase('postgresql')).toBe('postgresql');
  });

  it('maps mongodb service type', () => {
    expect(serviceTypeToQanDatabase('SERVICE_TYPE_MONGODB_SERVICE')).toBe('mongodb');
  });
});
