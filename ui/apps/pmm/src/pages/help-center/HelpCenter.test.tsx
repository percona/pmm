import { render, screen } from '@testing-library/react';
import { HelpCenter } from './HelpCenter';
import { CARD_IDS } from './HelpCenter.constants';
import { MemoryRouter } from 'react-router-dom';
import {
  TEST_USER_ADMIN,
  TEST_USER_EDITOR,
  TEST_USER_VIEWER,
} from 'utils/testStubs';
import { wrapWithQueryProvider, wrapWithUserProvider } from 'utils/testUtils';
import { User } from 'types/user.types';

const renderHelpCenter = (user?: User) =>
  render(
    wrapWithUserProvider(
      wrapWithQueryProvider(
        <MemoryRouter>
          <HelpCenter />
        </MemoryRouter>
      ),
      { user }
    )
  );

describe('HelpCenter', () => {
  it('should show pmm dump and pmm logs if user is admin', () => {
    renderHelpCenter(TEST_USER_ADMIN);

    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmDump}`)
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmLogs}`)
    ).toBeInTheDocument();
    expect(screen.queryAllByTestId(/^help-card-/).length).toEqual(7);
  });

  it('should not show pmm dump and pmm logs if user is viewer', () => {
    renderHelpCenter(TEST_USER_VIEWER);

    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmDump}`)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmLogs}`)
    ).not.toBeInTheDocument();
    expect(screen.queryAllByTestId(/^help-card-/).length).toEqual(5);
  });

  it('should not show pmm dump and pmm logs if user is editor', () => {
    renderHelpCenter(TEST_USER_EDITOR);

    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmDump}`)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByTestId(`help-card-${CARD_IDS.pmmLogs}`)
    ).not.toBeInTheDocument();
    expect(screen.queryAllByTestId(/^help-card-/).length).toEqual(5);
  });
});
