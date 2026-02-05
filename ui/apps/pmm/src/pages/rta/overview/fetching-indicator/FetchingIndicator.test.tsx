import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { TestWrapper } from 'utils/testWrapper';
import FetchingIndicator from './FetchingIndicator';
import { Messages } from './FetchingIndicator.messages';

describe('FetchingIndicator', () => {
  it('should render fetching indicator when isFetching is true', () => {
    render(
      <TestWrapper>
        <FetchingIndicator isFetching={true} />
      </TestWrapper>
    );

    expect(screen.getByTestId('fetching-indicator-on')).toBeInTheDocument();
    expect(screen.getByText(Messages.fetching)).toBeInTheDocument();
  });

  it("shouldn't render fetching indicator when isFetching is false", () => {
    render(
      <TestWrapper>
        <FetchingIndicator isFetching={false} />
      </TestWrapper>
    );

    expect(screen.getByTestId('fetching-indicator-off')).toBeInTheDocument();
    expect(screen.getByText(Messages.fetching)).toBeInTheDocument();
    expect(
      screen.queryByTestId('fetching-indicator-on')
    ).not.toBeInTheDocument();
  });
});
