import { fireEvent, render, screen } from '@testing-library/react';
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

const mocks = vi.hoisted(() => ({
  startTour: vi.fn(),
}));

vi.mock('contexts/tour', async () => ({
  useTour: () => ({ startTour: mocks.startTour }),
}));

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
  beforeEach(() => {
    mocks.startTour.mockClear();
  });

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

  it('starts product tour when the corresponding card action is clicked', async () => {
    renderHelpCenter(TEST_USER_ADMIN);

    const startTourButton = screen.getByTestId(
      'tips-card-start-product-tour-button'
    );
    fireEvent.click(startTourButton);

    expect(mocks.startTour).toHaveBeenCalledWith('product');
  });
});
