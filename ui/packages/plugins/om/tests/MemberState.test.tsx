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

import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { MemberState } from '../src/components/MemberState';
import { UNAVAILABLE_PHRASE } from '../src/constants';

const member = {
  state: null,
  status: 'SERVICE_STATUS_UP',
  process_role: 'PROCESS_ROLE_MONGOD',
} as const;

describe('MemberState', () => {
  it('spells out the member state MongoDB reports', () => {
    render(<MemberState service={{ ...member, state: 'SECONDARY' }} />);

    expect(screen.getByText('SECONDARY')).toBeInTheDocument();
  });

  // replication_set is optional at registration, so a down member registered with
  // only --cluster looks like a down standalone; neither reports a state this run.
  it('calls a down mongod unobserved, whatever its replica-set label', () => {
    render(
      <MemberState service={{ ...member, status: 'SERVICE_STATUS_DOWN' }} />
    );

    expect(
      screen.getByLabelText(UNAVAILABLE_PHRASE.service_not_observed)
    ).toBeInTheDocument();
  });

  it.each([
    ['an up router', { process_role: 'PROCESS_ROLE_MONGOS' }],
    [
      'a down router',
      {
        process_role: 'PROCESS_ROLE_MONGOS',
        status: 'SERVICE_STATUS_DOWN',
      },
    ],
    ['an up mongod with no state', {}],
  ] as const)('calls %s not applicable', (_name, overrides) => {
    render(<MemberState service={{ ...member, ...overrides }} />);

    expect(
      screen.getByLabelText(UNAVAILABLE_PHRASE.not_applicable)
    ).toBeInTheDocument();
  });
});
