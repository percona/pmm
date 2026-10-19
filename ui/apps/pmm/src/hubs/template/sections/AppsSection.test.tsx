import AppsRounded from '@mui/icons-material/AppsRounded';
import { screen, within } from '@testing-library/react';
import { renderHub } from '../../testUtils';
import { READY, testDoor } from '../hub.testFixtures';
import type { HubApp, HubSectionState } from '../hub.types';
import { AppsSection } from './AppsSection';

const OPERATIONS: HubApp = {
  id: 'operations',
  name: 'Operations for MongoDB',
  icon: AppsRounded,
  preview: 'developer',
  summary: {
    text: '11 of 15 services up · last scan 4m ago',
    doors: [
      testDoor('11 of 15 services up', '/operations?tab=services'),
      testDoor('last scan 4m ago', '/operations/automations?tab=scans'),
    ],
  },
  pages: [
    testDoor('Fleet', '/operations', { context: 'in Operations for MongoDB' }),
    testDoor('Nodes', '/operations/nodes'),
  ],
};

const BACKUPS: HubApp = {
  id: 'backups',
  name: 'MongoDB backups',
  icon: AppsRounded,
  summary: { text: 'Backup management is off' },
  pages: [testDoor('Inventory', '/graph/backup/inventory')],
};

const renderSection = (
  apps: HubApp[] = [OPERATIONS, BACKUPS],
  state: HubSectionState = READY
) =>
  renderHub(
    <AppsSection
      apps={apps}
      door={testDoor('All apps', '/prototype/all-apps')}
      state={state}
      onRetry={vi.fn()}
    />
  );

describe('AppsSection', () => {
  it('titles each card with an h3 and the Preview chip where it applies', () => {
    renderSection();

    const operations = screen.getByTestId('hub-app-operations');

    expect(
      within(operations).getByRole('heading', {
        level: 3,
        name: 'Operations for MongoDB',
      })
    ).toBeInTheDocument();
    expect(within(operations).getByTestId('preview-chip')).toBeInTheDocument();
    expect(
      within(screen.getByTestId('hub-app-backups')).queryByTestId(
        'preview-chip'
      )
    ).not.toBeInTheDocument();
  });

  it('turns the figures in a summary into doors', () => {
    renderSection();

    const operations = screen.getByTestId('hub-app-operations');

    expect(operations).toHaveTextContent(
      '11 of 15 services up · last scan 4m ago'
    );
    expect(
      within(operations).getByRole('link', { name: '11 of 15 services up' })
    ).toHaveAttribute('href', '/operations?tab=services');
    expect(
      within(operations).getByRole('link', { name: 'last scan 4m ago' })
    ).toHaveAttribute('href', '/operations/automations?tab=scans');
  });

  it('links into each page of the app', () => {
    renderSection();

    expect(
      screen.getByRole('link', { name: 'Fleet in Operations for MongoDB' })
    ).toHaveAttribute('href', '/operations');
    expect(screen.getByRole('link', { name: 'Inventory' })).toHaveAttribute(
      'href',
      '/graph/backup/inventory'
    );
  });

  it('keeps the cards when one source behind them fails', () => {
    renderSection([OPERATIONS, BACKUPS], {
      status: 'error',
      message: 'Could not read Operations.',
    });

    expect(screen.getByRole('alert')).toHaveTextContent(
      'Could not read Operations.'
    );
    expect(screen.getByTestId('hub-app-backups')).toBeInTheDocument();
  });
});
