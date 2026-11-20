import { describe, expect, it } from 'vitest';
import {
  areCustomLabelsValid,
  parseCustomLabels,
  toCustomLabelsText,
} from './labels.utils';

describe('custom labels', () => {
  it('round-trips between the API map and the text', () => {
    const labels = { team: 'db', region_name: 'eu' };

    expect(toCustomLabelsText(labels)).toBe('team:db\nregion_name:eu');
    expect(parseCustomLabels('team:db\n region_name:eu  ')).toEqual(labels);
  });

  it('keeps colons in label values', () => {
    const labels = { endpoint: 'db:3306', url: 'http://db' };

    expect(parseCustomLabels(toCustomLabelsText(labels))).toEqual(labels);
  });

  it('keeps spaces in label values', () => {
    const labels = { owner: 'DB Team', team: 'db' };

    expect(parseCustomLabels(toCustomLabelsText(labels))).toEqual(labels);
    expect(parseCustomLabels('owner:DB Team env:prod team:db')).toEqual({
      owner: 'DB Team',
      env: 'prod',
      team: 'db',
    });
  });

  it.each([
    ['', true],
    ['team:db\nregion_name:eu', true],
    ['team:db region_name:eu', true],
    ['team', false],
    ['team:', false],
    ['endpoint:db:3306', true],
    ['owner:DB Team', true],
    ['team:db 1team:db', false],
    ['1team:db', false],
    ['__reserved:db', false],
  ])('validates %j as %s', (text, valid) => {
    expect(areCustomLabelsValid(text)).toBe(valid);
  });
});
