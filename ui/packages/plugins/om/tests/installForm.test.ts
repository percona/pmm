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
import { defaultMemberConfig } from '../src/electionPlan';
import {
  type ConfigureFormState,
  configureBlockers,
  replicaSetNameError,
} from '../src/installForm';

const validForm = (): ConfigureFormState => ({
  replicaSetName: 'rs0',
  dataPath: '/var/lib/mongo',
  logPath: '/var/log/mongodb/mongod.log',
  port: '27017',
  customBindIp: null,
  members: [{ name: 'db01', config: defaultMemberConfig() }],
  electionErrorCount: 0,
  adminUsername: 'admin',
  adminPassword: null,
});

describe('replicaSetNameError', () => {
  it.each(['rs0', 'rs-orders_1', 'A'.repeat(64)])('accepts %s', (name) => {
    expect(replicaSetNameError(name)).toBeNull();
  });

  // The same rule PMM Extensions enforces: `.` is refused there, so it is refused here.
  it.each(['my set', 'bad/name', 'a:b', 'rs#1', 'rs.0', 'rs0 '])(
    'refuses %s',
    (name) => {
      expect(replicaSetNameError(name)).toMatch(/letters, digits, - and _/);
    }
  );

  it('refuses a name over 64 characters', () => {
    expect(replicaSetNameError('a'.repeat(65))).toMatch(/at most 64/);
  });

  it('does not flag an empty field', () => {
    expect(replicaSetNameError('')).toBeNull();
  });
});

describe('configureBlockers', () => {
  it('has nothing to say about a valid form', () => {
    expect(configureBlockers(validForm())).toEqual([]);
  });

  it('asks for a replica set name when it is empty', () => {
    expect(configureBlockers({ ...validForm(), replicaSetName: '' })).toEqual([
      'Enter a replica set name.',
    ]);
  });

  it('reports every problem at once, in form order', () => {
    const blockers = configureBlockers({
      ...validForm(),
      replicaSetName: 'bad/name',
      dataPath: ' ',
      logPath: '',
      port: '70000',
      customBindIp: '',
      members: [
        { name: 'db01', config: { ...defaultMemberConfig(), priority: 1001 } },
        { name: 'db02', config: { ...defaultMemberConfig(), delay_secs: NaN } },
      ],
      electionErrorCount: 2,
    });

    expect(blockers).toEqual([
      'Replica set name can use only letters, digits, - and _.',
      'Enter a data path.',
      'Enter a log path.',
      'Port must be a whole number from 1 to 65535.',
      'Enter a bind IP.',
      'Priority for db01 must be a whole number from 0 to 1000.',
      'Delay for db02 must be a whole number of seconds.',
      'Fix the 2 election settings problems shown above.',
    ]);
  });

  it.each(['', '0', '1.5', 'abc'])('refuses port %j', (port) => {
    expect(configureBlockers({ ...validForm(), port })).toEqual([
      'Port must be a whole number from 1 to 65535.',
    ]);
  });

  it('points at a single election settings problem without repeating it', () => {
    expect(
      configureBlockers({ ...validForm(), electionErrorCount: 1 })
    ).toEqual(['Fix the election settings problem shown above.']);
  });
});

describe('configureBlockers administrator', () => {
  it('refuses the monitoring user and characters outside the API pattern', () => {
    expect(configureBlockers({ ...validForm(), adminUsername: 'pmm' })).toEqual(
      ["pmm is PMM's monitoring user. Choose another administrator username."]
    );
    expect(
      configureBlockers({ ...validForm(), adminUsername: 'db admin' })
    ).toEqual([
      'Administrator username can use only letters, digits, ., _, @ and -, up to 64 characters.',
    ]);
    expect(configureBlockers({ ...validForm(), adminUsername: '' })).toEqual([
      'Enter an administrator username.',
    ]);
  });

  it('checks a typed password for length, then for a matching confirmation', () => {
    const typed = (value: string, confirmation: string) =>
      configureBlockers({
        ...validForm(),
        adminPassword: { value, confirmation },
      });

    expect(typed('short', 'short')).toEqual([
      'Administrator password must be at least 8 characters.',
    ]);
    expect(typed('long-enough', 'long-enoug')).toEqual([
      'The administrator passwords do not match.',
    ]);
    expect(typed('long-enough', 'long-enough')).toEqual([]);
  });
});
