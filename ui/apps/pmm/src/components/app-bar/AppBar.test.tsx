import { screen, render } from '@testing-library/react';
import { AppBar } from '.';
import { PMM_HOME_URL, PMM_SUPPORT_URL } from 'lib/constants';
import { TestWrapper } from 'utils/testWrapper';

describe('AppBar', () => {
  it('links back to older PMM', () => {
    render(
      <TestWrapper>
        <AppBar />
      </TestWrapper>
    );

    expect(screen.getByTestId('appbar-pmm-link')).toHaveAttribute(
      'href',
      PMM_HOME_URL
    );
  });

  it('links to support', () => {
    render(
      <TestWrapper>
        <AppBar />
      </TestWrapper>
    );

    expect(screen.getByTestId('appbar-support-link')).toHaveAttribute(
      'href',
      PMM_SUPPORT_URL
    );
  });
});
