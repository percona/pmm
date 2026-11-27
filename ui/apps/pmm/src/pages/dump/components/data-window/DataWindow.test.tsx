import { render, screen } from '@testing-library/react';
import type { Dump } from 'types/dump.types';
import { TEST_DUMP } from 'utils/testStubs';
import { TestWrapper } from 'utils/testWrapper';
import { DataWindow } from './DataWindow';

const renderWindow = (dump: Dump) =>
  render(
    <TestWrapper>
      <DataWindow dump={dump} />
    </TestWrapper>
  );

describe('DataWindow', () => {
  it('shows the window and its duration', () => {
    renderWindow(TEST_DUMP);

    expect(screen.getByText('Oct 26, 18:39 – Oct 27, 06:39')).toBeVisible();
    expect(screen.getByText('12 hours')).toBeVisible();
  });

  it('marks encrypted datasets', () => {
    const { unmount } = renderWindow(TEST_DUMP);
    expect(screen.queryByText('Encrypted')).not.toBeInTheDocument();
    unmount();

    renderWindow({ ...TEST_DUMP, encrypted: true });
    expect(screen.getByText('Encrypted')).toBeVisible();
  });

  it('shows N/A for unparsable times', () => {
    renderWindow({ ...TEST_DUMP, startTime: '', endTime: 'nope' });

    expect(screen.getAllByText('N/A')).toHaveLength(2);
  });
});
