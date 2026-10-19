import { describe } from 'vitest';
import { PerconaMoIcon } from '@percona/peak-ui';
import { fireEvent, screen, within } from '@testing-library/react';
import { hasPeakHub, renderHub } from '../../testUtils';
import { testView } from '../hub.testFixtures';
import type { HubMessages, HubViewModel } from '../hub.types';
import { HubHeader } from './HubHeader';

const MESSAGES = {
  name: 'MongoDB',
  unit: { cluster: ['cluster', 'clusters'] },
} as HubMessages;

const renderHeader = (view: Partial<HubViewModel> = {}) => {
  const setParams = vi.fn();
  const refetch = vi.fn();

  renderHub(
    <HubHeader
      view={testView(view)}
      icon={PerconaMoIcon}
      messages={MESSAGES}
      setParams={setParams}
      updatedAt={Date.now() - 20_000}
      refetch={refetch}
      loading={false}
    />
  );

  return { setParams, refetch };
};

describe.skipIf(!hasPeakHub)('HubHeader', () => {
  it('is the page’s one h1, with the status line under it', () => {
    renderHeader();

    expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1);
    expect(
      screen.getByRole('heading', { level: 1, name: 'MongoDB' })
    ).toBeInTheDocument();
    expect(screen.getByTestId('hub-status-line')).toHaveTextContent(
      '3 clusters · 3 need attention'
    );
  });

  it('marks sample data so a screenshot cannot pass for an estate', () => {
    renderHeader({ scenario: 'healthy' });

    expect(screen.getByText('Sample data')).toBeInTheDocument();
  });

  it('shows no Sample data chip on live data', () => {
    renderHeader({ scenario: 'live' });

    expect(screen.queryByText('Sample data')).not.toBeInTheDocument();
  });

  it('offers Add and Install to whoever the view allows', () => {
    renderHeader();

    expect(
      screen.getByRole('link', { name: 'Add MongoDB service' })
    ).toHaveAttribute('href', '/graph/add-instance/mongodb');
    expect(
      screen.getByRole('link', { name: 'Install MongoDB Developer preview' })
    ).toHaveAttribute('href', '/operations/nodes?filter=unmonitored');
  });

  it('shows no actions to a viewer', () => {
    renderHeader({ role: 'viewer', actions: {} });

    expect(
      screen.queryByRole('link', { name: 'Add MongoDB service' })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('link', { name: /Install MongoDB/ })
    ).not.toBeInTheDocument();
  });

  it('filters by environment through the params', () => {
    const { setParams } = renderHeader();

    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Environment' }));
    fireEvent.click(
      within(screen.getByRole('listbox')).getByRole('option', { name: 'prod' })
    );

    expect(setParams).toHaveBeenCalledWith({ environment: 'prod' });
  });

  it('clears the environment with All environments', () => {
    const { setParams } = renderHeader({
      environment: { ...testView().environment, selected: 'prod' },
    });

    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Environment' }));
    fireEvent.click(screen.getByRole('option', { name: 'All environments' }));

    expect(setParams).toHaveBeenCalledWith({ environment: null });
  });

  it('hides the environment select with a single environment', () => {
    renderHeader({
      environment: {
        selected: null,
        options: [{ value: 'prod', label: 'prod' }],
      },
    });

    expect(
      screen.queryByRole('combobox', { name: 'Environment' })
    ).not.toBeInTheDocument();
  });

  it('says how fresh the data is and refreshes every source', () => {
    const { refetch } = renderHeader();

    expect(screen.getByText(/^Updated 2\ds ago$/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));

    expect(refetch).toHaveBeenCalledTimes(1);
  });
});
