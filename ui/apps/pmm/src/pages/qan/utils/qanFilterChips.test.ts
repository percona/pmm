import { clearAllFilters, getActiveFilterChips, removeFilterChip } from './qanFilterChips';

describe('qanFilterChips', () => {
  it('lists active filter chips excluding $__all', () => {
    const chips = getActiveFilterChips({
      service_name: ['$__all'],
      database: ['app', 'analytics'],
    });
    expect(chips).toHaveLength(2);
    expect(chips[0].label).toContain('database: app');
  });

  it('clears filters to $__all', () => {
    const next = clearAllFilters({ database: ['app'] });
    expect(next.database).toEqual(['$__all']);
  });

  it('removes a single chip value', () => {
    const next = removeFilterChip({ database: ['app', 'logs'] }, 'database', 'app');
    expect(next.database).toEqual(['logs']);
  });

  it('ignores interval label in chips', () => {
    const chips = getActiveFilterChips({ interval: ['60'], database: ['app'] });
    expect(chips).toHaveLength(1);
    expect(chips[0].key).toBe('database');
  });

  it('ignores spurious by label from legacy filter_by param', () => {
    const chips = getActiveFilterChips({ by: ['query-id'], service_id: ['svc-1'] });
    expect(chips).toHaveLength(1);
    expect(chips[0].key).toBe('service_id');
  });
});
