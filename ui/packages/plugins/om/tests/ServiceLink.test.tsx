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
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import {
  ServiceLink,
  serviceDashboardPath,
} from '../src/components/ServiceLink';

describe('serviceDashboardPath', () => {
  it('opens the MongoDB instance dashboard filtered to the service', () => {
    expect(serviceDashboardPath('om-mongo1')).toBe(
      '/graph/d/mongodb-instance-summary/mongodb-instance-summary?var-service_name=om-mongo1'
    );
  });

  it('encodes a name that is not URL-safe', () => {
    expect(serviceDashboardPath('rs0 node&1')).toContain(
      'var-service_name=rs0+node%261'
    );
  });
});

describe('ServiceLink', () => {
  it('renders the service name as a link to its dashboard', () => {
    render(
      <MemoryRouter>
        <ServiceLink serviceName="om-mongo1" />
      </MemoryRouter>
    );

    expect(screen.getByRole('link', { name: 'om-mongo1' })).toHaveAttribute(
      'href',
      serviceDashboardPath('om-mongo1')
    );
  });
});
