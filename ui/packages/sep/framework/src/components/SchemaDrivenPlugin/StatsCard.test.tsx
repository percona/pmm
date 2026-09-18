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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, expect, it, vi, beforeEach } from 'vitest';

const { mockUseTaskStats } = vi.hoisted(() => ({
  mockUseTaskStats: vi.fn(),
}));

vi.mock('../../hooks/useTaskStats', async () => {
  const actual = await vi.importActual<
    typeof import('../../hooks/useTaskStats')
  >('../../hooks/useTaskStats');
  return {
    ...actual,
    useTaskStats: (...args: unknown[]) => mockUseTaskStats(...args),
  };
});

import { StatsCard } from './StatsCard';

function makeClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
}

function renderWithClient(ui: React.ReactElement) {
  return render(
    <QueryClientProvider client={makeClient()}>{ui}</QueryClientProvider>
  );
}

const POPULATED = {
  engine: 'nomad',
  total: 5,
  status: { pass: 4, fail: 1 },
  duration: {
    average_seconds: 1.234,
    last_seconds: 0.987,
    total_seconds: 6.17,
  },
  last_finished_at: new Date(Date.now() - 60_000).toISOString(),
};

beforeEach(() => {
  mockUseTaskStats.mockReset();
});

describe('StatsCard — render states', () => {
  it('renders a skeleton while loading', () => {
    mockUseTaskStats.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
    });
    const { container } = renderWithClient(<StatsCard taskName="foo" />);
    expect(screen.getByText('Stats')).toBeInTheDocument();
    expect(container.querySelector('.MuiSkeleton-root')).not.toBeNull();
    expect(screen.queryByText('No execution history yet')).toBeNull();
    expect(screen.queryByText('Executions')).toBeNull();
  });

  it('renders the empty state when total === 0', () => {
    mockUseTaskStats.mockReturnValue({
      data: {
        engine: 'nomad',
        total: 0,
        status: { pass: 0, fail: 0 },
        duration: {
          average_seconds: null,
          last_seconds: null,
          total_seconds: null,
        },
        last_finished_at: null,
      },
      isLoading: false,
      isError: false,
    });
    renderWithClient(<StatsCard taskName="foo" />);
    expect(screen.getByText('No execution history yet')).toBeInTheDocument();
    expect(screen.queryByText('Executions')).toBeNull();
  });

  it('renders all six fields with formatted values when populated', () => {
    mockUseTaskStats.mockReturnValue({
      data: POPULATED,
      isLoading: false,
      isError: false,
    });
    renderWithClient(<StatsCard taskName="foo" />);
    expect(screen.getByText('Executions')).toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument();
    expect(screen.getByText('Succeeded')).toBeInTheDocument();
    expect(screen.getByText('4')).toBeInTheDocument();
    expect(screen.getByText('Failed')).toBeInTheDocument();
    expect(screen.getByText('1')).toBeInTheDocument();
    expect(screen.getByText('Avg Duration')).toBeInTheDocument();
    expect(screen.getByText('1.234s')).toBeInTheDocument();
    expect(screen.getByText('Last Duration')).toBeInTheDocument();
    expect(screen.getByText('0.987s')).toBeInTheDocument();
    expect(screen.getByText('Last Finished')).toBeInTheDocument();
    // Relative time must not be the raw ISO string.
    expect(screen.queryByText(POPULATED.last_finished_at)).toBeNull();
  });

  it('renders an inline error message on fetch failure', () => {
    mockUseTaskStats.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: new Error('boom'),
    });
    renderWithClient(<StatsCard taskName="foo" />);
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Could not load execution stats'
    );
  });

  it('renders empty state for 404 error', async () => {
    const { ApiError } = await import('@sep/api');
    mockUseTaskStats.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: new ApiError({ kind: 'http', status: 404, message: 'Not found' }),
    });
    renderWithClient(<StatsCard taskName="foo" />);
    expect(screen.getByText('No execution history yet')).toBeInTheDocument();
    expect(screen.queryByRole('alert')).toBeNull();
  });

  it('renders inline error for 401 (does not redirect)', async () => {
    const { ApiError } = await import('@sep/api');
    mockUseTaskStats.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: new ApiError({
        kind: 'http',
        status: 401,
        message: 'Unauthorized',
      }),
    });
    renderWithClient(<StatsCard taskName="foo" />);
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Could not load execution stats'
    );
  });

  it('renders inline error for 502 upstream Tasks-API failure', async () => {
    const { ApiError } = await import('@sep/api');
    mockUseTaskStats.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: new ApiError({
        kind: 'http',
        status: 502,
        message: 'tasks unreachable',
      }),
    });
    renderWithClient(<StatsCard taskName="foo" />);
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Could not load execution stats'
    );
  });
});

describe('StatsCard — taskName guards', () => {
  it('renders nothing when taskName is undefined', () => {
    mockUseTaskStats.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: false,
    });
    const { container } = renderWithClient(<StatsCard taskName={undefined} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders nothing for empty-string taskName', () => {
    mockUseTaskStats.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: false,
    });
    const { container } = renderWithClient(<StatsCard taskName="" />);
    expect(container.firstChild).toBeNull();
  });

  it('renders nothing for whitespace-only taskName', () => {
    mockUseTaskStats.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: false,
    });
    const { container } = renderWithClient(<StatsCard taskName="   " />);
    expect(container.firstChild).toBeNull();
  });
});

describe('StatsCard — defensive formatting', () => {
  it('shows placeholder for null last_finished_at', () => {
    mockUseTaskStats.mockReturnValue({
      data: { ...POPULATED, last_finished_at: null },
      isLoading: false,
      isError: false,
    });
    renderWithClient(<StatsCard taskName="foo" />);
    expect(screen.getByText('Last Finished')).toBeInTheDocument();
    // The placeholder em-dash renders instead of throwing on new Date(null).
    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(1);
  });

  it('shows placeholder for malformed ISO timestamp', () => {
    mockUseTaskStats.mockReturnValue({
      data: { ...POPULATED, last_finished_at: 'not-a-date' },
      isLoading: false,
      isError: false,
    });
    renderWithClient(<StatsCard taskName="foo" />);
    expect(screen.queryByText('Invalid Date')).toBeNull();
    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(1);
  });

  it('shows placeholder for missing average_seconds', () => {
    mockUseTaskStats.mockReturnValue({
      data: { ...POPULATED, duration: { last_seconds: 0.5 } },
      isLoading: false,
      isError: false,
    });
    renderWithClient(<StatsCard taskName="foo" />);
    expect(screen.getByText('Avg Duration')).toBeInTheDocument();
    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('0.500s')).toBeInTheDocument();
  });

  it('defaults missing pass/fail keys to 0', () => {
    mockUseTaskStats.mockReturnValue({
      data: { ...POPULATED, status: {} },
      isLoading: false,
      isError: false,
    });
    renderWithClient(<StatsCard taskName="foo" />);
    expect(screen.getByText('Succeeded')).toBeInTheDocument();
    expect(screen.getByText('Failed')).toBeInTheDocument();
    const zeros = screen.getAllByText('0');
    expect(zeros.length).toBeGreaterThanOrEqual(2);
  });

  it('formats absurdly large duration without crash', () => {
    mockUseTaskStats.mockReturnValue({
      data: {
        ...POPULATED,
        duration: { average_seconds: 9999999.999, last_seconds: -1 },
      },
      isLoading: false,
      isError: false,
    });
    renderWithClient(<StatsCard taskName="foo" />);
    expect(screen.getByText('9999999.999s')).toBeInTheDocument();
    expect(screen.getByText('-1.000s')).toBeInTheDocument();
  });

  it('escapes XSS attempt in last_finished_at as text', () => {
    mockUseTaskStats.mockReturnValue({
      data: { ...POPULATED, last_finished_at: '<script>alert(1)</script>' },
      isLoading: false,
      isError: false,
    });
    const { container } = renderWithClient(<StatsCard taskName="foo" />);
    expect(container.querySelector('script')).toBeNull();
  });
});
