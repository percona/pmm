import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import InventoryRedirect from './InventoryRedirect';

const LocationProbe = () => {
  const { pathname, search } = useLocation();

  return <span data-testid="location-probe">{pathname + search}</span>;
};

describe('InventoryRedirect', () => {
  it.each([
    ['/graph/inventory', '/inventory/services'],
    [
      '/graph/inventory/services?search-text-input=service-1&search-select=serviceId',
      '/inventory/services?services.f.serviceId=service-1',
    ],
    [
      '/graph/inventory/nodes?isPmmServerNode=true',
      '/inventory/nodes?nodes.f.isPmmServerNode=true',
    ],
    ['/graph/inventory/nodes/node-1/agents', '/inventory/nodes/node-1/agents'],
  ])('redirects %s to %s', (from, to) => {
    render(
      <MemoryRouter initialEntries={[from]}>
        <Routes>
          <Route path="/graph/inventory/*" element={<InventoryRedirect />} />
          <Route path="*" element={<LocationProbe />} />
        </Routes>
      </MemoryRouter>
    );

    expect(screen.getByTestId('location-probe')).toHaveTextContent(to);
  });
});
