import { render, screen } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { DumpEmptyState } from './DumpEmptyState';

describe('DumpEmptyState', () => {
  it('offers to create a dataset', () => {
    render(
      <TestWrapper>
        <DumpEmptyState />
      </TestWrapper>
    );

    expect(
      screen.getByRole('heading', { name: 'No datasets yet' })
    ).toBeVisible();
    expect(screen.getByTestId('create-dataset')).toHaveAttribute(
      'href',
      '/pmm-dump/new'
    );
  });
});
