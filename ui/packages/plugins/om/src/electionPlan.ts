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

/**
 * Replica-set election settings, as plain functions.
 *
 * MongoDB's `rs.initiate()` rejects some member configurations outright and
 * accepts others that can never elect a primary. Both fail minutes into a
 * bootstrap, after every host already has mongod installed, so the wizard checks
 * the whole plan before Review rather than leaving it to the run.
 */

import type { OmBootstrapMemberConfig } from './types';

/** MongoDB's own defaults for a member `rs.initiate()` names no override for. */
export function defaultMemberConfig(): OmBootstrapMemberConfig {
  return { priority: 1, votes: true, hidden: false, delay_secs: 0 };
}

/**
 * Apply MongoDB's member rules to an edited config, the moment it is edited.
 *
 * A delayed member cannot vote or become primary, and is hidden: its copy is
 * deliberately behind, so an application reading from secondaries must not be
 * served it. A hidden member and a non-voting member cannot become primary.
 * Each rule forces the dependent field
 * rather than letting the form hold a value `rs.initiate()` would reject, and the
 * Configure step says which rule is holding a row (see {@link memberConstraint}).
 */
export function constrainMemberConfig(
  config: OmBootstrapMemberConfig
): OmBootstrapMemberConfig {
  const delayed = config.delay_secs > 0;
  const votes = delayed ? false : config.votes;
  const hidden = delayed || config.hidden;
  const canBePrimary = !delayed && !hidden && votes;
  return {
    ...config,
    votes,
    hidden,
    priority: canBePrimary ? config.priority : 0,
  };
}

/** Which rule, if any, is holding a member's priority (and votes) in place. */
export type OmMemberConstraint = 'delayed' | 'hidden' | 'non_voting' | null;

export function memberConstraint(
  config: OmBootstrapMemberConfig
): OmMemberConstraint {
  if (config.delay_secs > 0) {
    return 'delayed';
  }
  if (config.hidden) {
    return 'hidden';
  }
  if (!config.votes) {
    return 'non_voting';
  }
  return null;
}

/** The sentence the Configure step shows beside a constrained member. */
export const MEMBER_CONSTRAINT_PHRASE: Record<
  Exclude<OmMemberConstraint, null>,
  string
> = {
  delayed:
    'Priority set to 0, votes turned off and hidden: a delayed member cannot vote or become primary, and applications must not read its delayed data.',
  hidden: 'Priority set to 0: a hidden member cannot become primary.',
  non_voting:
    'Priority set to 0: a member without a vote cannot become primary.',
};

/** Whether a member can be elected primary at all. */
function isElectable(config: OmBootstrapMemberConfig): boolean {
  return (
    config.priority > 0 &&
    config.votes &&
    !config.hidden &&
    config.delay_secs === 0
  );
}

/** One member of the plan, by the name the reader knows its host by. */
export interface OmElectionPlanMember {
  name: string;
  config: OmBootstrapMemberConfig;
}

export interface OmElectionPlanCheck {
  /** Problems that stop the plan: Review stays disabled while any exist. */
  errors: string[];
  /** Advice the plan can go ahead without. */
  warnings: string[];
}

/** `a`, `a and b`, `a, b and c`. */
function listNames(names: string[]): string {
  if (names.length <= 1) {
    return names.join('');
  }
  return `${names.slice(0, -1).join(', ')} and ${names.at(-1)}`;
}

/**
 * Check the plan as a whole, the way `rs.initiate()` and the first election will.
 *
 * The per-member rules are checked again here although {@link constrainMemberConfig}
 * enforces them as the form is edited: this is what gates Review, so it must not
 * depend on every edit path having gone through the constraint.
 */
export function validateElectionPlan(
  members: OmElectionPlanMember[]
): OmElectionPlanCheck {
  const errors: string[] = [];
  const warnings: string[] = [];
  if (members.length === 0) {
    return { errors, warnings };
  }

  const ruleBreakers = members.filter(
    ({ config }) =>
      config.priority > 0 &&
      (config.hidden || !config.votes || config.delay_secs > 0)
  );
  if (ruleBreakers.length) {
    errors.push(
      `${listNames(ruleBreakers.map((m) => m.name))}: a hidden, delayed or non-voting member must have priority 0.`
    );
  }
  const delayedVoters = members.filter(
    ({ config }) => config.delay_secs > 0 && config.votes
  );
  if (delayedVoters.length) {
    errors.push(
      `${listNames(delayedVoters.map((m) => m.name))}: a delayed member cannot vote.`
    );
  }
  const visibleDelayed = members.filter(
    ({ config }) => config.delay_secs > 0 && !config.hidden
  );
  if (visibleDelayed.length) {
    errors.push(
      `${listNames(visibleDelayed.map((m) => m.name))}: a delayed member must be hidden, so applications are not served its delayed data.`
    );
  }

  const voters = members.filter(({ config }) => config.votes);
  if (voters.length === 0) {
    errors.push(
      'No member has a vote. A replica set needs at least one voting member to elect a primary.'
    );
  } else if (!members.some(({ config }) => isElectable(config))) {
    errors.push(
      `No member can become primary: ${listNames(members.map((m) => m.name))} ${
        members.length === 1 ? 'is' : 'are all'
      } at priority 0, hidden, delayed or without a vote. Give at least one member priority above 0, a vote, and no hidden or delay setting.`
    );
  }

  if (voters.length > 0 && voters.length % 2 === 0) {
    warnings.push(
      `${voters.length} voting members. An odd number of votes avoids tied elections: give one more member a vote, or take one away.`
    );
  }

  return { errors, warnings };
}
