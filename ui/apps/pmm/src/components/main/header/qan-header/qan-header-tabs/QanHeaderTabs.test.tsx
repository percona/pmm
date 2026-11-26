import { fireEvent, render, screen } from '@testing-library/react';
import { wrapWithRouter } from 'utils/testUtils';
import QanHeaderTabs from './QanHeaderTabs';
import { Route, Routes } from 'react-router-dom';
import { PMM_NEW_NAV_GRAFANA_PATH, PMM_NEW_NAV_PATH } from 'lib/constants';

const TabsTestComponent = () => (
  <>
    <QanHeaderTabs />
    <Routes>
      <Route path="/" element={<div>Home</div>} />
      <Route
        path={`${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-qan/pmm-query-analytics`}
        element={<div data-testid="historical-tab-content">Historical</div>}
      />
      <Route
        path={`${PMM_NEW_NAV_PATH}/rta`}
        element={<div data-testid="real-time-tab-content">Real-Time</div>}
      />
    </Routes>
  </>
);

describe('QanHeaderTabs', () => {
  it('should render', () => {
    render(wrapWithRouter(<TabsTestComponent />));

    expect(
      screen.getByTestId('qan-header-tabs-historical-tab')
    ).toBeInTheDocument();
    expect(
      screen.getByTestId('qan-header-tabs-real-time-tab')
    ).toBeInTheDocument();
  });

  it('should navigate to historical tab', async () => {
    render(wrapWithRouter(<TabsTestComponent />));

    fireEvent.click(screen.getByTestId('qan-header-tabs-historical-tab'));

    expect(screen.getByTestId('historical-tab-content')).toBeInTheDocument();
    expect(
      screen.queryByTestId('real-time-tab-content')
    ).not.toBeInTheDocument();
  });

  it('should navigate to real-time tab', async () => {
    render(wrapWithRouter(<TabsTestComponent />));

    fireEvent.click(screen.getByTestId('qan-header-tabs-real-time-tab'));

    expect(screen.getByTestId('real-time-tab-content')).toBeInTheDocument();
    expect(
      screen.queryByTestId('historical-tab-content')
    ).not.toBeInTheDocument();
  });
});
