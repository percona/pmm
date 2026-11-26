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
  constrainMemberConfig,
  defaultMemberConfig,
  memberConstraint,
  validateElectionPlan,
  type OmElectionPlanMember,
} from '../src/electionPlan';
import type { OmBootstrapMemberConfig } from '../src/types';

const member = (
  name: string,
  overrides: Partial<OmBootstrapMemberConfig> = {}
): OmElectionPlanMember => ({
  name,
  config: { ...defaultMemberConfig(), ...overrides },
});

describe('constrainMemberConfig', () => {
  it('leaves an ordinary member alone', () => {
    expect(
      constrainMemberConfig({ ...defaultMemberConfig(), priority: 5 })
    ).toEqual({
      ...defaultMemberConfig(),
      priority: 5,
    });
  });

  it('turns a delayed member into a hidden, non-voting, priority-0 one', () => {
    expect(
      constrainMemberConfig({ ...defaultMemberConfig(), delay_secs: 3600 })
    ).toEqual({ priority: 0, votes: false, hidden: true, delay_secs: 3600 });
  });

  it('keeps a delayed member hidden when Hidden is cleared', () => {
    expect(
      constrainMemberConfig({
        priority: 0,
        votes: false,
        hidden: false,
        delay_secs: 3600,
      }).hidden
    ).toBe(true);
  });

  it.each([
    ['hidden', { hidden: true }],
    ['non-voting', { votes: false }],
  ])(
    'sets a %s member to priority 0 and keeps the rest',
    (_name, overrides) => {
      expect(
        constrainMemberConfig({ ...defaultMemberConfig(), ...overrides })
      ).toEqual({ ...defaultMemberConfig(), ...overrides, priority: 0 });
    }
  );

  it('keeps a hidden member voting', () => {
    expect(
      constrainMemberConfig({ ...defaultMemberConfig(), hidden: true }).votes
    ).toBe(true);
  });
});

describe('memberConstraint', () => {
  it.each([
    [{}, null],
    [{ delay_secs: 10, hidden: true }, 'delayed'],
    [{ hidden: true }, 'hidden'],
    [{ votes: false, priority: 0 }, 'non_voting'],
  ] as const)('names %o as %s', (overrides, constraint) => {
    expect(memberConstraint({ ...defaultMemberConfig(), ...overrides })).toBe(
      constraint
    );
  });
});

describe('validateElectionPlan', () => {
  it('accepts the default plan with nothing to say', () => {
    expect(
      validateElectionPlan([member('a'), member('b'), member('c')])
    ).toEqual({ errors: [], warnings: [] });
  });

  it('blocks a plan with every member at priority 0, naming the hosts', () => {
    const { errors } = validateElectionPlan([
      member('a', { priority: 0 }),
      member('b', { priority: 0 }),
      member('c', { priority: 0 }),
    ]);

    expect(errors).toHaveLength(1);
    expect(errors[0]).toMatch(/^No member can become primary: a, b and c/);
  });

  it('blocks a plan whose only electable members are hidden or delayed', () => {
    const { errors } = validateElectionPlan([
      member('a', { priority: 0, votes: false, delay_secs: 60 }),
      member('b', { priority: 0, hidden: true }),
      member('c', { priority: 0, votes: false, delay_secs: 60 }),
    ]);

    expect(
      errors.some((e) => e.startsWith('No member can become primary'))
    ).toBe(true);
  });

  it('passes a plan with one electable member among hidden and delayed ones', () => {
    expect(
      validateElectionPlan([
        member('a'),
        member('b', { priority: 0, hidden: true }),
        member('c', {
          priority: 0,
          votes: false,
          hidden: true,
          delay_secs: 60,
        }),
      ]).errors
    ).toEqual([]);
  });

  it('blocks a hidden member left above priority 0', () => {
    const { errors } = validateElectionPlan([
      member('a'),
      member('b', { hidden: true }),
      member('c'),
    ]);

    expect(errors).toEqual([
      'b: a hidden, delayed or non-voting member must have priority 0.',
    ]);
  });

  it('blocks a delayed member that still votes', () => {
    const { errors } = validateElectionPlan([
      member('a'),
      member('b', { priority: 0, hidden: true, delay_secs: 60 }),
      member('c'),
    ]);

    expect(errors).toEqual(['b: a delayed member cannot vote.']);
  });

  it('blocks a delayed member that applications can still see', () => {
    const { errors } = validateElectionPlan([
      member('a'),
      member('b', { priority: 0, votes: false, delay_secs: 3600 }),
      member('c'),
    ]);

    expect(errors).toEqual([
      'b: a delayed member must be hidden, so applications are not served its delayed data.',
    ]);
  });

  it('blocks a plan where nobody votes', () => {
    const { errors } = validateElectionPlan([
      member('a', { priority: 0, votes: false }),
      member('b', { priority: 0, votes: false }),
      member('c', { priority: 0, votes: false }),
    ]);

    expect(errors).toEqual([
      'No member has a vote. A replica set needs at least one voting member to elect a primary.',
    ]);
  });

  it('warns, without blocking, on an even number of voters', () => {
    const { errors, warnings } = validateElectionPlan([
      member('a'),
      member('b'),
      member('c', { priority: 0, votes: false }),
    ]);

    expect(errors).toEqual([]);
    expect(warnings).toEqual([
      expect.stringMatching(/^2 voting members\. An odd number of votes/),
    ]);
  });

  it('blocks a single member at priority 0', () => {
    expect(
      validateElectionPlan([member('solo', { priority: 0 })]).errors[0]
    ).toMatch(/^No member can become primary: solo is at priority 0/);
  });

  it('says nothing about an empty plan', () => {
    expect(validateElectionPlan([])).toEqual({ errors: [], warnings: [] });
  });
});
