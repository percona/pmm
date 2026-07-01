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
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SnippetDetailPage } from './SnippetDetailPage';
import { useSnippetDownload, useSnippetSchema } from './hooks';

vi.mock('./hooks', () => ({
  useSnippetSchema: vi.fn(),
  useSnippetDownload: vi.fn(),
}));

vi.mock('@sep/framework', () => ({
  SnippetExecutionAccordion: ({
    title,
    description,
  }: {
    title?: string;
    description?: string;
  }) => (
    <div data-testid="snippet-execution-accordion">
      {title && <span data-testid="accordion-title">{title}</span>}
      {description && (
        <span data-testid="accordion-description">{description}</span>
      )}
    </div>
  ),
}));

const mockSchema = vi.mocked(useSnippetSchema);
const mockDownload = vi.mocked(useSnippetDownload);

function renderAt(filename: string) {
  return render(
    <MemoryRouter initialEntries={[`/snippets/${filename}`]}>
      <Routes>
        <Route path="/snippets/:filename" element={<SnippetDetailPage />} />
      </Routes>
    </MemoryRouter>
  );
}

describe('SnippetDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockSchema.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useSnippetSchema>);
    mockDownload.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
      isError: false,
      error: null,
      reset: vi.fn(),
    } as unknown as ReturnType<typeof useSnippetDownload>);
  });

  it('passes display_name from schema as accordion title', () => {
    mockSchema.mockReturnValue({
      data: { display_name: 'My Snippet', description: undefined },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useSnippetSchema>);

    renderAt('my-snippet.sh');

    expect(screen.getByTestId('accordion-title')).toHaveTextContent(
      'My Snippet'
    );
  });

  it('falls back to filename as title when schema has no display_name', () => {
    mockSchema.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useSnippetSchema>);

    renderAt('fallback.sh');

    expect(screen.getByTestId('accordion-title')).toHaveTextContent(
      'fallback.sh'
    );
  });

  it('passes description from schema to accordion', () => {
    mockSchema.mockReturnValue({
      data: { display_name: 'My Snippet', description: 'Does useful things' },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useSnippetSchema>);

    renderAt('my-snippet.sh');

    expect(screen.getByTestId('accordion-description')).toHaveTextContent(
      'Does useful things'
    );
  });

  it('renders back navigation link', () => {
    renderAt('check.sh');

    expect(
      screen.getByRole('button', { name: /back to snippets/i })
    ).toBeInTheDocument();
  });

  it('renders the SnippetExecutionAccordion', () => {
    renderAt('check.sh');

    expect(
      screen.getByTestId('snippet-execution-accordion')
    ).toBeInTheDocument();
  });

  it('calls useSnippetSchema with executionOnly=true to avoid a duplicate schema fetch', () => {
    renderAt('check.sh');

    expect(mockSchema).toHaveBeenCalledWith('check.sh', true);
  });
});
