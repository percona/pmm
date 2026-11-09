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
import { Unavailable } from '../src/components/Unavailable';
import { UNAVAILABLE_FALLBACK, UNAVAILABLE_PHRASE } from '../src/constants';

describe('Unavailable', () => {
  it('names the reason a value is missing', () => {
    render(<Unavailable reason="service_not_observed" />);
    expect(
      screen.getByLabelText(UNAVAILABLE_PHRASE.service_not_observed)
    ).toHaveTextContent('—');
  });

  it('explains an absent version catalog rather than claiming no update', () => {
    render(<Unavailable reason="no_version_catalog" />);
    expect(
      screen.getByLabelText(UNAVAILABLE_PHRASE.no_version_catalog)
    ).toBeInTheDocument();
  });

  it('distinguishes a topology that cannot have the field from a gap', () => {
    render(<Unavailable reason="not_applicable" />);
    expect(
      screen.getByLabelText(UNAVAILABLE_PHRASE.not_applicable)
    ).toHaveTextContent('—');
  });

  // A standalone has no oplog at all, which is a fact about the estate. Saying
  // "not collected" there would send someone hunting for a broken collector.
  it('phrases not_applicable differently from metric_not_collected', () => {
    expect(UNAVAILABLE_PHRASE.not_applicable).not.toBe(
      UNAVAILABLE_PHRASE.metric_not_collected
    );
  });

  // The worker may add reason codes before the frontend learns them; an unknown
  // code must still render as unavailable, never as a blank or a zero.
  it('falls back for a reason code it has not been taught', () => {
    render(<Unavailable reason="something_new" />);
    expect(screen.getByLabelText(UNAVAILABLE_FALLBACK)).toHaveTextContent('—');
  });

  it('falls back when no reason is given at all', () => {
    render(<Unavailable />);
    expect(screen.getByLabelText(UNAVAILABLE_FALLBACK)).toBeInTheDocument();
  });
});
