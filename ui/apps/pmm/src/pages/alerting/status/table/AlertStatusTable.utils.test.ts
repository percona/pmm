import { type MRT_Row } from 'material-react-table';
import { AlertsTableRow } from '../AlertsPage.types';
import {
  filterTriggeredAt,
  reviveTriggeredAtFilter,
} from './AlertStatusTable.utils';

const createRow = (activeAt?: string) =>
  ({
    getValue: () => (activeAt ? new Date(activeAt) : undefined),
  }) as unknown as MRT_Row<AlertsTableRow>;

describe('filterTriggeredAt', () => {
  const row = createRow('2026-04-15T12:00:00.000Z');

  it('passes every row when both bounds are empty', () => {
    expect(filterTriggeredAt(row, 'activeAt', ['', ''])).toBe(true);
    expect(filterTriggeredAt(createRow(), 'activeAt', ['', ''])).toBe(true);
  });

  it('filters by both bounds inclusively', () => {
    expect(
      filterTriggeredAt(row, 'activeAt', [
        new Date('2026-04-15T11:00:00.000Z'),
        new Date('2026-04-15T13:00:00.000Z'),
      ])
    ).toBe(true);
    expect(
      filterTriggeredAt(row, 'activeAt', [
        new Date('2026-04-15T12:00:00.000Z'),
        new Date('2026-04-15T12:00:00.000Z'),
      ])
    ).toBe(true);
    expect(
      filterTriggeredAt(row, 'activeAt', [
        new Date('2026-04-15T12:30:00.000Z'),
        new Date('2026-04-15T13:00:00.000Z'),
      ])
    ).toBe(false);
  });

  it('supports an open-ended "to" bound', () => {
    expect(
      filterTriggeredAt(row, 'activeAt', [
        new Date('2026-04-15T11:00:00.000Z'),
        '',
      ])
    ).toBe(true);
    expect(
      filterTriggeredAt(row, 'activeAt', [
        new Date('2026-04-15T13:00:00.000Z'),
        '',
      ])
    ).toBe(false);
  });

  it('supports an open-ended "from" bound', () => {
    expect(
      filterTriggeredAt(row, 'activeAt', [
        '',
        new Date('2026-04-15T13:00:00.000Z'),
      ])
    ).toBe(true);
    expect(
      filterTriggeredAt(row, 'activeAt', [
        '',
        new Date('2026-04-15T11:00:00.000Z'),
      ])
    ).toBe(false);
  });

  it('excludes rows without a triggered-at value when a bound is set', () => {
    expect(
      filterTriggeredAt(createRow(), 'activeAt', [
        new Date('2026-04-15T11:00:00.000Z'),
        '',
      ])
    ).toBe(false);
  });

  it('treats invalid dates as unset bounds', () => {
    expect(filterTriggeredAt(row, 'activeAt', [new Date('invalid'), ''])).toBe(
      true
    );
  });
});

describe('reviveTriggeredAtFilter', () => {
  it('turns ISO bounds read from the URL back into dates', () => {
    expect(
      reviveTriggeredAtFilter([
        { id: 'activeAt', value: ['2026-04-15T12:00:00.000Z', ''] },
      ])
    ).toEqual([
      { id: 'activeAt', value: [new Date('2026-04-15T12:00:00.000Z'), ''] },
    ]);
  });

  it('empties a bound that is not a date', () => {
    expect(
      reviveTriggeredAtFilter([{ id: 'activeAt', value: ['soon', ''] }])
    ).toEqual([{ id: 'activeAt', value: ['', ''] }]);
  });

  it('leaves other filters and Date bounds untouched', () => {
    const from = new Date('2026-04-15T12:00:00.000Z');
    const filters = [
      { id: 'serviceName', value: '2026-04-15' },
      { id: 'activeAt', value: [from, ''] },
    ];

    expect(reviveTriggeredAtFilter(filters)).toEqual(filters);
    expect((reviveTriggeredAtFilter(filters)[1].value as unknown[])[0]).toBe(
      from
    );
  });
});
