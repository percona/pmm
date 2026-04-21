import { render, screen } from '@testing-library/react';
import { Settings } from './Settings';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import * as settingsApi from 'api/settings';

vi.mock('api/settings');

const getSettingsMock = vi.mocked(settingsApi.getSettings);

describe('Settings', () => {
  beforeEach(() => {
    getSettingsMock.mockImplementation(() => new Promise(() => {}));
  });

  it('shows loading state when settings are not yet loaded', () => {
    render(<TestWrapper>{wrapWithQueryProvider(<Settings />)}</TestWrapper>);

    expect(screen.getByTestId('settings-loading')).toBeInTheDocument();
  });
});
