import { fireEvent, render, screen } from '@testing-library/react';
import { Route, Routes, useLocation } from 'react-router-dom';
import { TEST_USER_ADMIN, TEST_USER_VIEWER } from 'utils/testStubs';
import { TestWrapper } from 'utils/testWrapper';
import { Inventory } from './Inventory';

const LocationProbe = () => (
  <span data-testid="location-probe">{useLocation().pathname}</span>
);

const renderInventory = (path: string, user = TEST_USER_ADMIN) =>
  render(
    <TestWrapper
      routerProps={{ initialEntries: [path] }}
      userContext={{ isLoading: false, user }}
    >
      <Routes>
        <Route path="/inventory" element={<Inventory />}>
          <Route path="services" element={<LocationProbe />} />
          <Route path="nodes" element={<LocationProbe />} />
        </Route>
      </Routes>
    </TestWrapper>
  );

describe('Inventory', () => {
  it('switches between the services and nodes tabs', () => {
    renderInventory('/inventory/services');

    expect(screen.getByTestId('inventory-tab-services')).toHaveAttribute(
      'aria-selected',
      'true'
    );

    fireEvent.click(screen.getByTestId('inventory-tab-nodes'));

    expect(screen.getByTestId('location-probe')).toHaveTextContent(
      '/inventory/nodes'
    );
    expect(screen.getByTestId('inventory-tab-nodes')).toHaveAttribute(
      'aria-selected',
      'true'
    );
  });

  it('is restricted to admins', () => {
    renderInventory('/inventory/services', TEST_USER_VIEWER);

    expect(screen.getByTestId('unauthorized')).toBeInTheDocument();
    expect(screen.queryByTestId('location-probe')).not.toBeInTheDocument();
  });
});
