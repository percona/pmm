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
 * The install form's Configure step checks, as plain functions.
 *
 * One list of sentences both gates Review and tells the user why it is disabled, so
 * the two cannot drift apart.
 */

import { REPLICA_SET_NAME_PATTERN } from './constants';
import type { OmBootstrapMemberConfig } from './types';

const REPLICA_SET_NAME_MAX_LENGTH = 64;

/** BootstrapMemberConfig.priority's own ceiling in om.proto. */
const MAX_MEMBER_PRIORITY = 1000;

/** delay_secs's wire type is uint32; om.proto sets no narrower limit than that. */
const MAX_DELAY_SECS = 4294967295;

export interface ConfigureFormState {
  replicaSetName: string;
  dataPath: string;
  logPath: string;
  port: string;
  customBindIp: string | null;
  members: { name: string; config: OmBootstrapMemberConfig }[];
  electionErrorCount: number;
}

/**
 * Why a replica set name would be refused, or null when it is fine.
 *
 * An empty name is not an error here: a form that opens with its first field in red
 * reads as a mistake the user has not made yet. {@link configureBlockers} still
 * reports it.
 */
export function replicaSetNameError(name: string): string | null {
  if (name === '') {
    return null;
  }
  if (name.length > REPLICA_SET_NAME_MAX_LENGTH) {
    return `Replica set name can be at most ${REPLICA_SET_NAME_MAX_LENGTH} characters.`;
  }
  if (!REPLICA_SET_NAME_PATTERN.test(name)) {
    return 'Replica set name can use only letters, digits, - and _.';
  }
  return null;
}

/**
 * Every reason the Configure step cannot go on to Review, in form order.
 *
 * The number inputs only clamp what a spinner arrow can reach: HTML `min`/`max` do
 * not stop a typed value, and `Number(...)` on empty input is `NaN`, so the parsed
 * values are checked here before TriggerHostBootstrap would reject them.
 */
export function configureBlockers(form: ConfigureFormState): string[] {
  const blockers: string[] = [];

  if (form.replicaSetName === '') {
    blockers.push('Enter a replica set name.');
  } else {
    const nameError = replicaSetNameError(form.replicaSetName);
    if (nameError) {
      blockers.push(nameError);
    }
  }
  if (form.dataPath.trim() === '') {
    blockers.push('Enter a data path.');
  }
  if (form.logPath.trim() === '') {
    blockers.push('Enter a log path.');
  }
  const port = Number(form.port);
  if (
    form.port.trim() === '' ||
    !Number.isInteger(port) ||
    port < 1 ||
    port > 65535
  ) {
    blockers.push('Port must be a whole number from 1 to 65535.');
  }
  if (form.customBindIp !== null && form.customBindIp.trim() === '') {
    blockers.push('Enter a bind IP.');
  }

  for (const { name, config } of form.members) {
    if (
      !Number.isInteger(config.priority) ||
      config.priority < 0 ||
      config.priority > MAX_MEMBER_PRIORITY
    ) {
      blockers.push(
        `Priority for ${name} must be a whole number from 0 to ${MAX_MEMBER_PRIORITY}.`
      );
    }
    if (
      !Number.isInteger(config.delay_secs) ||
      config.delay_secs < 0 ||
      config.delay_secs > MAX_DELAY_SECS
    ) {
      blockers.push(`Delay for ${name} must be a whole number of seconds.`);
    }
  }
  if (form.electionErrorCount === 1) {
    blockers.push('Fix the election settings problem shown above.');
  } else if (form.electionErrorCount > 1) {
    blockers.push(
      `Fix the ${form.electionErrorCount} election settings problems shown above.`
    );
  }

  return blockers;
}
