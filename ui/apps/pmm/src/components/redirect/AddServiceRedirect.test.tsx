import { render, screen } from '@testing-library/react';
import { PMM_ADD_SERVICE_PATH } from 'lib/constants';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import AddServiceRedirect from './AddServiceRedirect';

const renderAt = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route
          path="graph/add-instance/:type?"
          element={<AddServiceRedirect />}
        />
        <Route path={PMM_ADD_SERVICE_PATH} element={<div>selection</div>} />
        <Route
          path={`${PMM_ADD_SERVICE_PATH}/:type`}
          element={<div>form</div>}
        />
      </Routes>
    </MemoryRouter>
  );

describe('AddServiceRedirect', () => {
  it('opens the native selection page', () => {
    renderAt('/graph/add-instance?orgId=1');

    expect(screen.getByText('selection')).toBeInTheDocument();
  });

  it('keeps the service type', () => {
    renderAt('/graph/add-instance/mysql');

    expect(screen.getByText('form')).toBeInTheDocument();
  });

  it('opens the selection page for an unknown type', () => {
    renderAt('/graph/add-instance/unknown');

    expect(screen.getByText('selection')).toBeInTheDocument();
  });
});
