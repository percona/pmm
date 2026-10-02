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
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { SchemaFormRenderer } from './SchemaFormRenderer';
import { EMPTY_SECTION_SUMMARY } from './utils/sectionValueSummary';
import type { FormSection } from './types';

function renderWithProviders(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  );
}

const SECTIONS: FormSection[] = [
  {
    title: 'Shared parameters',
    fields: [{ type: 'string', name: 'host', label: 'Host', default: 'db-1' }],
  },
  {
    title: 'Collect slow log',
    collapsible: true,
    collapsed_by_default: true,
    fields: [
      { type: 'integer', name: 'minutes', label: 'Minutes', default: 30 },
      { type: 'bool', name: 'verbose', label: 'Verbose' },
    ],
  },
];

/** The collapsed section's own disclosure control. */
function sectionHeader() {
  return screen.getByRole('button', {
    name: /Collect slow log/,
    expanded: false,
  });
}

describe('SchemaFormRenderer collapsed-section summary', () => {
  it('names a collapsed section’s values in its header', () => {
    renderWithProviders(
      <SchemaFormRenderer
        sections={SECTIONS}
        onSubmit={() => {}}
        sectionValueSummary
      />
    );

    expect(within(sectionHeader()).getByText('Minutes: 30')).toBeVisible();
  });

  it('says a section holds nothing rather than leaving the header bare', () => {
    const empty: FormSection[] = [
      {
        title: 'Collect slow log',
        collapsible: true,
        collapsed_by_default: true,
        fields: [{ type: 'string', name: 'note', label: 'Note' }],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer
        sections={empty}
        onSubmit={() => {}}
        sectionValueSummary
      />
    );

    expect(screen.getByText(EMPTY_SECTION_SUMMARY)).toBeVisible();
  });

  it('withholds the summary until the section is collapsed again', async () => {
    renderWithProviders(
      <SchemaFormRenderer
        sections={SECTIONS}
        onSubmit={() => {}}
        sectionValueSummary
      />
    );

    await userEvent.click(sectionHeader());

    // The open section shows every value in full, so repeating them in the
    // header would read as a second, staler copy.
    expect(screen.queryByText('Minutes: 30')).toBeNull();
  });

  it('tracks a value the reader changes', async () => {
    renderWithProviders(
      <SchemaFormRenderer
        sections={SECTIONS}
        onSubmit={() => {}}
        sectionValueSummary
      />
    );

    const header = sectionHeader();
    await userEvent.click(header);
    const input = screen.getByTestId('text-input-minutes');
    await userEvent.clear(input);
    await userEvent.type(input, '90');
    await userEvent.click(header);

    expect(within(sectionHeader()).getByText('Minutes: 90')).toBeVisible();
  });

  it('is opt-in — an unflagged form leaves the header to the title alone', () => {
    renderWithProviders(
      <SchemaFormRenderer sections={SECTIONS} onSubmit={() => {}} />
    );

    expect(screen.queryByText('Minutes: 30')).toBeNull();
  });
});

describe('SchemaFormRenderer sticky submit', () => {
  it('closes the form with the submit control by default', () => {
    const { container } = renderWithProviders(
      <SchemaFormRenderer sections={SECTIONS} onSubmit={() => {}} />
    );

    const form = container.querySelector('form') as HTMLFormElement;
    const submit = screen.getByRole('button', { name: 'Run' });
    expect(form.lastElementChild?.contains(submit)).toBe(true);
    expect(submit.parentElement).not.toHaveStyle({ position: 'sticky' });
  });

  it('pins the submit control above the sections at the offset it is given', () => {
    const { container } = renderWithProviders(
      <SchemaFormRenderer
        sections={SECTIONS}
        onSubmit={() => {}}
        submitPlacement="sticky-top"
        stickySubmitOffset={64}
      />
    );

    const form = container.querySelector('form') as HTMLFormElement;
    const submit = screen.getByRole('button', { name: 'Run' });
    // First in the form: a sticky row declared after the sections would unstick
    // the moment the last one scrolled past.
    expect(form.firstElementChild?.contains(submit)).toBe(true);
    expect(submit.parentElement).toHaveStyle({
      position: 'sticky',
      top: '64px',
    });
  });

  it('still submits the form it is pinned above', async () => {
    let submitted = false;
    renderWithProviders(
      <SchemaFormRenderer
        sections={SECTIONS}
        onSubmit={() => {
          submitted = true;
        }}
        submitPlacement="sticky-top"
      />
    );

    await userEvent.click(screen.getByRole('button', { name: 'Run' }));

    expect(submitted).toBe(true);
  });
});
