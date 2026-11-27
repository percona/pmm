import { render, screen } from '@testing-library/react';
import type { Dump } from 'types/dump.types';
import { TEST_DUMP } from 'utils/testStubs';
import { TestWrapper } from 'utils/testWrapper';
import { DatasetSummary } from './DatasetSummary';

const renderSummary = (dump: Dump) =>
  render(
    <TestWrapper>
      <DatasetSummary dump={dump} />
    </TestWrapper>
  );

describe('DatasetSummary', () => {
  it('names the window and services', () => {
    renderSummary(TEST_DUMP);

    expect(
      screen.getByText('Oct 26, 18:39 – Oct 27, 06:39 · mysql-1, pg-1')
    ).toBeVisible();
  });

  it('falls back to All services', () => {
    renderSummary({ ...TEST_DUMP, serviceNames: [] });

    expect(
      screen.getByText('Oct 26, 18:39 – Oct 27, 06:39 · All services')
    ).toBeVisible();
  });

  it('shows N/A for an empty window', () => {
    renderSummary({ ...TEST_DUMP, startTime: '', endTime: '' });

    expect(screen.getByText('N/A · mysql-1, pg-1')).toBeVisible();
  });
});
