/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { describe, expect, it } from 'vitest';
import {
  isAutoMongoBackupTaskName,
  slugifyServiceName,
  suggestMongoBackupTaskName,
} from './suggestMongoBackupTaskName';

describe('suggestMongoBackupTaskName', () => {
  const fixed = new Date('2026-07-16T14:30:22.123Z');

  it('uses mongodb when no service name is given', () => {
    expect(suggestMongoBackupTaskName(undefined, fixed)).toBe(
      'mongodb-backup-20260716T143022Z'
    );
  });

  it('slugifies the service name into the suggestion', () => {
    expect(suggestMongoBackupTaskName('My Mongo Cluster!', fixed)).toBe(
      'my-mongo-cluster-backup-20260716T143022Z'
    );
  });
});

describe('slugifyServiceName', () => {
  it('lowercases and collapses non-alphanumerics', () => {
    expect(slugifyServiceName('  Foo_Bar--Baz  ')).toBe('foo-bar-baz');
  });

  it('returns empty for blank input', () => {
    expect(slugifyServiceName('')).toBe('');
    expect(slugifyServiceName(null)).toBe('');
  });
});

describe('isAutoMongoBackupTaskName', () => {
  const schemaDefault = 'mongodb-backup';

  it('treats empty and schema default as auto', () => {
    expect(isAutoMongoBackupTaskName('', undefined)).toBe(true);
    expect(
      isAutoMongoBackupTaskName(schemaDefault, undefined, schemaDefault)
    ).toBe(true);
  });

  it('does not treat the schema default when none is provided', () => {
    expect(isAutoMongoBackupTaskName(schemaDefault, undefined)).toBe(false);
  });

  it('treats the previous auto value as auto', () => {
    expect(
      isAutoMongoBackupTaskName(
        'svc-backup-20260716T143022Z',
        'svc-backup-20260716T143022Z'
      )
    ).toBe(true);
  });

  it('treats a manual edit as not auto', () => {
    expect(
      isAutoMongoBackupTaskName('my-custom-name', 'svc-backup-20260716T143022Z')
    ).toBe(false);
  });
});
