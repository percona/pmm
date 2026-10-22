import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import PmmDumpRedirect from './PmmDumpRedirect';

const renderAt = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="graph/pmm-dump/*" element={<PmmDumpRedirect />} />
        <Route path="pmm-dump" element={<div>Dump list</div>} />
        <Route path="pmm-dump/new" element={<div>New dataset</div>} />
      </Routes>
    </MemoryRouter>
  );

describe('PmmDumpRedirect', () => {
  it('redirects the old Grafana list route', () => {
    renderAt('/graph/pmm-dump');

    expect(screen.getByText('Dump list')).toBeInTheDocument();
  });

  it('keeps the subpath', () => {
    renderAt('/graph/pmm-dump/new');

    expect(screen.getByText('New dataset')).toBeInTheDocument();
  });
});
